package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	coreenum "logisfy/core/enum"
	helperconverter "logisfy/helper/converter"
	"logisfy/helper/crypto"
	helperexception "logisfy/helper/exception"
	helperprocess "logisfy/helper/process"
	"logisfy/internal/entity"
	modelrequest "logisfy/internal/model/request"
	modelresponse "logisfy/internal/model/response"
	"logisfy/internal/repository"
	"logisfy/internal/worker"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/xuri/excelize/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type AMRUseCase struct {
	DB                        *gorm.DB
	Log                       *slog.Logger
	Validate                  *validator.Validate
	Crypto                    crypto.Crypto
	Location                  *time.Location
	NumberBatch               int
	DeletedDurationInHour     int
	AMRRepository             *repository.AMRRepository
	AMRDetailRepository       *repository.AMRDetailRepository
	AMRConfigRepository       *repository.AMRConfigRepository
	AMRWeightConfigRepository *repository.AMRWeightConfigRepository
	AMRDetailResultRepository *repository.AMRDetailResultRepository
	UploadTempDir             string

	uploadQueue chan worker.UploadJob
}

func NewAMRUseCase(
	db *gorm.DB,
	log *slog.Logger,
	validate *validator.Validate,
	crypto crypto.Crypto,
	location *time.Location,
	numberBatch int,
	deletedDurationInHour int,
	amrRepository *repository.AMRRepository,
	amrDetailRepository *repository.AMRDetailRepository,
	amrConfigRepository *repository.AMRConfigRepository,
	amrWeightConfigRepository *repository.AMRWeightConfigRepository,
	amrDetailResultRepository *repository.AMRDetailResultRepository,
	maxConcurrentUploads int,
	uploadTempDir string,
) *AMRUseCase {
	uc := &AMRUseCase{
		DB:                        db,
		Log:                       log,
		Validate:                  validate,
		Crypto:                    crypto,
		Location:                  location,
		NumberBatch:               numberBatch,
		DeletedDurationInHour:     deletedDurationInHour,
		AMRRepository:             amrRepository,
		AMRDetailRepository:       amrDetailRepository,
		AMRConfigRepository:       amrConfigRepository,
		AMRWeightConfigRepository: amrWeightConfigRepository,
		AMRDetailResultRepository: amrDetailResultRepository,
		UploadTempDir:             uploadTempDir,

		uploadQueue: make(chan worker.UploadJob, maxConcurrentUploads*4),
	}

	uc.StartUploadWorkers(maxConcurrentUploads)
	return uc
}

var headerExpect = []string{
	"LOCATION_CODE",
	"LOCATION_TYPE",
	"TYPE_METER",
	"TARIFF",
	"POWER",
	"CURRENT_L1",
	"CURRENT_L2",
	"CURRENT_L3",
	"CURRENT_N",
	"VOLTAGE_L1",
	"VOLTAGE_L2",
	"VOLTAGE_L3",
	"ACTIVE_POWER_L1",
	"ACTIVE_POWER_L2",
	"ACTIVE_POWER_L3",
	"ACTIVE_POWER_TOTAL",
	"KWH_ABS_TOTAL",
	"CURRENT_ANGLE_L1",
	"CURRENT_ANGLE_L2",
	"CURRENT_ANGLE_L3",
	"VOLTAGE_ANGLE_L1",
	"VOLTAGE_ANGLE_L2",
	"VOLTAGE_ANGLE_L3",
	"APPARENT_POWER_L1",
	"APPARENT_POWER_L2",
	"APPARENT_POWER_L3",
	"BILL_REFF_KWH",
	"READ_DATE",
}

func (u *AMRUseCase) StartUploadWorkers(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			u.Log.Info("AMRUseCase.StartUploadWorkers()", "worker started", workerID)
			for job := range u.uploadQueue {
				u.processUploadJob(job)
			}
		}(i)
	}
}

