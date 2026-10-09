package usecase

import (
	"context"
	"fmt"
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
	"logisfy/internal/storage"
	"logisfy/internal/worker"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type PrabayarUseCase struct {
	DB                       *gorm.DB
	Log                      *slog.Logger
	Validate                 *validator.Validate
	Crypto                   crypto.Crypto
	Location                 *time.Location
	PrabayarRepository       *repository.PrabayarRepository
	PrabayarDetailRepository *repository.PrabayarDetailRepository
	objectStorage            *storage.SeaweedFS
	DeletedDurationInHour    int
	NumberBatch              int

	uploadQueue chan worker.UploadJob
}

func NewPrabayarUseCase(
	db *gorm.DB,
	log *slog.Logger,
	validate *validator.Validate,
	crypto crypto.Crypto,
	location *time.Location,
	prabayarRepository *repository.PrabayarRepository,
	prabayarDetailRepository *repository.PrabayarDetailRepository,
	maxConcurrentUploads int,
	deletedDurationInHour int,
	numberBatch int,
) *PrabayarUseCase {
	uc := &PrabayarUseCase{
		DB:                       db,
		Log:                      log,
		Validate:                 validate,
		Crypto:                   crypto,
		Location:                 location,
		PrabayarRepository:       prabayarRepository,
		PrabayarDetailRepository: prabayarDetailRepository,
		DeletedDurationInHour:    deletedDurationInHour,
		NumberBatch:              numberBatch,

		uploadQueue: make(chan worker.UploadJob, maxConcurrentUploads*4),
	}

	uc.StartUploadWorkers(maxConcurrentUploads)
	return uc
}

func (u *PrabayarUseCase) StartUploadWorkers(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			u.Log.Info("AMRUseCase.StartUploadWorkers()", "worker started", workerID)
			for job := range u.uploadQueue {
				u.processUploadJob(job)
			}
		}(i)
	}
}

var prabayarHeaderExpect = []string{
	"IDPEL", "BLTH", "NO METER", "TARIF", "DAYA", "TGLBACA", "KDBACA",
	"TEGANGAN", "ARUS", "COSPHI", "SEGEL", "LCD", "KEYPAD", "JML_TERMINAL",
	"STATUS TEMPER", "RELAY", "INDI_TEMPER",
}

