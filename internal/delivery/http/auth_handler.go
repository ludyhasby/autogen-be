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

type AuthHandler struct {
	Log     *slog.Logger
	UseCase *usecase.AuthUseCase
}

func NewAuthHandler(
	log *slog.Logger,
	useCase *usecase.AuthUseCase,
) *AuthHandler {
	return &AuthHandler{
		Log:     log,
		UseCase: useCase,
	}
}

// Register
//
//	@Summary		Create User (Register)
//	@Description	Create User (Register) for User
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			data	body		modelrequest.RegisterUserReq		true	"Register User Request Parameter"
//	@Success		201		{object}	coreresponse.ApiResponse[modelresponse.RegisterUserResp]		"Result"
//	@Failure		400		{object}	coreresponse.ApiResponse[modelresponse.RegisterUserResp]		"Result"
//	@Router			/public/auth/register [post]
//	@Security		Bearer
func (handler *AuthHandler) Register(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()

	// Init Ctx
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	// Init Tracer
	tr := otel.Tracer("handler.AuthHandler")
	ctx, span := tr.Start(ctx, "Register()")
	defer span.End()

	// get form data
	requestData := &modelrequest.RegisterUserReq{}
	if err := fiberCtx.BodyParser(requestData); err != nil {
		errString := err.Error()
		handler.Log.Info("AuthHandler.Register()", "fiberCtx.BodyParser()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.RegisterUserResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusBadRequest,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	// Exec UseCase
	response, exc := handler.UseCase.Register(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AuthHandler.Register()", "UseCase.Register()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.RegisterUserResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}

	return fiberCtx.Status(fiber.StatusCreated).JSON(coreresponse.ApiResponse[modelresponse.RegisterUserResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusCreated,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// Login
//
// @Summary		Auth Login
// @Description	Auth Login
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		data		body		modelrequest.AuthUserReq					true		"Auth Request Parameter"
// @Success		201			{object}	coreresponse.ApiResponse[modelresponse.AuthUserResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.AuthUserResp]	"Result"
// @Router		/public/auth/login [post]
func (handler *AuthHandler) Login(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AuthHandler")
	ctx, span := tr.Start(ctx, "Login()")
	defer span.End()

	requestData := &modelrequest.AuthUserReq{}
	if err := fiberCtx.BodyParser(requestData); err != nil {
		errString := err.Error()
		handler.Log.Info("LoginHandler.Login()", "fiberCtx.BodyParser()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.AuthUserResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  fiber.StatusBadRequest,
			Error:   &errString,
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	// exec usecase
	response, exc := handler.UseCase.Login(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AuthHandler.Login()", "UseCase.Login()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.AuthUserResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.AuthUserResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// Activation
//
// @Summary		User Activation
// @Description	User Activation
// @Tags		Auth
// @Produce		json
// @Param		user_id		path		uint64		true	"User ID"
// @Success		200			{object}	coreresponse.ApiResponse[modelresponse.UserActivationResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.UserActivationResp]	"Result"
// @Router		/admin/auth/:user_id/activation [put]
// @Security	Bearer
func (handler *AuthHandler) Activation(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AuthHandler")
	ctx, span := tr.Start(ctx, "Activation()")
	defer span.End()

	requestData := &modelrequest.UserActivationReq{
		UserID: fiberCtx.Params("user_id"),
	}

	// exec usecase
	response, exc := handler.UseCase.Activation(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AuthHandler.Activation()", "UseCase.Activation()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.UserActivationResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.UserActivationResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// List
//
// @Summary		List User
// @Description	List User
// @Tags		Auth
// @Produce		json
// @Param		filter		query		string									false				"Search Parameter"
// @Param		sort		query		string									false				"Sorting Parameter"
// @Param		page		query		int										false				"Current Page"
// @Param		pageSize	query		int										false				"Page Size"
// @Success		200			{object}	coreresponse.ApiResponse[modelresponse.ListUserResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.ListUserResp]	"Result"
// @Router		/admin/auth [get]
// @Security	Bearer
func (handler *AuthHandler) List(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AuthHandler")
	ctx, span := tr.Start(ctx, "List()")
	defer span.End()

	// Get Param
	queryInfo, err := helpergenerator.GenerateQueryInfoPostgreSQL(fiberCtx)
	if err != nil {
		errString := "parameter query tidak valid"
		handler.Log.Info("List.List()", "helpergenerator.GenerateQueryInfoPostgreSQL()", "Info", err)
		return fiberCtx.Status(fiber.StatusBadRequest).JSON(coreresponse.ApiResponse[modelresponse.ListUserResp]{
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
	requestData := &modelrequest.ListUserReq{}
	requestData.QueryInfo = queryInfo

	// exec usecase
	response, exc := handler.UseCase.List(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AuthHandler.List()", "UseCase.List()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.ListUserResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.ListUserResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// DeActivation
//
// @Summary		User DeActivation
// @Description	User DeActivation
// @Tags		Auth
// @Produce		json
// @Param		user_id		path		uint64		true	"User ID"
// @Success		200			{object}	coreresponse.ApiResponse[modelresponse.UserDeActivationResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.UserDeActivationResp]	"Result"
// @Router		/admin/auth/:user_id/deactivation [put]
// @Security	Bearer
func (handler *AuthHandler) DeActivation(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AuthHandler")
	ctx, span := tr.Start(ctx, "DeActivation()")
	defer span.End()

	requestData := &modelrequest.UserDeActivationReq{
		UserID: fiberCtx.Params("user_id"),
	}

	// exec usecase
	response, exc := handler.UseCase.DeActivation(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AuthHandler.Activation()", "UseCase.Activation()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.UserDeActivationResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.UserDeActivationResp]{
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
// @Summary		Delete User
// @Description	Delete User
// @Tags		Auth
// @Produce		json
// @Param		user_id		path		uint64		true	"User ID"
// @Success		200			{object}	coreresponse.ApiResponse[modelresponse.DeleteUserResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.DeleteUserResp]	"Result"
// @Router		/admin/auth/:user_id [delete]
// @Security	Bearer
func (handler *AuthHandler) Delete(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AuthHandler")
	ctx, span := tr.Start(ctx, "Delete()")
	defer span.End()

	requestData := &modelrequest.DeleteUserReq{
		UserID: fiberCtx.Params("user_id"),
	}

	// exec usecase
	response, exc := handler.UseCase.Delete(ctx, requestData)
	if exc != nil {
		handler.Log.Info("AuthHandler.Delete()", "UseCase.Delete()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.DeleteUserResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.DeleteUserResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}

// Find
//
// @Summary		Find
// @Description	Find an Account by Token
// @Tags		Auth
// @Produce		json
// @Success		201			{object}	coreresponse.ApiResponse[modelresponse.AuthUserResp]	"Result"
// @Failure		400			{object}	coreresponse.ApiResponse[modelresponse.AuthUserResp]	"Result"
// @Router		/user [get]
// @Security	Bearer
func (handler *AuthHandler) Find(fiberCtx *fiber.Ctx) error {
	var timeIn = time.Now()
	ctx := helpergenerator.DefaultContextGenerator(fiberCtx)

	tr := otel.Tracer("handler.AuthHandler")
	ctx, span := tr.Start(ctx, "Find()")
	defer span.End()

	// exec usecase
	response, exc := handler.UseCase.Find(ctx)
	if exc != nil {
		handler.Log.Info("AuthHandler.Find()", "UseCase.Find()", "Info", exc.Error())
		return fiberCtx.Status(exc.GetHttpCode()).JSON(coreresponse.ApiResponse[modelresponse.FindUserResp]{
			Tin:     timeIn,
			Tout:    time.Now(),
			Success: false,
			Status:  exc.GetHttpCode(),
			Error:   exc.GetError(),
			Latency: helpergenerator.GetLatency(timeIn),
			Data:    nil,
		})
	}
	return fiberCtx.Status(fiber.StatusOK).JSON(coreresponse.ApiResponse[modelresponse.FindUserResp]{
		Tin:     timeIn,
		Tout:    time.Now(),
		Success: true,
		Status:  fiber.StatusOK,
		Error:   nil,
		Latency: helpergenerator.GetLatency(timeIn),
		Data:    &response,
	})
}