func (u *AMRUseCase) Upload(ctx context.Context, req *modelrequest.UploadAMRReq) (resp modelresponse.UploadAMRResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "Upload()")
	defer span.End()

	var (
		err       error
		dataCount int64
	)

	if err = u.Validate.Struct(req); err != nil {
		u.Log.Warn("AMRUseCase.Upload()", "Validate.Struct()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.Upload()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	readTx := u.DB.WithContext(ctx)
	eligibleActiveStageProcesses := []coreenum.CTXEnumStageProcess{
		coreenum.CTXEnumStageProcessQueue,
		coreenum.CTXEnumStageProcessConfigSetting,
		coreenum.CTXEnumStageProcessWeightSetting,
		coreenum.CTXEnumStageProcessReady,
		coreenum.CTXEnumStageProcessDone,
		coreenum.CTXEnumStageProcessProcessing,
	}
	amrEntityExist, err := u.AMRRepository.FindByUserID(readTx, uint64(userID), eligibleActiveStageProcesses)
	if err != nil {
		u.Log.Info("AMRUseCase.Upload()", "AMRRepository().FindByUserID()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if len(amrEntityExist) > 0 {
		exc = helperexception.Conflict("data amr sebelumnya sudah ada, hapus terlebih dahulu !")
		return
	}

	if len(u.uploadQueue) >= cap(u.uploadQueue) {
		exc = helperexception.PermissionDenied("antrian upload penuh, coba beberapa saat lagi")
		return
	}

	tmpPath, err := u.saveTempFile(req.File, req.Filename)
	if err != nil {
		u.Log.Error("AMRUseCase.Upload()", "saveTempFile()", "err", err.Error())
		exc = helperexception.Internal("gagal untuk menyimpan data", err)
		return
	}

	amrEntity := (entity.AMREntity{}).Create(uint64(userID), u.Location, u.DeletedDurationInHour, req, 0, coreenum.CTXEnumStageProcessQueue)
	if err = u.AMRRepository.Create(readTx, amrEntity); err != nil {
		u.Log.Error("AMRUseCase.Upload()", "AMRRepository.Create()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk membuat entity AMR", err)
		return
	}

	bgCtx := context.WithValue(context.Background(), string(coreenum.CTXEnumIDUserID), strconv.Itoa(userID))
	select {
	case u.uploadQueue <- worker.UploadJob{
		UseCaseName:     coreenum.CTXEnumUseCaseAMR,
		UseCaseDetailID: amrEntity.AMRID,
		UserID:          uint64(userID),
		FilePath:        tmpPath,
		Ctx:             bgCtx,
	}:
	default:
		_ = os.Remove(tmpPath)
		_ = u.AMRRepository.Delete(readTx, amrEntity)
		exc = helperexception.PermissionDenied("antrian upload penuh, coba beberapa saat lagi")
		return
	}

	resp.DataCount = dataCount
	resp.AMRID = amrEntity.AMRID
	return
}

func (u *AMRUseCase) parseRow(row []string, rowNumber int64, mapIndex map[string]int, amrID uint64) (result *entity.AMRDetailEntity, err error) {
	var locationCodeEncrypt []byte

	locationCode := helperprocess.CellGuard(row, mapIndex["LOCATION_CODE"])
	locationType := helperprocess.CellGuard(row, mapIndex["LOCATION_TYPE"])
	typeMeter := helperprocess.CellGuard(row, mapIndex["TYPE_METER"])
	tariff := helperprocess.CellGuard(row, mapIndex["TARIFF"])
	power := helperprocess.CellGuard(row, mapIndex["POWER"])
	currentL1 := helperprocess.CellGuard(row, mapIndex["CURRENT_L1"])
	currentL2 := helperprocess.CellGuard(row, mapIndex["CURRENT_L2"])
	currentL3 := helperprocess.CellGuard(row, mapIndex["CURRENT_L3"])
	currentN := helperprocess.CellGuard(row, mapIndex["CURRENT_N"])
	voltageL1 := helperprocess.CellGuard(row, mapIndex["VOLTAGE_L1"])
	voltageL2 := helperprocess.CellGuard(row, mapIndex["VOLTAGE_L2"])
	voltageL3 := helperprocess.CellGuard(row, mapIndex["VOLTAGE_L3"])
	activePowerL1 := helperprocess.CellGuard(row, mapIndex["ACTIVE_POWER_L1"])
	activePowerL2 := helperprocess.CellGuard(row, mapIndex["ACTIVE_POWER_L2"])
	activePowerL3 := helperprocess.CellGuard(row, mapIndex["ACTIVE_POWER_L3"])
	activePowerT := helperprocess.CellGuard(row, mapIndex["ACTIVE_POWER_TOTAL"])
	kwhAbsTotal := helperprocess.CellGuard(row, mapIndex["KWH_ABS_TOTAL"])
	currentAngleL1 := helperprocess.CellGuard(row, mapIndex["CURRENT_ANGLE_L1"])
	currentAngleL2 := helperprocess.CellGuard(row, mapIndex["CURRENT_ANGLE_L2"])
	currentAngleL3 := helperprocess.CellGuard(row, mapIndex["CURRENT_ANGLE_L3"])
	voltageAngleL1 := helperprocess.CellGuard(row, mapIndex["VOLTAGE_ANGLE_L1"])
	voltageAngleL2 := helperprocess.CellGuard(row, mapIndex["VOLTAGE_ANGLE_L2"])
	voltageAngleL3 := helperprocess.CellGuard(row, mapIndex["VOLTAGE_ANGLE_L3"])
	apparentPowerL1 := helperprocess.CellGuard(row, mapIndex["APPARENT_POWER_L1"])
	apparentPowerL2 := helperprocess.CellGuard(row, mapIndex["APPARENT_POWER_L2"])
	apparentPowerL3 := helperprocess.CellGuard(row, mapIndex["APPARENT_POWER_L3"])
	billReffKwh := helperprocess.CellGuard(row, mapIndex["BILL_REFF_KWH"])
	readDate := helperprocess.CellGuard(row, mapIndex["READ_DATE"])

	requiredFields := map[string]string{
		"LOCATION_CODE":     locationCode,
		"LOCATION_TYPE":     locationType,
		"TYPE_METER":        typeMeter,
		"TARIFF":            tariff,
		"POWER":             power,
		"CURRENT_L1":        currentL1,
		"CURRENT_L2":        currentL2,
		"CURRENT_L3":        currentL3,
		"CURRENT_N":         currentN,
		"VOLTAGE_L1":        voltageL1,
		"VOLTAGE_L2":        voltageL2,
		"VOLTAGE_L3":        voltageL3,
		"ACTIVE_POWER_L1":   activePowerL1,
		"ACTIVE_POWER_L2":   activePowerL2,
		"ACTIVE_POWER_L3":   activePowerL3,
		"ACTIVE_POWER_T":    activePowerT,
		"KWH_ABS_TOTAL":     kwhAbsTotal,
		"CURRENT_ANGLE_L1":  currentAngleL1,
		"CURRENT_ANGLE_L2":  currentAngleL2,
		"CURRENT_ANGLE_L3":  currentAngleL3,
		"VOLTAGE_ANGLE_L1":  voltageAngleL1,
		"VOLTAGE_ANGLE_L2":  voltageAngleL2,
		"VOLTAGE_ANGLE_L3":  voltageAngleL3,
		"APPARENT_POWER_L1": apparentPowerL1,
		"APPARENT_POWER_L2": apparentPowerL2,
		"APPARENT_POWER_L3": apparentPowerL3,
		"BILL_REFF_KWH":     billReffKwh,
		"READ_DATE":         readDate,
	}

	for fieldName, value := range requiredFields {
		if strings.TrimSpace(value) == "" {
			err = helperprocess.AddEmptyRowErr(rowNumber, fieldName)
			return
		}
	}

	locationCodeEncrypt, err = u.Crypto.Encrypt(locationCode)
	if err != nil {
		return
	}

	locationTypeEnum := coreenum.CTXEnumLocationType(locationType)
	if !locationTypeEnum.IsValid() {
		err = helperprocess.AddUnsupportFieldErr(rowNumber, "location type", "Harus salah satu CUSTOMER, METERPOINT, atau PRE CUSTOMER.")
		return
	}

	powerVal, parseErr := helperprocess.ParseIntField(power, rowNumber, "power")
	if parseErr != nil {
		err = parseErr
		return
	}
	phase := 1
	if powerVal > 11000 || powerVal == 10600 || powerVal == 6600 {
		phase = 3
	}
	measurementType := coreenum.CTXEnumMeasurementTypeTakLangsung
	if powerVal < 53000 {
		measurementType = coreenum.CTXEnumMeasurementTypeLangsung
	}

	currentL1F, parseErr := helperprocess.ParseFloatField(currentL1, rowNumber, "currentL1")
	if parseErr != nil {
		err = parseErr
		return
	}
	currentL2F, parseErr := helperprocess.ParseFloatField(currentL2, rowNumber, "currentL2")
	if parseErr != nil {
		err = parseErr
		return
	}
	currentL3F, parseErr := helperprocess.ParseFloatField(currentL3, rowNumber, "currentL3")
	if parseErr != nil {
		err = parseErr
		return
	}
	currentMax := math.Max(math.Max(currentL1F, currentL2F), currentL3F)
	currentMin := math.Min(math.Min(currentL1F, currentL2F), currentL3F)

	currentNF, parseErr := helperprocess.ParseFloatField(currentN, rowNumber, "CURRENT_N")
	if parseErr != nil {
		err = parseErr
		return
	}

	voltageL1F, parseErr := helperprocess.ParseFloatField(voltageL1, rowNumber, "VOLTAGE_L1")
	if parseErr != nil {
		err = parseErr
		return
	}

	voltageL2F, parseErr := helperprocess.ParseFloatField(voltageL2, rowNumber, "VOLTAGE_L2")
	if parseErr != nil {
		err = parseErr
		return
	}

	voltageL3F, parseErr := helperprocess.ParseFloatField(voltageL3, rowNumber, "VOLTAGE_L3")
	if parseErr != nil {
		err = parseErr
		return
	}

	voltageMax := math.Max(math.Max(voltageL1F, voltageL2F), voltageL3F)
	voltageType := coreenum.CTXEnumVoltageTypeTM
	if voltageMax > 70 {
		voltageType = coreenum.CTXEnumVoltageTypeTR
	}

	activePowerL1F, parseErr := helperprocess.ParseFloatField(activePowerL1, rowNumber, "ACTIVE_POWER_L1")
	if parseErr != nil {
		err = parseErr
		return
	}

	activePowerL2F, parseErr := helperprocess.ParseFloatField(activePowerL2, rowNumber, "ACTIVE_POWER_L2")
	if parseErr != nil {
		err = parseErr
		return
	}

	activePowerL3F, parseErr := helperprocess.ParseFloatField(activePowerL3, rowNumber, "ACTIVE_POWER_L3")
	if parseErr != nil {
		err = parseErr
		return
	}

	activePowerTF, parseErr := helperprocess.ParseFloatField(activePowerT, rowNumber, "ACTIVE_POWER_TOTAL")
	if parseErr != nil {
		err = parseErr
		return
	}

	kwhAbsTotalF, parseErr := helperprocess.ParseFloatField(kwhAbsTotal, rowNumber, "KWH_ABS_TOTAL")
	if parseErr != nil {
		err = parseErr
		return
	}

	currentAngleL1F, parseErr := helperprocess.ParseFloatField(currentAngleL1, rowNumber, "CURRENT_ANGLE_L1")
	if parseErr != nil {
		err = parseErr
		return
	}

	currentAngleL2F, parseErr := helperprocess.ParseFloatField(currentAngleL2, rowNumber, "CURRENT_ANGLE_L2")
	if parseErr != nil {
		err = parseErr
		return
	}

	currentAngleL3F, parseErr := helperprocess.ParseFloatField(currentAngleL3, rowNumber, "CURRENT_ANGLE_L3")
	if parseErr != nil {
		err = parseErr
		return
	}

	voltageAngleL1F, parseErr := helperprocess.ParseFloatField(voltageAngleL1, rowNumber, "VOLTAGE_ANGLE_L1")
	if parseErr != nil {
		err = parseErr
		return
	}

	voltageAngleL2F, parseErr := helperprocess.ParseFloatField(voltageAngleL2, rowNumber, "VOLTAGE_ANGLE_L2")
	if parseErr != nil {
		err = parseErr
		return
	}

	voltageAngleL3F, parseErr := helperprocess.ParseFloatField(voltageAngleL3, rowNumber, "VOLTAGE_ANGLE_L3")
	if parseErr != nil {
		err = parseErr
		return
	}

	apparentPowerL1F, parseErr := helperprocess.ParseFloatField(apparentPowerL1, rowNumber, "APPARENT_POWER_L1")
	if parseErr != nil {
		err = parseErr
		return
	}

	apparentPowerL2F, parseErr := helperprocess.ParseFloatField(apparentPowerL2, rowNumber, "APPARENT_POWER_L2")
	if parseErr != nil {
		err = parseErr
		return
	}

	apparentPowerL3F, parseErr := helperprocess.ParseFloatField(apparentPowerL3, rowNumber, "APPARENT_POWER_L3")
	if parseErr != nil {
		err = parseErr
		return
	}

	billReffKwhVal, parseErr := helperprocess.ParseIntField(billReffKwh, rowNumber, "BILL_REFF_KWH")
	if parseErr != nil {
		err = parseErr
		return
	}
	readDateTime, parseErr := helperconverter.ConvertStringToTime(readDate)
	if parseErr != nil {
		err = helperprocess.AddUnsupportFieldErr(rowNumber, "READ_DATE", "Berformat \"1/2/2006 15:04\"")
		return
	}

	powerFactorL1 := helperprocess.CalculatePowerFactor(
		activePowerL1F,
		apparentPowerL1F,
		currentAngleL1F,
		voltageAngleL1F,
	)

	powerFactorL2 := helperprocess.CalculatePowerFactor(
		activePowerL2F,
		apparentPowerL2F,
		currentAngleL2F,
		voltageAngleL2F,
	)

	powerFactorL3 := helperprocess.CalculatePowerFactor(
		activePowerL3F,
		apparentPowerL3F,
		currentAngleL3F,
		voltageAngleL3F,
	)

	result = &entity.AMRDetailEntity{
		AMRID:               amrID,
		LocationCodeEncrypt: locationCodeEncrypt,
		LocationType:        locationTypeEnum,
		TypeMeter:           typeMeter,
		Tariff:              tariff,
		Power:               int64(powerVal),
		Phase:               phase,
		MeasurementType:     measurementType,
		CurrentL1:           currentL1F,
		CurrentL2:           currentL2F,
		CurrentL3:           currentL3F,
		CurrentMax:          currentMax,
		CurrentMin:          currentMin,
		CurrentN:            currentNF,
		VoltageL1:           voltageL1F,
		VoltageL2:           voltageL2F,
		VoltageL3:           voltageL3F,
		VoltageMax:          voltageMax,
		VoltageType:         voltageType,
		ActivePowerL1:       activePowerL1F,
		ActivePowerL2:       activePowerL2F,
		ActivePowerL3:       activePowerL3F,
		ActivePowerTotal:    activePowerTF,
		KWHAbsTotal:         kwhAbsTotalF,
		CurrentAngleL1:      currentAngleL1F,
		CurrentAngleL2:      currentAngleL2F,
		CurrentAngleL3:      currentAngleL3F,
		VoltageAngleL1:      voltageAngleL1F,
		VoltageAngleL2:      voltageAngleL2F,
		VoltageAngleL3:      voltageAngleL3F,
		ApparentPowerL1:     apparentPowerL1F,
		ApparentPowerL2:     apparentPowerL2F,
		ApparentPowerL3:     apparentPowerL3F,
		BillReffKwh:         billReffKwhVal,
		ReadDate:            readDateTime,
		PowerFactorL1:       powerFactorL1,
		PowerFactorL2:       powerFactorL2,
		PowerFactorL3:       powerFactorL3,
	}
	return
}

func (u *AMRUseCase) saveTempFile(src io.Reader, originalName string) (string, error) {
	ext := filepath.Ext(originalName)
	tempDir := u.UploadTempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	tmpFile, err := os.CreateTemp(tempDir, "amr-*"+ext)
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()
	// io.Copy streaming: hanya buffer kecil (32KB) di RAM setiap saat
	if _, err = io.Copy(tmpFile, src); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", err
	}
	return tmpFile.Name(), nil
}

func (u *AMRUseCase) processUploadJob(job worker.UploadJob) {
	defer os.Remove(job.FilePath)

	ctx, cancel := context.WithCancel(job.Ctx)
	defer cancel()

	log := u.Log.With("use_case_name", job.UseCaseName, "use_case_detail_id", job.UseCaseDetailID, "user_id", job.UserID)
	now := time.Now()

	readTx := u.DB.WithContext(ctx)
	entityAMR, err := u.AMRRepository.Find(readTx, job.UseCaseDetailID)
	if err != nil {
		u.Log.Error("AMRUseCase.processUploadJob()", "AMRRepository().Find()", "error", err.Error())
		return
	}
	if !entityAMR.CheckFound() {
		u.Log.Error("AMRUseCase.processUploadJob()", "AMRRepository().Find()", "error", "data tidak ditemukan")
		return
	}
	entityAMR.StageProcess = coreenum.CTXEnumStageProcessProcessing
	err = u.AMRRepository.Update(readTx, entityAMR)
	if err != nil {
		u.Log.Error("AMRUseCase.processUploadJob()", "AMRRepository().Update()", "error", err.Error())
		return
	}

	xlsx, err := excelize.OpenFile(job.FilePath)
	if err != nil {
		log.Error("AMRUseCase.processUploadJob()", "excelize.OpenFile()", "error", err.Error())
		u.markAMRFailed(readTx, entityAMR, "gagal membuka file excel")
		return
	}
	defer xlsx.Close()

	rows, mapIndex, err := helperprocess.ExtractAndValidateHeader(xlsx, headerExpect)
	if err != nil {
		log.Warn("AMRUseCase.processUploadJob()", "excelize.ExtractAndValidateHeader()", "error", err.Error())
		u.markAMRFailed(readTx, entityAMR, "format header tidak valid "+err.Error())
		return
	}
	const numParseWorkers = 2
	rawCh := make(chan worker.RowJob, numParseWorkers*2)
	doneCh := make(chan *entity.AMRDetailEntity, numParseWorkers*2)
	errCh := make(chan error, 1)

	go func() {
		defer close(rawCh)
		var rowNum int64
		for rows.Next() {
			rowNum++
			cols, e := rows.Columns()
			if e != nil {
				select {
				case errCh <- e:
				default:
				}
				return
			}
			select {
			case rawCh <- worker.RowJob{Cols: cols, RowNumber: rowNum + 1}:
			case <-ctx.Done():
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < numParseWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range rawCh {
				detail, e := u.parseRow(job.Cols, job.RowNumber, mapIndex, entityAMR.AMRID)
				if e != nil {
					select {
					case errCh <- e:
					default:
					}
					return
				}
				select {
				case doneCh <- detail:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() { wg.Wait(); close(doneCh) }()
	// Batcher: collect → CopyIn
	var (
		lastDetail *entity.AMRDetailEntity
		dataCount  int64
		batch      = make([]*entity.AMRDetailEntity, 0, u.NumberBatch)
	)

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		if e := u.AMRDetailRepository.CopyIn(ctx, batch, now); e != nil {
			return e
		}
		batch = batch[:0]
		return nil
	}
	var processingErr error

COLLECT:
	for {
		select {
		case detail, ok := <-doneCh:
			if !ok {
				break COLLECT
			}
			dataCount++
			lastDetail = detail
			batch = append(batch, detail)
			if len(batch) >= u.NumberBatch {
				if processingErr = flushBatch(); processingErr != nil {
					break COLLECT
				}
			}
		case e := <-errCh:
			processingErr = e
			break COLLECT
		}
	}
	// Cek error dari worker setelah done
	select {
	case e := <-errCh:
		processingErr = e
	default:
	}
	if processingErr != nil {
		log.Error("AMRUseCase.processUploadJob()", "pipeline error", processingErr.Error())
		u.markAMRFailed(readTx, entityAMR, processingErr.Error())
		for i := 0; i < 3; i++ {
			if err = u.AMRDetailRepository.DeleteByAMRID(readTx, entityAMR.AMRID); err == nil {
				break
			}
			time.Sleep(time.Duration(1<<i) * 500 * time.Millisecond)
		}
		return
	}
	if err = flushBatch(); err != nil {
		log.Error("AMRUseCase.processUploadJob()", "flushBatch()", "error", err.Error())
		u.markAMRFailed(readTx, entityAMR, "gagal menyimpan batch terakhir")
		return
	}

	if dataCount == 0 {
		log.Warn("AMRUseCase.processUploadJob()", "warning", "file tidak memiliki baris data")
		u.markAMRFailed(readTx, entityAMR, "file tidak memiliki baris data")
		return
	}
	entityAMR.ReadDate = lastDetail.ReadDate
	entityAMR.RowNumbers = dataCount
	entityAMR.StageProcess = coreenum.CTXEnumStageProcessConfigSetting
	if err = u.AMRRepository.Update(readTx, entityAMR); nil != err {
		log.Error("AMRUseCase.processUploadJob()", "AMRRepository().Update()", "error", err.Error())
		u.markAMRFailed(readTx, entityAMR, err.Error())
		return
	}
	log.Info("AMRUsecase.processUploadJob()", "status", "DONE", "rows", dataCount)
}

func (u *AMRUseCase) markAMRFailed(tx *gorm.DB, amrEntity *entity.AMREntity, reason string) {
	amrEntity.StageProcess = coreenum.CTXEnumStageProcessFailed
	amrEntity.FailedReason = reason
	if err := u.AMRRepository.Update(tx, amrEntity); err != nil {
		u.Log.Error("AMRUseCase.markAMRFailed()", "AMRRepository.Update()", "error", err.Error())
	}
}

func (u *AMRUseCase) CreateParamConfig(ctx context.Context, req *modelrequest.CreateParamConfigAMRReq) (resp modelresponse.CreateParamConfigAMRResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "CreateParamConfig()")
	defer span.End()

	var err error

	if err = u.Validate.Struct(req); err != nil {
		u.Log.Warn("AMRUseCase.CreateParamConfig()", "Validate.Struct()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Info("AMRUseCase.CreateParamConfig()", "strconv.Atoi()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Info("AMRUseCase.CreateParamConfig()", "AMRRepository().FindByAMRID()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.CreateParamConfig()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	if amrEntity.StageProcess != coreenum.CTXEnumStageProcessConfigSetting {
		exc = helperexception.Conflict("data amr belum siap digunakan, berada di stage " + amrEntity.StageProcess.ConvertToStr())
		return
	}

	tx = tx.Begin()
	defer func() {
		if exc != nil || err != nil {
			tx.Rollback()
		}
	}()

	amrConfigEntity := (entity.AMRConfigEntity{}).Create(req, amrEntity.AMRID)
	if err = u.AMRConfigRepository.Create(tx, amrConfigEntity); err != nil {
		u.Log.Info("AMRUseCase.CreateParamConfig", "AMRConfigRepository.Create()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk membuat data amr config", err)
		return
	}

	amrEntity.StageProcess = coreenum.CTXEnumStageProcessWeightSetting
	if err = u.AMRRepository.Update(tx, amrEntity); err != nil {
		u.Log.Info("AMRUseCase.CreateParamConfig()", "AMRRepository().Update()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mengupdate data amr", err)
		return
	}

	resp.AMRConfigID = amrConfigEntity.AMRConfigID

	if err = tx.Commit().Error; err != nil {
		u.Log.Error("AMRUseCase.CreateParamConfig()", "tx.Commit()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk membuat konfigurasi parameter amr", err)
		return
	}
	return
}

func (u *AMRUseCase) CreateWeightConfig(ctx context.Context, req *modelrequest.CreateWeightConfigAMRReq) (resp modelresponse.CreateWeightConfigAMRResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "CreateWeightConfig()")
	defer span.End()

	var err error
	if err = u.Validate.Struct(req); err != nil {
		u.Log.Warn("AMRUseCase.CreateWeightConfig()", "Validate.Struct()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Info("AMRUseCase.CreateWeightConfig()", "strconv.Atoi()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Info("AMRUseCase.CreateWeightConfig()", "AMRRepository().FindByAMRID()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.CreateWeightConfig()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	if amrEntity.StageProcess != coreenum.CTXEnumStageProcessWeightSetting {
		exc = helperexception.Conflict("data amr belum siap digunakan, berada di stage " + amrEntity.StageProcess.ConvertToStr())
		return
	}

	tx = tx.Begin()
	defer func() {
		if exc != nil || err != nil {
			tx.Rollback()
		}
	}()
	weightConfigEntity := (entity.AMRWeightConfigEntity{}).Create(req, amrEntity.AMRID)
	if err = u.AMRWeightConfigRepository.Create(tx, weightConfigEntity); err != nil {
		u.Log.Error("AMRUseCase.CreateWeightConfig()", "AMRWeightConfigRepository.Create()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk membuat konfigurasi bobot", err)
		return
	}

	amrEntity.StageProcess = coreenum.CTXEnumStageProcessReady
	if err = u.AMRRepository.Update(tx, amrEntity); err != nil {
		u.Log.Error("AMRUseCase.CreateWeightConfig()", "AMRRepository.Update()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk update status amr", err)
		return
	}
	resp.AMRWeightConfigID = weightConfigEntity.AMRWeightConfigID

	if err = tx.Commit().Error; err != nil {
		u.Log.Error("AMRUseCase.CreateWeightConfig()", "tx.Commit()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk membuat konfigurasi bobot", err)
		return
	}

	return
}

func (u *AMRUseCase) List(ctx context.Context, req *modelrequest.ListAMRReq) (resp modelresponse.ListAMRResp, exc *helperexception.Exception) {
	// init tracer
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "List()")
	defer span.End()

	// Init DB
	tx := u.DB.WithContext(ctx)

	// exec repo
	entityAMRList, items, totalPages, size, err := u.AMRRepository.List(tx, req.QueryInfo)
	if err != nil {
		u.Log.Info("AMRUseCase.List()", "AMRRepository.List()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses list amr", err)
		return
	}

	// resp
	resp.List = (&entity.AMREntity{}).ConvertToList(entityAMRList)
	resp.TotalItems = items
	resp.Page = req.QueryInfo.SelectParameter.PageDescriptor.PageIndex
	resp.PageSize = size
	resp.TotalPages = totalPages
	return
}

func (u *AMRUseCase) GenerateReport(ctx context.Context, req *modelrequest.GenerateReportAMRReq) (resp modelresponse.GenerateReportAMRResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "GenerateReport()")
	defer span.End()

	var (
		err                error
		amrEntity          *entity.AMREntity
		paramConfigEntity  *entity.AMRConfigEntity
		amrDetailEntities  []*entity.AMRDetailEntity
		weightConfigEntity *entity.AMRWeightConfigEntity
		lastAMRDetailID    uint64
	)
	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Info("AMRUseCase.GenerateReport()", "strconv.Atoi()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	readTx := u.DB.WithContext(ctx)
	amrEntity, err = u.AMRRepository.Find(readTx, uint64(amrID))
	if err != nil {
		u.Log.Info("AMRUseCase.GenerateReport()", "AMRRepository().FindByAMRID()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.GenerateReport()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	if amrEntity.StageProcess != coreenum.CTXEnumStageProcessReady {
		exc = helperexception.Conflict("data amr belum siap digunakan, berada di stage " + amrEntity.StageProcess.ConvertToStr())
		return
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		paramConfigEntityGo, errGo := u.AMRConfigRepository.FindByAMRID(u.DB.WithContext(gCtx), amrEntity.AMRID)
		if errGo != nil {
			u.Log.Info("AMRUseCase.GenerateReport()", "AMRConfigRepository().FindByAMRID()", "error", errGo.Error())
			return errGo
		}
		if !paramConfigEntityGo.CheckFound() {
			return errors.New("data konfigurasi parameter tidak ditemukan")
		}
		paramConfigEntity = paramConfigEntityGo
		return nil
	})
	g.Go(func() error {
		weightConfigEntityGo, errGo := u.AMRWeightConfigRepository.FindByAMRID(u.DB.WithContext(gCtx), amrEntity.AMRID)
		if errGo != nil {
			u.Log.Info("AMRUseCase.GenerateReport()", "AMRWeightConfigRepository().FindByAMRID()", "error", errGo.Error())
			return errGo
		}
		if !weightConfigEntityGo.CheckFound() {
			return errors.New("data konfigurasi bobot tidak ditemukan")
		}
		weightConfigEntity = weightConfigEntityGo
		return nil
	})
	if err = g.Wait(); err != nil {
		exc = helperexception.Internal(err.Error(), err)
		return
	}

	tx := readTx.Begin()
	defer func() {
		if exc != nil || err != nil {
			tx.Rollback()
		}
	}()

	for {
		amrDetailEntities, err = u.AMRDetailRepository.FindBatchByAMRID(readTx, amrEntity.AMRID, lastAMRDetailID, u.NumberBatch)
		if err != nil {
			u.Log.Info("AMRUseCase.GenerateReport()", "AMRDetailRepository().FindByAMRID()", "error", err.Error())
			exc = helperexception.Internal("gagal untuk mencari data detail amr", err)
			return
		}
		if len(amrDetailEntities) < 1 {
			break
		}
		lastData := len(amrDetailEntities) - 1
		lastAMRDetailID = amrDetailEntities[lastData].AMRDetailID

		amrDetailResults := (&entity.AMRDetailResultEntity{}).Create(amrDetailEntities, paramConfigEntity, weightConfigEntity)
		if err = u.AMRDetailResultRepository.CreateBatch(tx, amrDetailResults); err != nil {
			u.Log.Error("AMRUseCase.GenerateReport()", "AMRDetailResultRepository.CreateBatch()", "error", err.Error())
			exc = helperexception.Internal("gagal untuk generasi hasil amr", err)
			return
		}
	}

	amrEntity.StageProcess = coreenum.CTXEnumStageProcessDone
	if err = u.AMRRepository.Update(tx, amrEntity); err != nil {
		u.Log.Error("AMRUseCase.GenerateReport()", "AMRRepository.Update()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk update status amr", err)
		return
	}

	resp.AMRID = amrEntity.AMRID

	if err = tx.Commit().Error; err != nil {
		u.Log.Error("AMRUseCase.GenerateReport()", "tx.Commit()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk membuat report", err)
		return
	}

	return
}

func (u *AMRUseCase) Report(ctx context.Context, req *modelrequest.ListReportAMRReq) (resp modelresponse.ListReportAMRResp, exc *helperexception.Exception) {
	// init tracer
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "Report()")
	defer span.End()

	var err error

	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Warn("AMRUseCase.Report()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.Report()", "AMRRepository.Find()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.Report()", "strconv.Atoi() userID", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}

	if amrEntity.StageProcess != coreenum.CTXEnumStageProcessDone {
		exc = helperexception.Conflict("laporan belum tersedia, data amr berada di stage " + amrEntity.StageProcess.ConvertToStr())
		return
	}

	entityReport, items, totalPages, size, err := u.AMRDetailResultRepository.Report(tx, uint64(amrID), req.QueryInfo)
	if err != nil {
		u.Log.Warn("AMRUseCase.Report()", "AMRDetailResultRepository.Report()", "warn", err.Error())
		exc = helperexception.Internal("gagal saat proses list detail amr", err)
		return
	}

	resp.List = (&entity.AMRDetailResultEntity{}).ConvertToReport(entityReport, u.Crypto)
	resp.TotalItems = items
	resp.Page = req.QueryInfo.SelectParameter.PageDescriptor.PageIndex
	resp.PageSize = size
	resp.TotalPages = totalPages
	return
}

func (u *AMRUseCase) FindAMR(ctx context.Context, req *modelrequest.FindAMRReq) (resp modelresponse.FindAMRResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "FindAMR()")
	defer span.End()

	var err error

	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMR()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMR()", "AMRRepository.Find()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMR()", "strconv.Atoi() userID", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}

	readDateStr := helperconverter.ConvertTimeToString(&amrEntity.ReadDate)
	createdAtStr := helperconverter.ConvertTimeToString(&amrEntity.CreatedAt)
	autoDeletedAt := helperconverter.ConvertTimeToString(&amrEntity.AutoDeletedAt)

	resp.AMRID = amrEntity.AMRID
	resp.Filename = amrEntity.Filename
	resp.Extension = amrEntity.Extension
	resp.RowNumbers = amrEntity.RowNumbers
	resp.ReadDate = readDateStr
	resp.StageProcess = amrEntity.StageProcess
	resp.CreatedAt = createdAtStr
	resp.AutoDeletedAt = autoDeletedAt
	return
}

func (u *AMRUseCase) FindAMRParamConfig(ctx context.Context, req *modelrequest.FindAMRParamConfigReq) (resp modelresponse.FindAMRParamConfigResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "FindAMRParamConfig()")
	defer span.End()

	var err error

	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMRParamConfig()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMRParamConfig()", "AMRRepository.Find()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMRParamConfig()", "strconv.Atoi() userID", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}

	paramConfigEntity, err := u.AMRConfigRepository.FindByAMRID(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMRParamConfig()", "AMRConfigRepository.FindByAMRID()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data konfigurasi parameter AMR", err)
		return
	}
	if !paramConfigEntity.CheckFound() {
		exc = helperexception.NotFound("data konfigurasi parameter AMR tidak ditemukan")
		return
	}

	resp = (entity.AMRConfigEntity{}).ConvertToResp(paramConfigEntity)

	return
}

func (u *AMRUseCase) FindAMRWeightConfig(ctx context.Context, req *modelrequest.FindAMRWeightConfigReq) (resp modelresponse.FindAMRWeightConfigResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "FindAMRWeightConfig()")
	defer span.End()

	var err error

	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMRWeightConfig()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMRWeightConfig()", "AMRRepository.Find()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMRWeightConfig()", "strconv.Atoi() userID", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}

	weightConfigEntity, err := u.AMRWeightConfigRepository.FindByAMRID(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMRWeightConfig()", "AMRWeightConfigRepository.FindByAMRID()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data konfigurasi bobot AMR", err)
		return
	}
	if !weightConfigEntity.CheckFound() {
		exc = helperexception.NotFound("data konfigurasi bobot AMR tidak ditemukan")
		return
	}

	resp = (entity.AMRWeightConfigEntity{}).ConvertToResp(weightConfigEntity)

	return
}

func (u *AMRUseCase) Delete(ctx context.Context, req *modelrequest.DeleteAMRReq) (resp modelresponse.DeleteAMRResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "Delete()")
	defer span.End()

	var err error

	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Warn("AMRUseCase.Delete()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.Delete()", "AMRRepository.Find()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.Delete()", "strconv.Atoi() userID", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}

	if err = u.AMRRepository.Delete(tx, amrEntity); err != nil {
		u.Log.Error("AMRUseCase.Delete()", "AMRRepository.Delete()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk menghapus amr", err)
		return
	}

	resp.AMRID = amrEntity.AMRID
	return
}

func (u *AMRUseCase) Summary(ctx context.Context, req *modelrequest.SummaryAMRReq) (resp modelresponse.SummaryAMRResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "Summary()")
	defer span.End()

	var err error

	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Warn("AMRUseCase.Summary()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.Summary()", "AMRRepository.Find()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.FindAMR()", "strconv.Atoi() userID", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}

	paramConfigEntity, err := u.AMRConfigRepository.FindByAMRID(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.Summary()", "AMRConfigRepository.FindByAMRID()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data konfigurasi parameter AMR", err)
		return
	}
	if !paramConfigEntity.CheckFound() {
		exc = helperexception.NotFound("data konfigurasi parameter AMR tidak ditemukan")
		return
	}

	weightConfigEntity, err := u.AMRWeightConfigRepository.FindByAMRID(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.Summary()", "AMRWeightConfigRepository.FindByAMRID()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data konfigurasi bobot AMR", err)
		return
	}
	if !weightConfigEntity.CheckFound() {
		exc = helperexception.NotFound("data konfigurasi bobot AMR tidak ditemukan")
		return
	}

	summaryAMRDetailResultEntity, err := u.AMRDetailResultRepository.Summary(tx, uint64(amrID))
	fmt.Println(summaryAMRDetailResultEntity)
	if err != nil {
		u.Log.Warn("AMRUseCase.Summary()", "AMRDetailResultRepository.Summary()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk membuat ringkasan hasil AMR detail", err)
		return
	}

	readDateStr := helperconverter.ConvertTimeToString(&amrEntity.ReadDate)
	createdAtStr := helperconverter.ConvertTimeToString(&amrEntity.CreatedAt)
	autoDeletedAt := helperconverter.ConvertTimeToString(&amrEntity.AutoDeletedAt)

	resp.AMRID = amrEntity.AMRID
	resp.Filename = amrEntity.Filename
	resp.ReadDate = readDateStr
	resp.CreatedAt = createdAtStr
	resp.AutoDeletedAt = autoDeletedAt
	resp.RowNumbers = amrEntity.RowNumbers
	resp.TotalVDrop = summaryAMRDetailResultEntity.TotalVDrop
	resp.TotalVLoss = summaryAMRDetailResultEntity.TotalVLoss
	resp.TotalCosPhiKecil = summaryAMRDetailResultEntity.TotalCosPhiKecil
	resp.TotalILoss = summaryAMRDetailResultEntity.TotalILoss
	resp.TotalVLoss = summaryAMRDetailResultEntity.TotalVLoss
	resp.TotalOverI = summaryAMRDetailResultEntity.TotalOverI
	resp.TotalOverV = summaryAMRDetailResultEntity.TotalOverV
	resp.TotalUnbalanceI = summaryAMRDetailResultEntity.TotalUnbalanceI
	resp.TotalILowVLow = summaryAMRDetailResultEntity.TotalILowVLow
	resp.TotalCurrentLoop = summaryAMRDetailResultEntity.TotalCurrentLoop
	resp.TotalActivePLoss = summaryAMRDetailResultEntity.TotalActivePLoss
	resp.TotalFreeze = summaryAMRDetailResultEntity.TotalFreeze
	resp.TotalInGreaterIMax = summaryAMRDetailResultEntity.TotalInGreaterIMax
	resp.TotalReversePower = summaryAMRDetailResultEntity.TotalReversePower
	resp.AMRParamConfig = (entity.AMRConfigEntity{}).ConvertToResp(paramConfigEntity)
	resp.AMRWeightConfig = (entity.AMRWeightConfigEntity{}).ConvertToResp(weightConfigEntity)

	return
}

