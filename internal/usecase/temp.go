package usecase

//
//import (
//	"io"
//	coreenum "logisfy/core/enum"
//	helperexception "logisfy/helper/exception"
//	helperprocess "logisfy/helper/process"
//	"logisfy/internal/entity"
//	"os"
//	"strconv"
//
//	"github.com/xuri/excelize/v2"
//	"go.opentelemetry.io/otel"
//)
//
//func (u *AMRUseCase) Upload(ctx context.Context, req *modelrequest.UploadAMRReq) (resp modelresponse.UploadAMRResp, exc *helperexception.Exception) {
//	// Init Tracer
//	tr := otel.Tracer("useCase.AMRUseCase")
//	ctx, span := tr.Start(ctx, "Upload()")
//	defer span.End()
//
//	if err := u.Validate.Struct(req); err != nil {
//		u.Log.Warn("AMRUseCase.Upload()", "Validate.Struct()", "warn", err.Error())
//		exc = helperexception.InvalidArgument(err, req)
//		return
//	}
//
//	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
//	if err != nil {
//		u.Log.Warn("AMRUseCase.Upload()", "strconv.Atoi()", "warn", err.Error())
//		exc = helperexception.InvalidArgument(err, req)
//		return
//	}
//
//	readTx := u.DB.WithContext(ctx)
//	amrEntityExist, err := u.AMRRepository.FindByUserID(readTx, uint64(userID))
//	if err != nil {
//		u.Log.Info("AMRUseCase.Upload()", "AMRRepository().FindByUserID()", "error", err.Error())
//		exc = helperexception.Internal("gagal untuk mencari data amr", err)
//		return
//	}
//	if len(amrEntityExist) > 0 {
//		exc = helperexception.Conflict("data amr sebelumnya sudah ada, hapus terlebih dahulu !")
//		return
//	}
//
//	tmpFile, err := os.CreateTemp("./temp/amr", "amr_upload_*.xlsx")
//	if err != nil {
//		u.Log.Error("AMRUseCase.Upload()", "os.CreateTemp()", "error", err.Error())
//		exc = helperexception.Internal("gagal membuat file temporary", err)
//		return
//	}
//	tmpFilePath := tmpFile.Name()
//
//	if _, err = io.Copy(tmpFile, req.File); err != nil {
//		_ = tmpFile.Close()
//		_ = os.Remove(tmpFilePath)
//		u.Log.Error("AMRUseCase.Upload()", "io.Copy()", "error", err.Error())
//		exc = helperexception.Internal("gagal menyalin file upload", err)
//		return
//	}
//	_ = tmpFile.Close()
//
//	amrEntity := (entity.AMREntity{}).Create(uint64(userID), u.Location, u.DeletedDurationInHour, req, 0, coreenum.CTXEnumStageProcessProcessing)
//	if err = u.AMRRepository.Create(readTx, amrEntity); err != nil {
//		_ = os.Remove(tmpFilePath)
//		u.Log.Error("AMRUseCase.Upload()", "AMRRepository.Create()", "error", err.Error())
//		exc = helperexception.Internal("gagal untuk membuat entity AMR", err)
//		return
//	}
//
//	go u.processAsyncUpload(amrEntity.AMRID, tmpFilePath)
//
//	resp.DataCount = 0
//	resp.AMRID = amrEntity.AMRID
//
//	return
//}
//
//func (u *AMRUseCase) processAsyncUpload(amrID uint64, tmpFilePath string) {
//	var (
//		dataCount         int64
//		amrDetailEntities = make([]*entity.AMRDetailEntity, 0, u.NumberBatch)
//		lastDetail        *entity.AMRDetailEntity
//		err               error
//	)
//	defer func() {
//		if removeErr := os.Remove(tmpFilePath); removeErr != nil && os.IsExist(removeErr) {
//			u.Log.Warn("AMRUseCase.processAsyncUpload()", "os.Remove()", "warn", removeErr.Error())
//		}
//	}()
//
//	bgCtx := context.Background()
//	readTx := u.DB.WithContext(bgCtx)
//	amrEntity, err := u.AMRRepository.Find(readTx, amrID)
//	if err != nil || !amrEntity.CheckFound() {
//		u.Log.Error("AMRUseCase.processAsyncUpload()", "AMRRepository.Find()", "error", "amr data tidak ditemukan")
//		return
//	}
//	amrEntity.StageProcess = coreenum.CTXEnumStageFailed
//
//	tx := readTx.Begin()
//	defer func() {
//		if err = tx.Commit().Error; err != nil {
//			u.Log.Error("AMRUseCase.processAsyncUpload()", "tx.Commit()", "err", err.Error())
//		}
//		if err != nil {
//			tx.Rollback()
//		}
//	}()
//
//	xlsx, err := excelize.OpenFile(tmpFilePath)
//	if err != nil {
//		u.Log.Error("AMRUseCase.processAsyncUpload()", "excelize.OpenFile()", "error", err.Error())
//		return
//	}
//	defer func() {
//		if closeErr := xlsx.Close(); closeErr != nil {
//			u.Log.Error("AMRUseCase.processAsyncUpload()", "xlsx.Close()", "error", closeErr.Error())
//		}
//	}()
//
//	rows, mapIndex, extractErr := helperprocess.ExtractAndValidateHeader(xlsx, headerExpect)
//	if extractErr != nil {
//		u.Log.Error("AMRUseCase.processAsyncUpload()", "ExtractAndValidateHeader()", "error", err.Error())
//		return
//	}
//
//	for rows.Next() {
//		dataCount++
//		cols, err := rows.Columns()
//		if err != nil {
//			u.Log.Error("AMRUseCase.processAsyncUpload()", "rows.Columns()", "error", err.Error())
//			return
//		}
//
//		amrEntityDetail, err := u.parseRow(cols, dataCount+1, mapIndex, amrID)
//		if err != nil {
//			u.Log.Error("AMRUseCase.processAsyncUpload()", "u.parseRow()", "error", err.Error())
//			return
//		}
//		lastDetail = amrEntityDetail
//		amrDetailEntities = append(amrDetailEntities, amrEntityDetail)
//
//		if len(amrDetailEntities) == u.NumberBatch {
//			if err = u.AMRDetailRepository.CopyIn(bgCtx, amrDetailEntities); err != nil {
//				u.Log.Error("AMRUseCase.processAsyncUpload()", "AMRDetailRepository.CopyIn()", "error", err.Error())
//				return
//			}
//			amrDetailEntities = amrDetailEntities[:0]
//		}
//	}
//
//	if len(amrDetailEntities) > 0 {
//		if err = u.AMRDetailRepository.CopyIn(bgCtx, amrDetailEntities); err != nil {
//			u.Log.Error("AMRUseCase.processAsyncUpload()", "AMRDetailRepository.CopyIn()", "error", err.Error())
//			return
//		}
//	}
//
//	if lastDetail != nil {
//		amrEntity.ReadDate = lastDetail.ReadDate
//	}
//	amrEntity.RowNumbers = dataCount
//	amrEntity.StageProcess = coreenum.CTXEnumStageProcessConfigSetting
//
//	if err = u.AMRRepository.Update(tx, amrEntity); err != nil {
//		tx.Rollback()
//		u.Log.Error("AMRUseCase.processAsyncUpload()", "AMRRepository.Update()", "error", err.Error())
//		return
//	}
//
//	if err = tx.Commit().Error; err != nil {
//		u.Log.Error("AMRUseCase.processAsyncUpload()", "tx.Commit()", "error", err.Error())
//		return
//	}
//}