func (u *PrabayarUseCase) Upload(ctx context.Context, req *modelrequest.UploadPrabayarReq) (resp modelresponse.UploadPrabayarResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("repository.PrabayarUseCase")
	ctx, span := tr.Start(ctx, "Upload")
	defer span.End()

	var (
		err       error
		dataCount int64
	)

	if err = u.Validate.Struct(req); err != nil {
		u.Log.Warn("PrabayarUseCase.Upload()", "Validate.Struct()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("PrabayarUseCase.Upload()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	readTx := u.DB.WithContext(ctx)
	userIDInt := uint64(userID)
	eligibleActiveStageProcesses := []coreenum.CTXEnumStageProcess{
		coreenum.CTXEnumStageProcessQueue,
		coreenum.CTXEnumStageProcessConfigSetting,
		coreenum.CTXEnumStageProcessWeightSetting,
		coreenum.CTXEnumStageProcessReady,
		coreenum.CTXEnumStageProcessDone,
		coreenum.CTXEnumStageProcessProcessing,
	}
	prabayarFilter := modelrequest.PrabayarFilter{
		UserID:         &userIDInt,
		StageProcesses: eligibleActiveStageProcesses,
	}
	prabayarEntityExist, err := u.PrabayarRepository.FindByFilter(readTx, prabayarFilter)
	if err != nil {
		u.Log.Info("PrabayarUseCase.Upload()", "PrabayarRepository().FindOneByFilter()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data prabayar", err)
		return
	}
	if len(prabayarEntityExist) > 0 {
		exc = helperexception.Conflict("data prabayar sebelumnya sudah ada, hapus terlebih dahulu !")
		return
	}

	if len(u.uploadQueue) >= cap(u.uploadQueue) {
		exc = helperexception.PermissionDenied("antrian upload penuh, coba beberapa saat lagi")
		return
	}

	objectKey := fmt.Sprintf("prabayar/%d/%s%s", userID, uuid.NewString(), req.Extension)
	err = u.objectStorage.PutObject(ctx, objectKey, req.File, req.ContentType, req.Size)
	if err != nil {
		exc = helperexception.Internal("gagal untuk menyimpan file", err)
	}

	prabayarEntity := (entity.PrabayarEntity{}).Create(userIDInt, u.Location, u.DeletedDurationInHour, req, 0, coreenum.CTXEnumStageProcessQueue)
	if err = u.PrabayarRepository.Create(readTx, prabayarEntity); err != nil {
		u.Log.Error("PrabayarUseCase.Upload()", "PrabayarRepository.Create()", "error", err.Error())
		exc = helperexception.Internal("gagal saat menyimpan prabayar", err)
		return
	}

	bgCtx := context.WithValue(context.Background(), string(coreenum.CTXEnumIDUserID), strconv.Itoa(userID))
	select {
	case u.uploadQueue <- worker.UploadJob{
		UseCaseName:     coreenum.CTXEnumUseCasePrabayar,
		UseCaseDetailID: prabayarEntity.PrabayarID,
		UserID:          userIDInt,
		FilePath:        objectKey,
		Ctx:             bgCtx,
	}:
	default:
		_ = u.objectStorage.Delete(bgCtx, objectKey)
		_ = u.PrabayarRepository.Delete(readTx, prabayarEntity)
		exc = helperexception.PermissionDenied("antrian upload penuh, coba beberapa saat lagi")
		return
	}

	resp.DataCount = dataCount
	resp.PrabayarID = prabayarEntity.PrabayarID
	return
}

func (u *PrabayarUseCase) markPrabayarFailed(tx *gorm.DB, prabayarEntity *entity.PrabayarEntity, reason string) {
	prabayarEntity.StageProcess = coreenum.CTXEnumStageProcessFailed
	prabayarEntity.FailedReason = reason
	if err := u.PrabayarRepository.Update(tx, prabayarEntity); err != nil {
		u.Log.Error("PrabayarUseCase.markAMRFailed()", "PrabayarRepository.Update()", "error", err.Error())
	}
}

func (u *PrabayarUseCase) processUploadJob(job worker.UploadJob) {
	defer os.Remove(job.FilePath)

	ctx, cancel := context.WithCancel(job.Ctx)
	defer cancel()

	log := u.Log.With("use_case_name", job.UseCaseName, "use_case_detail_id", job.UseCaseDetailID, "user_id", job.UserID)
	now := time.Now()

	readTx := u.DB.WithContext(ctx)
	prabayarFilter := modelrequest.PrabayarFilter{
		PrabayarID: &job.UseCaseDetailID,
	}
	entityPrabayar, err := u.PrabayarRepository.FindOneByFilter(readTx, prabayarFilter)
	if err != nil {
		u.Log.Error("PrabayarUseCase.processUploadJob()", "PrabayarRepository().FindOneByFilter()", "error", err.Error())
		return
	}
	if !entityPrabayar.CheckFound() {
		u.Log.Error("PrabayarUseCase.processUploadJob()", "PrabayarRepository().FindOneByFilter()", "error", "data tidak ditemukan")
		return
	}

	entityPrabayar.StageProcess = coreenum.CTXEnumStageProcessProcessing
	err = u.PrabayarRepository.Update(readTx, entityPrabayar)
	if err != nil {
		u.Log.Error("PrabayarUseCase.processUploadJob()", "PrabayarRepository().Update()", "error", err.Error())
		return
	}

	xlsx, err := excelize.OpenFile(job.FilePath)
	if err != nil {
		log.Error("AMRUseCase.processUploadJob()", "excelize.OpenFile()", "error", err.Error())
		u.markPrabayarFailed(readTx, entityPrabayar, "gagal membuka file excel")
		return
	}
	defer xlsx.Close()

	rows, mapIndex, err := helperprocess.ExtractAndValidateHeader(xlsx, prabayarHeaderExpect, 1)
	if err != nil {
		log.Warn("AMRUseCase.processUploadJob()", "helperprocess.ExtractAndValidateHeader()", "error", err.Error())
		u.markPrabayarFailed(readTx, entityPrabayar, "format header tidak valid "+err.Error())
		return
	}

	const numParseWorkers = 2
	rawCh := make(chan worker.RowJob, numParseWorkers*2)
	doneCh := make(chan *entity.PrabayarDetailEntity, numParseWorkers*2)
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
				detail, e := u.parseRow(job.Cols, job.RowNumber, mapIndex, entityPrabayar.PrabayarID)
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
		lastDetail *entity.PrabayarDetailEntity
		dataCount  int64
		batch      = make([]*entity.PrabayarDetailEntity, 0, u.NumberBatch)
	)

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		if e := u.PrabayarDetailRepository.CopyIn(ctx, batch, now); e != nil {
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
		log.Error("PrabayarUseCase.processUploadJob()", "pipeline error", processingErr.Error())
		u.markPrabayarFailed(readTx, entityPrabayar, processingErr.Error())
		for i := 0; i < 3; i++ {
			if err = u.PrabayarDetailRepository.DeleteByPrabayarID(readTx, entityPrabayar.PrabayarID); err == nil {
				break
			}
			time.Sleep(time.Duration(1<<i) * 500 * time.Millisecond)
		}
		return
	}
	if err = flushBatch(); err != nil {
		log.Error("PrabayarUseCase.processUploadJob()", "flushBatch()", "error", err.Error())
		u.markPrabayarFailed(readTx, entityPrabayar, "gagal menyimpan batch terakhir")
		return
	}

	if dataCount == 0 {
		log.Warn("PrabayarUseCase.processUploadJob()", "warning", "file tidak memiliki baris data")
		u.markPrabayarFailed(readTx, entityPrabayar, "file tidak memiliki baris data")
		return
	}
	entityPrabayar.ReadDate = lastDetail.ReadDate
	entityPrabayar.RowNumbers = dataCount
	entityPrabayar.StageProcess = coreenum.CTXEnumStageProcessConfigSetting
	if err = u.PrabayarRepository.Update(readTx, entityPrabayar); nil != err {
		log.Error("PrabayarUseCase.processUploadJob()", "PrabayarRepository().Update()", "error", err.Error())
		u.markPrabayarFailed(readTx, entityPrabayar, err.Error())
		return
	}
	log.Info("PrabayarUseCase.processUploadJob()", "status", "DONE", "rows", dataCount)
}

func (u *PrabayarUseCase) parseRow(row []string, rowNumber int64, mapIndex map[string]int, prabayarID uint64) (result *entity.PrabayarDetailEntity, err error) {
	var idpelCrypt []byte
	var noMeterCrypt []byte

	idpel := helperprocess.CellGuard(row, mapIndex["IDPEL"])
	noMeter := helperprocess.CellGuard(row, mapIndex["NO_METER"])
	tariff := helperprocess.CellGuard(row, mapIndex["TARIF"])
	power := helperprocess.CellGuard(row, mapIndex["DAYA"])
	readDate := helperprocess.CellGuard(row, mapIndex["TGLBACA"])
	voltage := helperprocess.CellGuard(row, mapIndex["TEGANGAN"])
	current := helperprocess.CellGuard(row, mapIndex["ARUS"])
	cosphi := helperprocess.CellGuard(row, mapIndex["COSPHI"])
	seal := helperprocess.CellGuard(row, mapIndex["SEGEL"])
	lcd := helperprocess.CellGuard(row, mapIndex["LCD"])
	keypad := helperprocess.CellGuard(row, mapIndex["KEYPAD"])
	terminalAmount := helperprocess.CellGuard(row, mapIndex["JML_TERMINAL"])
	temperStatus := helperprocess.CellGuard(row, mapIndex["STATUS TEMPER"])
	relay := helperprocess.CellGuard(row, mapIndex["RELAY"])
	temperIndicator := helperprocess.CellGuard(row, mapIndex["INDI_TEMPER"])

	requiredFields := map[string]string{
		"IDPEL":         idpel,
		"NO_METER":      noMeter,
		"TARIF":         tariff,
		"POWER":         power,
		"TEGANGAN":      voltage,
		"ARUS":          current,
		"COSPHI":        cosphi,
		"SEGEL":         seal,
		"LCD":           lcd,
		"KEYPAD":        keypad,
		"JML_TERMINAL":  terminalAmount,
		"STATUS TEMPER": temperStatus,
		"RELAY":         relay,
		"INDI_TEMPER":   temperIndicator,
		"TGLBACA":       readDate,
	}

	for fieldName, value := range requiredFields {
		if strings.TrimSpace(value) == "" {
			err = helperprocess.AddEmptyRowErr(rowNumber, fieldName)
			return
		}
	}

	idpelCrypt, err = u.Crypto.Encrypt(idpel)
	if err != nil {
		return
	}

	noMeterCrypt, err = u.Crypto.Encrypt(noMeter)
	if err != nil {
		return
	}

	powerVal, parseErr := helperprocess.ParseFloatField(power, rowNumber, "DAYA")
	if parseErr != nil {
		err = parseErr
		return
	}

	currentVal, parseErr := helperprocess.ParseFloatField(current, rowNumber, "ARUS")
	if parseErr != nil {
		err = parseErr
		return
	}

	voltageVal, parseErr := helperprocess.ParseFloatField(voltage, rowNumber, "TEGANGAN")
	if parseErr != nil {
		err = parseErr
		return
	}

	cosphiVal, parseErr := helperprocess.ParseFloatField(cosphi, rowNumber, "COSPHI")
	if parseErr != nil {
		err = parseErr
		return
	}

	terminalAmountVal, parseErr := helperprocess.ParseIntField(terminalAmount, rowNumber, "JML_TERMINAL")
	if parseErr != nil {
		err = parseErr
		return
	}

	readDateTime, parseErr := helperconverter.ConvertStringToTime(readDate)
	if parseErr != nil {
		err = helperprocess.AddUnsupportFieldErr(
			rowNumber,
			"READ_DATE",
			"Format READ_DATE tidak valid. Format yang didukung, diantaranya "+
				"\"1/2/2006 3:04:05 PM\", "+
				"\"1/2/2006 3:04 PM\", "+
				"\"1/2/2006 15:04:05\", "+
				"\"1/2/2006 15:04\", "+
				"\"2006-01-02 15:04:05\", dan "+
				"\"2006-01-02 15:04\".",
		)
		return
	}

	result = &entity.PrabayarDetailEntity{
		PrabayarID:      prabayarID,
		IDPelCrypt:      idpelCrypt,
		NoMeterCrypt:    noMeterCrypt,
		Tariff:          tariff,
		Power:           powerVal,
		ReadDate:        readDateTime,
		Voltage:         voltageVal,
		Current:         currentVal,
		Cosphi:          cosphiVal,
		SealCondition:   seal,
		LCDCondition:    lcd,
		KeypadCondition: keypad,
		TerminalAmount:  terminalAmountVal,
		TemperIndicator: temperIndicator,
		RelayIndicator:  relay,
	}
	return
}