func (u *AMRUseCase) DownloadTemplate(ctx context.Context) (resp modelresponse.DownloadAMRTemplateResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "DownloadTemplate()")
	defer span.End()

	const templatePath = "file/template-amr.xlsx"

	resp.FileName = "template-amr.xlsx"
	resp.ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

	bytes, err := os.ReadFile(templatePath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		u.Log.Error("AMRUseCase.DownloadTemplate()", "os.ReadFile()", "error", err.Error())
		exc = helperexception.Internal("gagal membaca file template", err)
		return
	}
	resp.XLSXBytes = bytes

	return
}

func (u *AMRUseCase) Export(ctx context.Context, req *modelrequest.ExportAMRReq) (resp modelresponse.ExportAMRResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "Export()")
	defer span.End()

	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Warn("AMRUseCase.Export() failed parsing amrID", "method", "strconv.Atoi()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)

	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.Export() failed finding amr", "method", "AMRRepository.Find()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.Export() failed parsing userID", "method", "strconv.Atoi() userID", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	if amrEntity.StageProcess != coreenum.CTXEnumStageProcessDone {
		exc = helperexception.Conflict("laporan belum tersedia, data amr berada di stage " + amrEntity.StageProcess.ConvertToStr())
		return
	}

	fileExcel := excelize.NewFile()
	defer func() {
		if err := fileExcel.Close(); err != nil {
			u.Log.Warn("AMRUseCase.Export() failed closing file", "method", "fileExcel.Close()", "error", err.Error())
		}
	}()

	detailSheetName := "Detail"
	index, err := fileExcel.NewSheet(detailSheetName)
	if err != nil {
		u.Log.Error("AMRUseCase.Export()", "method", "fileExcel.NewSheet()", "error", err.Error())
		exc = helperexception.Internal("gagal membuat sheet baru", err)
		return
	}
	_ = fileExcel.DeleteSheet("Sheet1")

	titleStyle, alignCenter := helperprocess.Style(fileExcel)

	fileExcel.SetActiveSheet(index)
	_ = fileExcel.MergeCell(detailSheetName, "A1", "AU1")
	_ = fileExcel.SetCellValue(detailSheetName, "A1", "DETAIL AUTOGEN REPORT")
	_ = fileExcel.SetCellStyle(detailSheetName, "A1", "A1", titleStyle)
	columns := []struct {
		Col    string
		Header string
		Width  float64
	}{
		{"A", "LOCATION_CODE", 20},
		{"B", "TYPE_METER", 15},
		{"C", "TARIFF", 15},
		{"D", "POWER", 15},
		{"E", "LOCATION_TYPE", 15},
		{"F", "READ_DATE", 15},
		{"G", "VOLTAGE_L1", 15},
		{"H", "VOLTAGE_L2", 15},
		{"I", "VOLTAGE_L3", 15},
		{"J", "VOLTAGE_TYPE", 15},
		{"K", "CURRENT_L1", 15},
		{"L", "CURRENT_L2", 15},
		{"M", "CURRENT_L3", 15},
		{"N", "CURRENT_N", 15},
		{"O", "VOLTAGE_ANGLE_L1", 15},
		{"P", "VOLTAGE_ANGLE_L2", 15},
		{"Q", "VOLTAGE_ANGLE_L3", 15},
		{"R", "CURRENT_ANGLE_L1", 15},
		{"S", "CURRENT_ANGLE_L2", 15},
		{"T", "CURRENT_ANGLE_L3", 15},
		{"U", "POWER_FACTOR_L1", 15},
		{"V", "POWER_FACTOR_L2", 15},
		{"W", "POWER_FACTOR_L3", 15},
		{"X", "ACTIVE_POWER_L1", 15},
		{"Y", "ACTIVE_POWER_L2", 15},
		{"Z", "ACTIVE_POWER_L3", 15},
		{"AA", "APPARENT_POWER_L1", 15},
		{"AB", "APPARENT_POWER_L2", 15},
		{"AC", "APPARENT_POWER_L3", 15},
		{"AD", "KWH_ABS_TOTAL", 15},
		{"AE", "BILL_REFF_KWH", 15},
		{"AF", "PHASE", 15},
		{"AG", "MEASUREMENT_TYPE", 15},
		{"AH", "V_DROP", 10},
		{"AI", "V_LOSS", 10},
		{"AJ", "COS_PHI_KECIL", 10},
		{"AK", "I_LOSS", 10},
		{"AL", "IN_GREATER_I_MAX", 10},
		{"AM", "OVER_I", 10},
		{"AN", "OVER_V", 10},
		{"AO", "REVERSE_POWER", 10},
		{"AP", "UNBALANCE_I", 10},
		{"AQ", "I_LOW_V_LOW", 10},
		{"AR", "CURRENT_LOOP", 10},
		{"AS", "ACTIVE_P_LOSS", 10},
		{"AT", "FREEZE", 10},
		{"AU", "TOTAL_WEIGHTED_VALUE", 20},
	}
	for _, col := range columns {
		cell := col.Col + "2"
		_ = fileExcel.SetCellValue(detailSheetName, cell, col.Header)
		_ = fileExcel.SetColWidth(detailSheetName, col.Col, col.Col, col.Width)
		_ = fileExcel.SetCellStyle(detailSheetName, cell, cell, alignCenter)
	}

	page := int32(1)
	rowOffset := 3
	batchSize := int32(u.NumberBatch)

	for {
		req.QueryInfo.SelectParameter.PageDescriptor.PageIndex = page
		req.QueryInfo.SelectParameter.PageDescriptor.PageSize = batchSize

		reportEntities, _, totalPages, _, err := u.AMRDetailResultRepository.Report(tx, uint64(amrID), req.QueryInfo)
		if err != nil {
			u.Log.Warn("AMRUseCase.Export() failed listing report entities", "method", "AMRDetailResultRepository.Report()", "page", page, "error", err.Error())
			exc = helperexception.Internal("gagal saat proses list detail amr", err)
			return
		}

		if len(reportEntities) == 0 {
			break
		}

		for i, entity := range reportEntities {
			locationCodeDecrypt, err := u.Crypto.Decrypt(entity.LocationCodeEncrypt)
			if err != nil {
				u.Log.Error("AMRUseCase.Export() failed decrypting location code", "method", "Crypto.Decrypt()", "error", err.Error())
				exc = helperexception.Internal("gagal mendeskripsi location code", err)
				return
			}
			readDateStr := helperconverter.ConvertTimeToString(&entity.ReadDate)

			rowVals := []interface{}{
				locationCodeDecrypt,
				entity.TypeMeter,
				entity.Tariff,
				entity.Power,
				entity.LocationType.String(),
				readDateStr,
				entity.VoltageL1,
				entity.VoltageL2,
				entity.VoltageL3,
				entity.VoltageType.String(),
				entity.CurrentL1,
				entity.CurrentL2,
				entity.CurrentL3,
				entity.CurrentN,
				entity.VoltageAngleL1,
				entity.VoltageAngleL2,
				entity.VoltageAngleL3,
				entity.CurrentAngleL1,
				entity.CurrentAngleL2,
				entity.CurrentAngleL3,
				entity.PowerFactorL1,
				entity.PowerFactorL2,
				entity.PowerFactorL3,
				entity.ActivePowerL1,
				entity.ActivePowerL2,
				entity.ActivePowerL3,
				entity.ApparentPowerL1,
				entity.ApparentPowerL2,
				entity.ApparentPowerL3,
				entity.KWHAbsTotal,
				entity.BillReffKwh,
				entity.Phase,
				entity.MeasurementType.String(),
				entity.VDrop,
				entity.VLoss,
				entity.CosPhiKecil,
				entity.ILoss,
				entity.InGreaterIMax,
				entity.OverI,
				entity.OverV,
				entity.ReversePower,
				entity.UnbalanceI,
				entity.ILowVLow,
				entity.CurrentLoop,
				entity.ActivePLoss,
				entity.Freeze,
				entity.TotalWeightedValue,
			}

			currentRow := rowOffset + int(i)
			err = fileExcel.SetSheetRow(detailSheetName, "A"+strconv.Itoa(currentRow), &rowVals)
			if err != nil {
				u.Log.Error("AMRUseCase.Export()", "method", "fileExcel.SetSheetRow()", "row", currentRow, "error", err.Error())
				exc = helperexception.Internal("gagal menulis data ke sheet", err)
				return
			}
		}

		if page >= totalPages {
			break
		}
		rowOffset += len(reportEntities)
		page++
	}

	buffer, err := fileExcel.WriteToBuffer()
	if err != nil {
		u.Log.Error("AMRUseCase.Export() failed writing buffer", "method", "fileExcel.WriteToBuffer()", "error", err.Error())
		exc = helperexception.Internal("gagal menulis file excel ke buffer", err)
		return
	}

	fileName := amrEntity.Filename
	if !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
		ext := filepath.Ext(fileName)
		fileName = strings.TrimSuffix(fileName, ext) + ".xlsx"
	}

	resp.FileName = "autogen_amr_" + fileName
	resp.ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	resp.XLSXBytes = buffer.Bytes()
	return
}

