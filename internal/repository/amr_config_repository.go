package repository

import (
	"errors"
	"log/slog"
	"logisfy/internal/entity"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type AMRConfigRepository struct {
	*Repository[entity.AMRConfigEntity]
	Log *slog.Logger
}

func NewAMRConfigRepository(
	log *slog.Logger,
) *AMRConfigRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.AMRConfigEntity{}).TableName(), "created_at")
	repo := NewRepositoryImpl[entity.AMRConfigEntity](sqlGenerator)
	return &AMRConfigRepository{
		Repository: repo,
		Log:        log,
	}
}

func (r *AMRConfigRepository) FindByAMRID(tx *gorm.DB, amrID uint64) (result *entity.AMRConfigEntity, err error) {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.AMRConfigRepository")
	ctx, span := tr.Start(ctx, "FindByAMRID")
	defer span.End()

	if err = tx.Where("amr_id = ?", amrID).First(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		r.Log.Error("failed to find AMR param config", "method", "AMRConfigRepository.FindByAMRID()", "error", err.Error())
		return
	}
	return
}
