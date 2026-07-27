package usecase

import (
	"context"
	"errors"
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
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/xuri/excelize/v2"
	"go.opentelemetry.io/otel"
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
) *AMRUseCase {
	return &AMRUseCase{
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
	}
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

func (u *AMRUseCase) Upload(ctx context.Context, req *modelrequest.UploadAMRReq) (resp modelresponse.UploadAMRResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AMRUseCase")
	ctx, span := tr.Start(ctx, "Upload()")
	defer span.End()

	var (
		err               error
		amrEntityDetail   *entity.AMRDetailEntity
		amrDetailEntities []*entity.AMRDetailEntity
		dataCount         int64
		cols              []string
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
	amrEntityExist, err := u.AMRRepository.FindByUserID(readTx, uint64(userID))
	if err != nil {
		u.Log.Info("AMRUseCase.Upload()", "AMRRepository().FindByUserID()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari data amr", err)
		return
	}
	if len(amrEntityExist) > 0 {
		exc = helperexception.Conflict("data amr sebelumnya sudah ada, hapus terlebih dahulu !")
		return
	}

	tx := readTx.Begin()
	defer func() {
		if exc != nil || err != nil {
			tx.Rollback()
		}
	}()

	amrEntity := (entity.AMREntity{}).Create(uint64(userID), u.Location, u.DeletedDurationInHour, req, 0, coreenum.CTXEnumStageProcessConfigSetting)
	if err = u.AMRRepository.Create(tx, amrEntity); err != nil {
		u.Log.Error("AMRUseCase.Upload()", "AMRRepository.Create()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk membuat entity AMR", err)
		return
	}

	// Commit TX1 lebih dulu agar amrEntity terlihat di koneksi lain (diperlukan untuk FK di CopyIn)
	if err = tx.Commit().Error; err != nil {
		u.Log.Error("AMRUseCase.Upload()", "tx.Commit() phase-1", "error", err.Error())
		exc = helperexception.Internal("gagal commit entity AMR", err)
		return
	}

	// Jika CopyIn atau update gagal, hapus amrEntity sebagai kompensasi
	var copyInSuccess bool
	defer func() {
		if !copyInSuccess && amrEntity != nil && amrEntity.AMRID > 0 {
			cleanupTx := u.DB.WithContext(ctx)
			if delErr := u.AMRRepository.Delete(cleanupTx, amrEntity); delErr != nil {
				u.Log.Error("AMRUseCase.Upload()", "AMRRepository.Delete() compensation", "error", delErr.Error())
			}
		}
	}()

	xlsx, err := excelize.OpenReader(req.File)
	if err != nil {
		u.Log.Warn("AMRUseCase.Upload()", "excelize.OpenReader()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	defer func() {
		if closeErr := xlsx.Close(); closeErr != nil {
			u.Log.Error("AMRUseCase.Upload()", "xlsx.Close()", "error", closeErr.Error())
			return
		}
	}()

	rows, mapIndex, err := helperprocess.ExtractAndValidateHeader(xlsx, headerExpect)
	if err != nil {
		u.Log.Warn("AMRUseCase.Upload()", "helperprocess.ExtractAndValidateHeader()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	for rows.Next() {
		dataCount += 1
		cols, err = rows.Columns()
		if err != nil {
			u.Log.Error("AMRUseCase.Upload()", "rows.Columns()", "error", err.Error())
			exc = helperexception.InvalidArgument(err, req)
			return
		}

		amrEntityDetail, err = u.parseRow(cols, dataCount+1, mapIndex, amrEntity.AMRID)
		if err != nil {
			u.Log.Error("AMRUseCase.Upload()", "u.parseRow()", "error", err.Error())
			exc = helperexception.InvalidArgument(err, req)
			return
		}
		amrDetailEntities = append(amrDetailEntities, amrEntityDetail)

		if len(amrDetailEntities) == u.NumberBatch {
			if err = u.AMRDetailRepository.CopyIn(ctx, amrDetailEntities); err != nil {
				u.Log.Error("AMRUseCase.Upload()", "AMRDetailRepository.CopyIn()", "error", err.Error())
				exc = helperexception.Internal("gagal untuk membuat entity detail AMR", err)
				return
			}
			amrDetailEntities = amrDetailEntities[:0]
		}
	}
	if len(amrDetailEntities) > 0 {
		if err = u.AMRDetailRepository.CopyIn(ctx, amrDetailEntities); err != nil {
			u.Log.Error("AMRUseCase.Upload()", "AMRDetailRepository.CopyIn()", "error", err.Error())
			exc = helperexception.Internal("gagal untuk membuat entity detail AMR", err)
			return
		}
		amrDetailEntities = amrDetailEntities[:0]
	}

	// TX2: update data count pada amrEntity
	tx2 := u.DB.WithContext(ctx).Begin()
	defer func() {
		if exc != nil || err != nil {
			tx2.Rollback()
		}
	}()

	// update data count
	amrEntity.ReadDate = amrEntityDetail.ReadDate
	amrEntity.RowNumbers = dataCount
	if err = u.AMRRepository.Update(tx2, amrEntity); err != nil {
		u.Log.Error("AMRUseCase.Upload()", "AMRRepository.Update()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk update jumlah baris", err)
		return
	}

	// resp
	resp.DataCount = dataCount
	resp.AMRID = amrEntity.AMRID

	// commit TX2
	if err = tx2.Commit().Error; err != nil {
		u.Log.Error("AMRUseCase.Upload()", "tx2.Commit()", "error", err.Error())
		exc = helperexception.Internal("failed to commit update amr", err)
		return
	}

	copyInSuccess = true
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
	entityUserList, items, totalPages, size, err := u.AMRRepository.List(tx, req.QueryInfo)
	if err != nil {
		u.Log.Info("AMRUseCase.List()", "AMRRepository.List()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses list amr", err)
		return
	}

	// resp
	resp.List = (&entity.AMREntity{}).ConvertToList(entityUserList)
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
		exc = helperexception.NotFound("data konfigurasi parameter AMR")
		return
	}

	createdAtStr := helperconverter.ConvertTimeToString(&paramConfigEntity.CreatedAt)

	resp.AMRID = amrEntity.AMRID
	resp.AMRConfigID = paramConfigEntity.AMRConfigID
	resp.VDropVTM = paramConfigEntity.VDropVTM
	resp.VDropVTR = paramConfigEntity.VDropVTR
	resp.VDropITM = paramConfigEntity.VDropITM
	resp.VDropITR = paramConfigEntity.VDropITR
	resp.VLossVTM = paramConfigEntity.VLossVTM
	resp.VLossVTR = paramConfigEntity.VLossVTR
	resp.VLossITM = paramConfigEntity.VLossITM
	resp.VLossITR = paramConfigEntity.VLossITR
	resp.CosPhiKecilITM = paramConfigEntity.CosPhiKecilITM
	resp.CosPhiKecilITR = paramConfigEntity.CosPhiKecilITR
	resp.CosPhiKecilUpperLimitTM = paramConfigEntity.CosPhiKecilUpperLimitTM
	resp.CosPhiKecilUpperLimitTR = paramConfigEntity.CosPhiKecilUpperLimitTR
	resp.ILossITM = paramConfigEntity.ILossITM
	resp.ILossITR = paramConfigEntity.ILossITR
	resp.ILossIMaxTM = paramConfigEntity.ILossIMaxTM
	resp.ILossIMaxTR = paramConfigEntity.ILossIMaxTR
	resp.InGreaterIMaxInTM = paramConfigEntity.InGreaterIMaxInTM
	resp.InGreaterIMaxInTR = paramConfigEntity.InGreaterIMaxInTR
	resp.OverCurrentIMaxTM = paramConfigEntity.OverCurrentIMaxTM
	resp.OverCurrentIMaxTR = paramConfigEntity.OverCurrentIMaxTR
	resp.OverVoltageVMaxTM = paramConfigEntity.OverVoltageVMaxTM
	resp.OverVoltageVMaxTR = paramConfigEntity.OverVoltageVMaxTR
	resp.ReversePowerVTM = paramConfigEntity.ReversePowerVTM
	resp.ReversePowerVTR = paramConfigEntity.ReversePowerVTR
	resp.ReversePowerITM = paramConfigEntity.ReversePowerITM
	resp.ReversePowerITR = paramConfigEntity.ReversePowerITR
	resp.IUnbalanceTolTM = paramConfigEntity.IUnbalanceTolTM
	resp.IUnbalanceTolTR = paramConfigEntity.IUnbalanceTolTR
	resp.IUnbalanceITM = paramConfigEntity.IUnbalanceITM
	resp.IUnbalanceITR = paramConfigEntity.IUnbalanceITR
	resp.PLossI = paramConfigEntity.PLossI
	resp.ILowVLowTM = paramConfigEntity.ILowVLowTM
	resp.ILowVLowTR = paramConfigEntity.ILowVLowTR
	resp.MinIndicatorAmount = paramConfigEntity.MinIndicatorAmount
	resp.MinWeight = paramConfigEntity.MinWeight
	resp.NShowRecommendation = paramConfigEntity.NShowRecommendation
	resp.CreatedAt = createdAtStr

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
		exc = helperexception.NotFound("data konfigurasi bobot AMR")
		return
	}

	createdAtStr := helperconverter.ConvertTimeToString(&weightConfigEntity.CreatedAt)

	resp.AMRID = amrEntity.AMRID
	resp.AMRWeightConfigID = weightConfigEntity.AMRWeightConfigID
	resp.VIndirectDrop = weightConfigEntity.VIndirectDrop
	resp.VDirectDrop = weightConfigEntity.VDirectDrop
	resp.VLoss = weightConfigEntity.VLoss
	resp.CosPhiKecil = weightConfigEntity.CosPhiKecil
	resp.ILoss = weightConfigEntity.ILoss
	resp.InGreatedImax = weightConfigEntity.InGreatedImax
	resp.OverCurrent = weightConfigEntity.OverCurrent
	resp.OverVoltage = weightConfigEntity.OverVoltage
	resp.ReversePower = weightConfigEntity.ReversePower
	resp.UnbalanceI = weightConfigEntity.UnbalanceI
	resp.ILowVLow = weightConfigEntity.ILowVLow
	resp.CurrentLoop = weightConfigEntity.CurrentLoop
	resp.ActivePLoss = weightConfigEntity.ActivePLoss
	resp.Freeze = weightConfigEntity.Freeze
	resp.CreatedAt = createdAtStr

	return
}
