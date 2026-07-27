package http

import (
	"log/slog"
	coreresponse "logisfy/core/response"
	helpergenerator "logisfy/helper/generator"
	modelrequest "logisfy/internal/model/request"
	modelresponse "logisfy/internal/model/response"
	"logisfy/internal/usecase"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
)

type NewsHandler struct {
	Log     *slog.Logger
	UseCase *usecase.NewsUseCase
}

func NewNewsHandler(log *slog.Logger, useCase *usecase.NewsUseCase) *NewsHandler {
	return &NewsHandler{
		Log:     log,
		UseCase: useCase,
	}
}

// List
//
// @Summary		List News
// @Description	List News
// @Tags		Auth
// @Produce		json
// @Param		filter		query		string									false				"Search Parameter"
// @Param		sort		query		string									false				"Sorting Parameter"
// @Param		page		query		int										false				"Current Page"
// @Param		pageSize	query		int										false				"Page Size"
// @Success		200			{object}	coreresponse.ApiResponse[modelresponse.ListNewsResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.ListNewsResp]	"Result"
// @Router		/public/news [get]
func (handler *NewsHandler) List(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.NewsHandler")
	ctx, span := tr.Start(ctx, "List()")
	defer span.End()

	// Get Param
	queryInfo, err := helpergenerator.GenerateQueryInfoPostgreSQL(fiberCtx)
	if err != nil {
		errString := "parameter query tidak valid"
		handler.Log.Info("NewsHandler.List()", "helpergenerator.GenerateQueryInfoPostgreSQL()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.ListNewsResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusBadRequest,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	// pointing param
	requestData := &modelrequest.ListNewsReq{}
	requestData.QueryInfo = queryInfo

	// exec usecase
	response, exc := handler.UseCase.List(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AuthHandler.List()", "UseCase.List()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.ListNewsResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.ListNewsResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}
