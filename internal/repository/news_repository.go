package repository

import (
	"log/slog"
	"logisfy/core"
	"logisfy/internal/entity"
	"math"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type NewsRepository struct {
	*Repository[entity.NewsEntity]
	Log *slog.Logger
}

func NewNewsRepository(
	log *slog.Logger,
) *NewsRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.NewsEntity{}).TableName(), "published_at")
	repo := NewRepositoryImpl[entity.NewsEntity](sqlGenerator)
	return &NewsRepository{
		Repository: repo,
		Log:        log,
	}
}

func (r *NewsRepository) List(tx *gorm.DB, param core.QueryInfo) (results []entity.NewsEntity, totalItems int32, totalPages int32, pageSize int32, err error) {
	ctx := tx.Statement.Context

	// init tracer
	tr := otel.Tracer("repository.NewsRepository")
	ctx, span := tr.Start(ctx, "List()")
	defer span.End()

	// default sort
	sortDefault := dancok.SortDescriptor{
		FieldName:     r.queryGenerator.DefaultFieldForSort,
		SortDirection: dancok.Descending,
	}
	if len(param.SelectParameter.SortDescriptors) == 0 {
		param.SelectParameter.SortDescriptors = append(param.SelectParameter.SortDescriptors, sortDefault)
	}

	sqlQuery, sqlQueryCount := r.queryGenerator.Generate(param.SelectParameter, (&entity.NewsEntity{}).TableName())

	queryResult := []entity.NewsEntity{}
	var queryTotal int32

	err = tx.Raw(sqlQuery).Scan(&queryResult).Error
	if err != nil {
		r.Log.Error("failed get list news", "method", "NewsRepository.List()", "sub_method", "tx.Raw()", "Err", err.Error())
		return
	}
	err = tx.Raw(sqlQueryCount).Scan(&queryTotal).Error
	if err != nil {
		r.Log.Error("failed get list news", "method", "NewsRepository.List()", "sub_method", "tx.Raw()", "Err", err.Error())
		return
	}

	// output
	results = queryResult
	totalItems = queryTotal
	pageSize = param.SelectParameter.PageDescriptor.PageSize
	totalPages = int32(math.Ceil(float64(totalItems) / float64(pageSize)))
	return
}