//xlsx, err := excelize.OpenReader(req.File)
//if err != nil {
//u.Log.Warn("AMRUseCase.Upload()", "excelize.OpenReader()", "warn", err.Error())
//exc = helperexception.InvalidArgument(err, req)
//return
//}
//defer func() {
//if closeErr := xlsx.Close(); closeErr != nil {
//u.Log.Error("AMRUseCase.Upload()", "xlsx.Close()", "error", closeErr.Error())
//return
//}
//}()
//
//rows, mapIndex, err := helperprocess.ExtractAndValidateHeader(xlsx, headerExpect)
//if err != nil {
//u.Log.Warn("AMRUseCase.Upload()", "helperprocess.ExtractAndValidateHeader()", "warn", err.Error())
//exc = helperexception.InvalidArgument(err, req)
//return
//}
//
//for rows.Next() {
//dataCount += 1
//cols, err = rows.Columns()
//if err != nil {
//u.Log.Error("AMRUseCase.Upload()", "rows.Columns()", "error", err.Error())
//exc = helperexception.InvalidArgument(err, req)
//return
//}
//
//amrEntityDetail, err = u.parseRow(cols, dataCount+1, mapIndex, amrEntity.AMRID)
//if err != nil {
//u.Log.Error("AMRUseCase.Upload()", "u.parseRow()", "error", err.Error())
//exc = helperexception.InvalidArgument(err, req)
//return
//}
//amrDetailEntities = append(amrDetailEntities, amrEntityDetail)
//
//if len(amrDetailEntities) == u.NumberBatch {
//if err = u.AMRDetailRepository.CopyIn(ctx, amrDetailEntities); err != nil {
//u.Log.Error("AMRUseCase.Upload()", "AMRDetailRepository.CopyIn()", "error", err.Error())
//exc = helperexception.Internal("gagal untuk membuat entity detail AMR", err)
//return
//}
//amrDetailEntities = amrDetailEntities[:0]
//}
//}
//if len(amrDetailEntities) > 0 {
//if err = u.AMRDetailRepository.CopyIn(ctx, amrDetailEntities); err != nil {
//u.Log.Error("AMRUseCase.Upload()", "AMRDetailRepository.CopyIn()", "error", err.Error())
//exc = helperexception.Internal("gagal untuk membuat entity detail AMR", err)
//return
//}
//amrDetailEntities = amrDetailEntities[:0]
//}
//
//tx := readTx.Begin()
//defer func() {
//if exc != nil || err != nil {
//tx.Rollback()
//_ = u.AMRRepository.Delete(readTx, amrEntity)
//}
//}()
//amrEntity.ReadDate = amrEntityDetail.ReadDate
//amrEntity.RowNumbers = dataCount
//amrEntity.StageProcess = coreenum.CTXEnumStageProcessConfigSetting
//if err = u.AMRRepository.Update(tx, amrEntity); err != nil {
//u.Log.Error("AMRUseCase.Upload()", "AMRRepository.Update()", "error", err.Error())
//exc = helperexception.Internal("gagal untuk update jumlah baris", err)
//return
//}
//
//if err = tx.Commit().Error; err != nil {
//u.Log.Error("AMRUseCase.Upload()", "tx.Commit() phase-1", "error", err.Error())
//exc = helperexception.Internal("gagal commit entity AMR", err)
//return
//}
