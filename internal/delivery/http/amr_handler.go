package http

import (
	"fmt"
	"log/slog"
	coreresponse "logisfy/core/response"
	helpergenerator "logisfy/helper/generator"
	modelrequest "logisfy/internal/model/request"
	modelresponse "logisfy/internal/model/response"
	"logisfy/internal/usecase"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
)

const maxUploadSizeBytes = 200 << 20

type AMRHandler struct {
	Log     *slog.Logger
	UseCase *usecase.AMRUseCase
}

func NewAMRHandler(
	log *slog.Logger,
	useCase *usecase.AMRUseCase,
) *AMRHandler {
	return &AMRHandler{
		Log:     log,
		UseCase: useCase,
	}
}

// Upload
//
//	@Summary		Upload AMR dataset
//	@Description	Upload AMR dataset
//	@Tags			AMR
//	@Produce		json
//	@Param			attachment	formData	file							true	"File Excel AMR (.xlsx/.xls/.csv)"
//	@Success		201			{object}	coreresponse.ApiResponse[modelresponse.UploadAMRResp]	"Result"
//	@Failure		400			{object}	coreresponse.ApiResponse[modelresponse.UploadAMRResp]	"Result"
//	@Router			/user/amr [post]
//	@Security		Bearer
func (handler *AMRHandler) Upload(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()

	// Init Ctx
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)
	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "Upload()")
	defer span.End()

	fileHeader, err := fiberCtx.FormFile("attachment")
	if err != nil {
		errString := err.Error()
		handler.Log.Warn("AMRHandler.Upload()", "fiberCtx.FormFile()", "warn", err.Error())
		return fiberCtx.Status(fiber.StatusUnprocessableEntity).JSON(coreresponse.ApiResponse[modelresponse.UploadAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusUnprocessableEntity,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	if fileHeader.Size > maxUploadSizeBytes {
		handler.Log.Warn("AMRHandler.Upload()", "file size exceeded", "warn", fileHeader.Size)
		return fiberCtx.Status(fiber.StatusRequestEntityTooLarge).JSON(coreresponse.ApiResponse[modelresponse.UploadAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusRequestEntityTooLarge,
			Error:   &fiber.ErrRequestEntityTooLarge.Message,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	file, err := fileHeader.Open()
	if err != nil {
		errString := err.Error()
		handler.Log.Warn("AMRHandler.Upload()", "fileHeader.Open()", "warn", err.Error())
		return fiberCtx.Status(fiber.StatusUnprocessableEntity).JSON(coreresponse.ApiResponse[modelresponse.UploadAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusUnprocessableEntity,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	defer file.Close()

	requestData := &modelrequest.UploadAMRReq{
		File:      file,
		Size:      fileHeader.Size,
		Filename:  fileHeader.Filename,
		Extension: filepath.Ext(fileHeader.Filename),
	}

	// exec usecase
	response, exc := handler.UseCase.Upload(ctx, requestData)
	if exc != nil {
		handler.Log.Warn("AMRHandler.Upload()", "UseCase.Upload()", "warn", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.UploadAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusCreated).JSON(coreresponse.ApiResponse[modelresponse.UploadAMRResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusCreated,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// FindAMR
//
//	@Summary		Find AMR
//	@Description	Find AMR
//	@Tags			AMR
//	@Produce		json
//	@Param			amr_id		path		uint64		true	"AMR ID"
//	@Success		200			{object}	coreresponse.ApiResponse[modelresponse.FindAMRResp]	"Result"
//	@Failure		400			{object}	coreresponse.ApiResponse[modelresponse.FindAMRResp]	"Result"
//	@Router			/user/amr/:amr_id [get]
//	@Security		Bearer
func (handler *AMRHandler) FindAMR(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "FindAMR()")
	defer span.End()

	requestData := &modelrequest.FindAMRReq{
		AMRID: fiberCtx.Params("amr_id"),
	}

	// exec usecase
	response, exc := handler.UseCase.FindAMR(ctx, requestData)
	if exc != nil {
		handler.Log.Warn("AMRHandler.FindAMR()", "UseCase.FindAMR()", "warn", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.FindAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.FindAMRResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// FindParamConfig
//
//	@Summary		Find AMR Param Config
//	@Description	Find AMR Param Config by amr_id
//	@Tags			AMR
//	@Produce		json
//	@Param			amr_id		path		uint64		true	"AMR ID"
//	@Success		200			{object}	coreresponse.ApiResponse[modelresponse.FindAMRParamConfigResp]	"Result"
//	@Failure		400			{object}	coreresponse.ApiResponse[modelresponse.FindAMRParamConfigResp]	"Result"
//	@Router			/user/amr/:amr_id/param-config [get]
//	@Security		Bearer
func (handler *AMRHandler) FindParamConfig(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "FindParamConfig()")
	defer span.End()

	requestData := &modelrequest.FindAMRParamConfigReq{
		AMRID: fiberCtx.Params("amr_id"),
	}

	// exec usecase
	response, exc := handler.UseCase.FindAMRParamConfig(ctx, requestData)
	if exc != nil {
		handler.Log.Warn("AMRHandler.FindParamConfig()", "UseCase.FindAMRParamConfig()", "warn", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.FindAMRParamConfigResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.FindAMRParamConfigResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// FindWeightConfig
//
//	@Summary		Find AMR Weight Config
//	@Description	Find AMR Weight Config by amr_id
//	@Tags			AMR
//	@Produce		json
//	@Param			amr_id		path		uint64		true	"AMR ID"
//	@Success		200			{object}	coreresponse.ApiResponse[modelresponse.FindAMRWeightConfigResp]	"Result"
//	@Failure		400			{object}	coreresponse.ApiResponse[modelresponse.FindAMRWeightConfigResp]	"Result"
//	@Router			/user/amr/:amr_id/weight-config [get]
//	@Security		Bearer
func (handler *AMRHandler) FindWeightConfig(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "FindWeightConfig()")
	defer span.End()

	requestData := &modelrequest.FindAMRWeightConfigReq{
		AMRID: fiberCtx.Params("amr_id"),
	}

	// exec usecase
	response, exc := handler.UseCase.FindAMRWeightConfig(ctx, requestData)
	if exc != nil {
		handler.Log.Warn("AMRHandler.FindWeightConfig()", "UseCase.FindAMRWeightConfig()", "warn", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.FindAMRWeightConfigResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.FindAMRWeightConfigResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// CreateParamConfig
//
// @Summary		Create Param Config
// @Description	Create Param Config of AMR
// @Tags		AMR
// @Accept		json
// @Produce		json
// @Param		amr_id		path		uint64		true	"AMR ID"
// @Param		data		body		modelrequest.CreateParamConfigAMRReq					true		"CreateParamConfig of AMR Request Parameter"
// @Success		201			{object}	coreresponse.ApiResponse[modelresponse.CreateParamConfigAMRResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.CreateParamConfigAMRResp]	"Result"
// @Router		/user/amr/:amr_id/param-config [post]
func (handler *AMRHandler) CreateParamConfig(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "CreateParamConfig()")
	defer span.End()

	requestData := &modelrequest.CreateParamConfigAMRReq{
		AMRID: fiberCtx.Params("amr_id"),
	}

	if err := fiberCtx.BodyParser(requestData); err != nil {
		errString := err.Error()
		handler.Log.Info("AMRHandler.CreateParamConfig()", "fiberCtx.BodyParser()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.CreateParamConfigAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusBadRequest,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	response, exc := handler.UseCase.CreateParamConfig(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AMRHandler.CreateParamConfig()", "UseCase.CreateParamConfig()", "Info", exc.GetError())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.CreateParamConfigAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusCreated).JSON(coreresponse.ApiResponse[modelresponse.CreateParamConfigAMRResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusCreated,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// CreateWeightConfig
//
// @Summary		Create Weight Config
// @Description	Create Weight Config of AMR
// @Tags		AMR
// @Accept		json
// @Produce		json
// @Param		amr_id		path		uint64		true	"AMR ID"
// @Param		data		body		modelrequest.CreateWeightConfigAMRReq					true		"CreateWeightConfig of AMR Request Parameter"
// @Success		201			{object}	coreresponse.ApiResponse[modelresponse.CreateWeightConfigAMRResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.CreateWeightConfigAMRResp]	"Result"
// @Router		/user/amr/:amr_id/weight-config [post]
func (handler *AMRHandler) CreateWeightConfig(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "CreateWeightConfig()")
	defer span.End()

	requestData := &modelrequest.CreateWeightConfigAMRReq{
		AMRID: fiberCtx.Params("amr_id"),
	}

	if err := fiberCtx.BodyParser(requestData); err != nil {
		errString := err.Error()
		handler.Log.Info("AMRHandler.CreateWeightConfig()", "fiberCtx.BodyParser()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.CreateWeightConfigAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusBadRequest,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	response, exc := handler.UseCase.CreateWeightConfig(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AMRHandler.CreateWeightConfig()", "UseCase.CreateWeightConfig()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.CreateWeightConfigAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusCreated).JSON(coreresponse.ApiResponse[modelresponse.CreateWeightConfigAMRResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusCreated,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// List
//
//	@Summary		FindByAMRID All AMR
//	@Description	FindByAMRID All AMR
//	@Tags			AMR
//	@Accept			json
//	@Produce		json
//	@Param			filter		query		string									false				"Search Parameter"
//	@Param			sort		query		string									false				"Sorting Parameter"
//	@Param			page		query		int										false				"Current Page"
//	@Param			pageSize	query		int										false				"Rows Count"
//	@Success		200		{object}	coreresponse.ApiResponse[modelresponse.ListAMRResp]		"Result"
//	@Router			/user/amr [get]
//	@Security		Bearer
func (handler *AMRHandler) List(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "List()")
	defer span.End()

	// Get Param
	queryInfo, err := helpergenerator.GenerateQueryInfoPostgreSQL(fiberCtx)
	if err != nil {
		errString := "parameter query tidak valid"
		handler.Log.Info("AMRHandler.List()", "helpergenerator.GenerateQueryInfoPostgreSQL()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.ListAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusBadRequest,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	// Pointing Param
	requestData := &modelrequest.ListAMRReq{}
	requestData.QueryInfo = queryInfo

	// exec usecase
	response, exc := handler.UseCase.List(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AMRHandler.List()", "UseCase.List()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.ListAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.ListAMRResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// GenerateReport
//
// @Summary		Generate Report AMR
// @Description	Create Generate Report AMR
// @Tags		AMR
// @Accept		json
// @Produce		json
// @Param		amr_id		path		uint64		true	"AMR ID"
// @Param		data		body		modelrequest.GenerateReportAMRReq					true		"Generare Report of AMR Request Parameter"
// @Success		201			{object}	coreresponse.ApiResponse[modelresponse.GenerateReportAMRResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.GenerateReportAMRResp]	"Result"
// @Router		/user/amr/:amr_id/generate-report [post]
func (handler *AMRHandler) GenerateReport(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "GenerateReport()")
	defer span.End()

	requestData := &modelrequest.GenerateReportAMRReq{
		AMRID: fiberCtx.Params("amr_id"),
	}

	response, exc := handler.UseCase.GenerateReport(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AMRHandler.GenerateReport()", "UseCase.GenerateReport()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.GenerateReportAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusCreated).JSON(coreresponse.ApiResponse[modelresponse.GenerateReportAMRResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusCreated,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// Report
//
//	@Summary		Get Report AMR
//	@Description	Get Report AMR by AMR ID
//	@Tags			AMR
//	@Accept			json
//	@Produce		json
//	@Param			amr_id		path		uint64									true				"AMR ID"
//	@Param			filter		query		string									false				"Search Parameter"
//	@Param			sort		query		string									false				"Sorting Parameter"
//	@Param			page		query		int										false				"Current Page"
//	@Param			pageSize	query		int										false				"Rows Count"
//	@Success		200			{object}	coreresponse.ApiResponse[modelresponse.ListReportAMRResp]	"Result"
//	@Router			/user/amr/:amr_id/report [get]
//	@Security		Bearer
func (handler *AMRHandler) Report(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "Report()")
	defer span.End()

	// Get Param
	queryInfo, err := helpergenerator.GenerateQueryInfoPostgreSQL(fiberCtx)
	if err != nil {
		errString := "parameter query tidak valid"
		handler.Log.Info("AMRHandler.Report()", "helpergenerator.GenerateQueryInfoPostgreSQL()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.ListReportAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusBadRequest,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	// Pointing Param
	requestData := &modelrequest.ListReportAMRReq{}
	requestData.AMRID = fiberCtx.Params("amr_id")
	requestData.QueryInfo = queryInfo

	// exec usecase
	response, exc := handler.UseCase.Report(ctx, requestData)
	if exc != nil {
		handler.Log.Warn("AMRHandler.Report()", "UseCase.Report()", "warn", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.ListReportAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.ListReportAMRResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// Delete
//
//	@Summary		Delete AMR
//	@Description	Delete AMR
//	@Tags			AMR
//	@Produce		json
//	@Param			amr_id		path		uint64		true	"AMR ID"
//	@Success		200			{object}	coreresponse.ApiResponse[modelresponse.DeleteAMRResp]	"Result"
//	@Failure		400			{object}	coreresponse.ApiResponse[modelresponse.DeleteAMRResp]	"Result"
//	@Router			/user/amr/:amr_id [delete]
//	@Security		Bearer
func (handler *AMRHandler) Delete(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "FindAMR()")
	defer span.End()

	requestData := &modelrequest.DeleteAMRReq{
		AMRID: fiberCtx.Params("amr_id"),
	}

	// exec usecase
	response, exc := handler.UseCase.Delete(ctx, requestData)
	if exc != nil {
		handler.Log.Warn("AMRHandler.Delete()", "UseCase.Delete()", "warn", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.DeleteAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.DeleteAMRResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// Summary
//
//	@Summary		Get Summary AMR
//	@Description	Get Summary AMR by AMR ID
//	@Tags			AMR
//	@Accept			json
//	@Produce		json
//	@Param			amr_id		path		uint64									true				"AMR ID"
//	@Success		200			{object}	coreresponse.ApiResponse[modelresponse.SummaryAMRResp]	"Result"
//	@Router			/user/amr/:amr_id/summary [get]
//	@Security		Bearer
func (handler *AMRHandler) Summary(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "Summary()")
	defer span.End()

	// Pointing Param
	requestData := &modelrequest.SummaryAMRReq{}
	requestData.AMRID = fiberCtx.Params("amr_id")

	// exec usecase
	response, exc := handler.UseCase.Summary(ctx, requestData)
	if exc != nil {
		handler.Log.Warn("AMRHandler.Summary()", "UseCase.Summary()", "warn", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.SummaryAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.SummaryAMRResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// DownloadTemplate
//
//	@Summary		Download AMR template file
//	@Description	Download the Excel template file (.xlsx) for uploading AMR data
//	@Tags			AMR
//	@Accept			json
//	@Produce		octet-stream
//	@Success		200	{file}		binary	"XLSX Template file"
//	@Failure		500	{object}	coreresponse.ApiResponse[any]	"Error"
//	@Router			/user/amr/template [get]
//	@Security		Bearer
func (handler *AMRHandler) DownloadTemplate(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "DownloadTemplate()")
	defer span.End()

	// exec usecase
	response, exc := handler.UseCase.DownloadTemplate(ctx)
	if exc != nil {
		handler.Log.Warn("AMRHandler.DownloadTemplate()", "UseCase.DownloadTemplate()", "warn", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.DownloadAMRTemplateResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	fiberCtx.Set("Content-Type", response.ContentType)
	fiberCtx.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, response.FileName))

	return fiberCtx.Send(response.XLSXBytes)
}

// Export
//
//	@Summary		Export Data
//	@Description	Export Data
//	@Tags			AMR
//	@Accept			json
//	@Produce		json
//	@Param			filter				query		string									false				"Search Parameter"
//	@Param			sort				query		string									false				"Sorting Parameter"
//	@Success		200					{file}		binary									"XLSX Template file"
//	@Failure		500					{object}	coreresponse.ApiResponse[modelresponse.ExportAMRResp]		"Error"
//	@Failure      	400   				{object}  	coreresponse.ApiResponse[modelresponse.ExportAMRResp]  		"Bad Request"
//	@Router			/user/amr/:amr_id/export [get]
//	@Security		Bearer
func (handler *AMRHandler) Export(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "Export()")
	defer span.End()

	// Get Param
	queryInfo, err := helpergenerator.GenerateQueryInfoPostgreSQL(fiberCtx)
	if err != nil {
		errString := err.Error()
		handler.Log.Info("AMRHandler.Export()", "helpergenerator.GenerateQueryInfoPostgreSQL()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.ExportAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusBadRequest,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	// Pointing Param
	requestData := &modelrequest.ExportAMRReq{
		AMRID:     fiberCtx.Params("amr_id"),
		QueryInfo: queryInfo,
	}

	// exec usecase
	response, exc := handler.UseCase.Export(ctx, requestData)
	if exc != nil {
		handler.Log.Warn("AMRHandler.Export() failed", "usecase", "UseCase.Export()", "error", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.ExportAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	fiberCtx.Set("Content-Type", response.ContentType)
	fiberCtx.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, response.FileName))

	return fiberCtx.Send(response.XLSXBytes)
}

// ExportRecommendation
//
//	@Summary		Export Recommendation Data
//	@Description	Export Recommendation Data
//	@Tags			AMR
//	@Accept			json
//	@Produce		json
//	@Param			filter				query		string									false				"Search Parameter"
//	@Param			sort				query		string									false				"Sorting Parameter"
//	@Success		200					{file}		binary									"XLSX Template file"
//	@Failure		500					{object}	coreresponse.ApiResponse[modelresponse.ExportRecommendationAMRResp]		"Error"
//	@Failure      	400   				{object}  	coreresponse.ApiResponse[modelresponse.ExportRecommendationAMRResp]  		"Bad Request"
//	@Router			/user/amr/:amr_id/export-recommendation [get]
//	@Security		Bearer
func (handler *AMRHandler) ExportRecommendation(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AMRHandler")
	ctx, span := tr.Start(ctx, "ExportRecommendation()")
	defer span.End()

	// Get Param
	queryInfo, err := helpergenerator.GenerateQueryInfoPostgreSQL(fiberCtx)
	if err != nil {
		errString := err.Error()
		handler.Log.Info("AMRHandler.ExportRecommendation()", "helpergenerator.GenerateQueryInfoPostgreSQL()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.ExportRecommendationAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusBadRequest,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	// Pointing Param
	requestData := &modelrequest.ExportRecommendationAMRReq{
		AMRID:     fiberCtx.Params("amr_id"),
		QueryInfo: queryInfo,
	}

	// exec usecase
	response, exc := handler.UseCase.ExportRecommendation(ctx, requestData)
	if exc != nil {
		handler.Log.Warn("AMRHandler.ExportRecommendation() failed", "usecase", "UseCase.ExportRecommendation()", "error", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.ExportRecommendationAMRResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	fiberCtx.Set("Content-Type", response.ContentType)
	fiberCtx.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, response.FileName))

	return fiberCtx.Send(response.XLSXBytes)
}
