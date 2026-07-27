package repository

import (
	"errors"
	"log/slog"
	"logisfy/internal/entity"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type AMRWeightConfigRepository struct {
	*Repository[entity.AMRWeightConfigEntity]
	Log *slog.Logger
}

func NewAMRWeightConfigRepository(
	log *slog.Logger,
) *AMRWeightConfigRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.AMRWeightConfigEntity{}).TableName(), "created_at")
	repo := NewRepositoryImpl[entity.AMRWeightConfigEntity](sqlGenerator)
	return &AMRWeightConfigRepository{
		Repository: repo,
		Log:        log,
	}
}

func (r *AMRWeightConfigRepository) FindByAMRID(tx *gorm.DB, amrID uint64) (result *entity.AMRWeightConfigEntity, err error) {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.AMRWeightConfigRepository")
	ctx, span := tr.Start(ctx, "FindByAMRID")
	defer span.End()

	if err = tx.Where("amr_id = ?", amrID).First(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		r.Log.Error("failed to find AMR weight config", "method", "AMRWeightConfigRepository.FindByAMRID()", "error", err.Error())
		return
	}
	return
}
