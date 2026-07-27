package usecase

import (
	"context"
	"log/slog"
	helperexception "logisfy/helper/exception"
	"logisfy/internal/entity"
	modelrequest "logisfy/internal/model/request"
	modelresponse "logisfy/internal/model/response"
	"logisfy/internal/repository"

	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type NewsUseCase struct {
	DB             *gorm.DB
	Log            *slog.Logger
	Validate       *validator.Validate
	NewsRepository *repository.NewsRepository
}

func NewNewsUseCase(db *gorm.DB, log *slog.Logger, validate *validator.Validate, newsRepository *repository.NewsRepository) *NewsUseCase {
	return &NewsUseCase{
		DB:             db,
		Log:            log,
		Validate:       validate,
		NewsRepository: newsRepository,
	}
}

func (u *NewsUseCase) List(ctx context.Context, req *modelrequest.ListNewsReq) (resp modelresponse.ListNewsResp, exc *helperexception.Exception) {
	// init tracer
	tr := otel.Tracer("useCase.NewsUseCase")
	ctx, span := tr.Start(ctx, "List()")
	defer span.End()

	// Init DB
	tx := u.DB.WithContext(ctx)

	// exec repo
	entityNewsList, items, totalPages, size, err := u.NewsRepository.List(tx, req.QueryInfo)
	if err != nil {
		u.Log.Info("NewsUseCase.List()", "NewsRepository.List()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses list", err)
		return
	}

	// resp
	resp.List = (&entity.NewsEntity{}).ConvertToList(entityNewsList)
	resp.TotalItems = items
	resp.Page = req.QueryInfo.SelectParameter.PageDescriptor.PageIndex
	resp.PageSize = size
	resp.TotalPages = totalPages
	return
}
