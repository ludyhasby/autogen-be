package repository

import (
	"errors"
	"log/slog"
	"logisfy/core"
	"logisfy/internal/entity"
	"math"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type AMRRepository struct {
	*Repository[entity.AMREntity]
	Log *slog.Logger
}

func NewAMRRepository(
	log *slog.Logger,
) *AMRRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.AMREntity{}).TableName(), "created_at")
	repo := NewRepositoryImpl[entity.AMREntity](sqlGenerator)
	return &AMRRepository{
		Repository: repo,
		Log:        log,
	}
}

func (r *AMRRepository) Find(tx *gorm.DB, amrID uint64) (result *entity.AMREntity, err error) {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.AMRRepository")
	ctx, span := tr.Start(ctx, "Find")
	defer span.End()

	if err = tx.Where("amr_id = ?", amrID).First(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		r.Log.Error("failed to find AMR", "method", "AMRRepository.Find()", "error", err.Error())
		return
	}
	return
}

func (r *AMRRepository) FindByUserID(tx *gorm.DB, userID uint64) (results []*entity.AMREntity, err error) {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.AMRRepository")
	ctx, span := tr.Start(ctx, "FindByUserID")
	defer span.End()

	if err = tx.Where("user_id = ?", userID).Find(&results).Error; err != nil {
		r.Log.Error("failed to find AMR by by user ID", "method", "AMRRepository.FindByUserID()", "error", err.Error())
		return nil, err
	}
	return
}

func (r *AMRRepository) List(tx *gorm.DB, param core.QueryInfo) (results []entity.AMREntity, totalItems int32, totalPages int32, pageSize int32, err error) {
	ctx := tx.Statement.Context

	// init tracer
	tr := otel.Tracer("repository.AMRRepository")
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

	sqlQuery, sqlQueryCount := r.queryGenerator.Generate(param.SelectParameter, (&entity.AMREntity{}).TableName())

	queryResult := []entity.AMREntity{}
	var queryTotal int32

	err = tx.Raw(sqlQuery).Scan(&queryResult).Error
	if err != nil {
		r.Log.Error("failed get list amr", "method", "AMRRepository.List()", "sub_method", "tx.Raw()", "Err", err.Error())
		return
	}
	err = tx.Raw(sqlQueryCount).Scan(&queryTotal).Error
	if err != nil {
		r.Log.Error("failed get list amr", "method", "AMRRepository.List()", "sub_method", "tx.Raw()", "Err", err.Error())
		return
	}

	// output
	results = queryResult
	totalItems = queryTotal
	pageSize = param.SelectParameter.PageDescriptor.PageSize
	totalPages = int32(math.Ceil(float64(totalItems) / float64(pageSize)))
	return
}