func (u *AMRUseCase) ExportRecommendation(ctx context.Context, req *modelrequest.ExportRecommendationAMRReq) (resp modelresponse.ExportRecommendationAMRResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "Export()")
	defer span.End()

	amrID, err := strconv.Atoi(req.AMRID)
	if err != nil {
		u.Log.Warn("AMRUseCase.ExportRecommendation()", "method", "strconv.Atoi()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)

	amrEntity, err := u.AMRRepository.Find(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.ExportRecommendation()", "method", "AMRRepository.Find()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if !amrEntity.CheckFound() {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AMRUseCase.ExportRecommendation()", "method", "strconv.Atoi() userID", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	if amrEntity.UserID != uint64(userID) {
		exc = helperexception.NotFound("data amr tidak ditemukan")
		return
	}
	if amrEntity.StageProcess != coreenum.CTXEnumStageProcessDone {
		exc = helperexception.Conflict("laporan belum tersedia, data amr berada di stage " + amrEntity.StageProcess.ConvertToStr())
		return
	}

	paramConfigEntity, err := u.AMRConfigRepository.FindByAMRID(tx, uint64(amrID))
	if err != nil {
		u.Log.Warn("AMRUseCase.ExportRecommendation()", "AMRConfigRepository.FindByAMRID()", "warn", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data konfigurasi parameter AMR", err)
		return
	}
	if !paramConfigEntity.CheckFound() {
		exc = helperexception.NotFound("data konfigurasi parameter AMR tidak ditemukan")
		return
	}

	fileExcel := excelize.NewFile()
	defer func() {
		if err := fileExcel.Close(); err != nil {
			u.Log.Warn("AMRUseCase.Export() failed closing file", "method", "fileExcel.Close()", "error", err.Error())
		}
	}()

	detailSheetName := "Detail"
	index, err := fileExcel.NewSheet(detailSheetName)
	if err != nil {
		u.Log.Error("AMRUseCase.Export() failed creating sheet", "method", "fileExcel.NewSheet()", "error", err.Error())
		exc = helperexception.Internal("gagal membuat sheet baru", err)
		return
	}
	_ = fileExcel.DeleteSheet("Sheet1")

	titleStyle, alignCenter := helperprocess.Style(fileExcel)

	fileExcel.SetActiveSheet(index)
	_ = fileExcel.MergeCell(detailSheetName, "A1", "AU1")
	_ = fileExcel.SetCellValue(detailSheetName, "A1", "DETAIL AUTOGEN REKOMENDASI TOP "+strconv.Itoa(paramConfigEntity.NShowRecommendation))
	_ = fileExcel.SetCellStyle(detailSheetName, "A1", "A1", titleStyle)
	columns := []struct {
		Col    string
		Header string
		Width  float64
	}{
		{"A", "LOCATION_CODE", 20},
		{"B", "TYPE_METER", 15},
		{"C", "TARIFF", 15},
		{"D", "POWER", 15},
		{"E", "LOCATION_TYPE", 15},
		{"F", "READ_DATE", 15},
		{"G", "VOLTAGE_L1", 15},
		{"H", "VOLTAGE_L2", 15},
		{"I", "VOLTAGE_L3", 15},
		{"J", "VOLTAGE_TYPE", 15},
		{"K", "CURRENT_L1", 15},
		{"L", "CURRENT_L2", 15},
		{"M", "CURRENT_L3", 15},
		{"N", "CURRENT_N", 15},
		{"O", "VOLTAGE_ANGLE_L1", 15},
		{"P", "VOLTAGE_ANGLE_L2", 15},
		{"Q", "VOLTAGE_ANGLE_L3", 15},
		{"R", "CURRENT_ANGLE_L1", 15},
		{"S", "CURRENT_ANGLE_L2", 15},
		{"T", "CURRENT_ANGLE_L3", 15},
		{"U", "POWER_FACTOR_L1", 15},
		{"V", "POWER_FACTOR_L2", 15},
		{"W", "POWER_FACTOR_L3", 15},
		{"X", "ACTIVE_POWER_L1", 15},
		{"Y", "ACTIVE_POWER_L2", 15},
		{"Z", "ACTIVE_POWER_L3", 15},
		{"AA", "APPARENT_POWER_L1", 15},
		{"AB", "APPARENT_POWER_L2", 15},
		{"AC", "APPARENT_POWER_L3", 15},
		{"AD", "KWH_ABS_TOTAL", 15},
		{"AE", "BILL_REFF_KWH", 15},
		{"AF", "PHASE", 15},
		{"AG", "MEASUREMENT_TYPE", 15},
		{"AH", "V_DROP", 10},
		{"AI", "V_LOSS", 10},
		{"AJ", "COS_PHI_KECIL", 10},
		{"AK", "I_LOSS", 10},
		{"AL", "IN_GREATER_I_MAX", 10},
		{"AM", "OVER_I", 10},
		{"AN", "OVER_V", 10},
		{"AO", "REVERSE_POWER", 10},
		{"AP", "UNBALANCE_I", 10},
		{"AQ", "I_LOW_V_LOW", 10},
		{"AR", "CURRENT_LOOP", 10},
		{"AS", "ACTIVE_P_LOSS", 10},
		{"AT", "FREEZE", 10},
		{"AU", "TOTAL_WEIGHTED_VALUE", 20},
	}
	for _, col := range columns {
		cell := col.Col + "2"
		_ = fileExcel.SetCellValue(detailSheetName, cell, col.Header)
		_ = fileExcel.SetColWidth(detailSheetName, col.Col, col.Col, col.Width)
		_ = fileExcel.SetCellStyle(detailSheetName, cell, cell, alignCenter)
	}

	req.QueryInfo.SelectParameter.PageDescriptor.PageIndex = 1
	req.QueryInfo.SelectParameter.PageDescriptor.PageSize = int32(paramConfigEntity.NShowRecommendation)
	rowOffset := 3

	reportEntities, _, _, _, err := u.AMRDetailResultRepository.Report(tx, uint64(amrID), req.QueryInfo)
	if err != nil {
		u.Log.Warn("AMRUseCase.ExportRecommendation()", "method", "AMRDetailResultRepository.Report()", "error", err.Error())
		exc = helperexception.Internal("gagal saat proses list detail amr", err)
		return
	}

	for i, entity := range reportEntities {
		locationCodeDecrypt, err := u.Crypto.Decrypt(entity.LocationCodeEncrypt)
		if err != nil {
			u.Log.Error("AMRUseCase.ExportRecommendation()", "method", "Crypto.Decrypt()", "error", err.Error())
			exc = helperexception.Internal("gagal mendeskripsi location code", err)
			return
		}
		readDateStr := helperconverter.ConvertTimeToString(&entity.ReadDate)

		rowVals := []interface{}{
			locationCodeDecrypt,
			entity.TypeMeter,
			entity.Tariff,
			entity.Power,
			entity.LocationType.String(),
			readDateStr,
			entity.VoltageL1,
			entity.VoltageL2,
			entity.VoltageL3,
			entity.VoltageType.String(),
			entity.CurrentL1,
			entity.CurrentL2,
			entity.CurrentL3,
			entity.CurrentN,
			entity.VoltageAngleL1,
			entity.VoltageAngleL2,
			entity.VoltageAngleL3,
			entity.CurrentAngleL1,
			entity.CurrentAngleL2,
			entity.CurrentAngleL3,
			entity.PowerFactorL1,
			entity.PowerFactorL2,
			entity.PowerFactorL3,
			entity.ActivePowerL1,
			entity.ActivePowerL2,
			entity.ActivePowerL3,
			entity.ApparentPowerL1,
			entity.ApparentPowerL2,
			entity.ApparentPowerL3,
			entity.KWHAbsTotal,
			entity.BillReffKwh,
			entity.Phase,
			entity.MeasurementType.String(),
			entity.VDrop,
			entity.VLoss,
			entity.CosPhiKecil,
			entity.ILoss,
			entity.InGreaterIMax,
			entity.OverI,
			entity.OverV,
			entity.ReversePower,
			entity.UnbalanceI,
			entity.ILowVLow,
			entity.CurrentLoop,
			entity.ActivePLoss,
			entity.Freeze,
			entity.TotalWeightedValue,
		}

		currentRow := rowOffset + int(i)
		err = fileExcel.SetSheetRow(detailSheetName, "A"+strconv.Itoa(currentRow), &rowVals)
		if err != nil {
			u.Log.Error("AMRUseCase.ExportRecommendation()", "method", "fileExcel.SetSheetRow()", "row", currentRow, "error", err.Error())
			exc = helperexception.Internal("gagal menulis data ke sheet", err)
			return
		}
	}

	buffer, err := fileExcel.WriteToBuffer()
	if err != nil {
		u.Log.Error("AMRUseCase.ExportRecommendation()", "method", "fileExcel.WriteToBuffer()", "error", err.Error())
		exc = helperexception.Internal("gagal menulis file excel ke buffer", err)
		return
	}

	fileName := amrEntity.Filename
	if !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
		ext := filepath.Ext(fileName)
		fileName = strings.TrimSuffix(fileName, ext) + ".xlsx"
	}

	resp.FileName = "report_autogen_amr_recommendation_" + fileName
	resp.ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	resp.XLSXBytes = buffer.Bytes()
	return
}
