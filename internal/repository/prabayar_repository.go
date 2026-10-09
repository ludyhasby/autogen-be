package repository

import (
	"errors"
	"log/slog"
	"logisfy/internal/entity"
	modelrequest "logisfy/internal/model/request"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type PrabayarRepository struct {
	*Repository[entity.PrabayarEntity]
	Log *slog.Logger
}

func NewPrabayarRepository(
	log *slog.Logger,
) *PrabayarRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.PrabayarEntity{}).TableName(), "created_at")
	repo := NewRepositoryImpl[entity.PrabayarEntity](sqlGenerator)
	return &PrabayarRepository{
		Repository: repo,
		Log:        log,
	}
}

func (r *PrabayarRepository) FindByFilter(tx *gorm.DB, prabayarFilter modelrequest.PrabayarFilter) (results []*entity.PrabayarEntity, err error) {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.PrabayarRepository")
	ctx, span := tr.Start(ctx, "FindOneByFilter")
	defer span.End()

	if err = r.parsePrabayarFilter(tx, prabayarFilter).Find(&results).Error; err != nil {
		r.Log.Error("failed to find one prabayar by filter", "method", "PrabayarRepository.FindOneByFilter()", "error", err.Error())
		return nil, err
	}

	return
}

func (r *PrabayarRepository) FindOneByFilter(tx *gorm.DB, prabayarFilter modelrequest.PrabayarFilter) (result *entity.PrabayarEntity, err error) {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.PrabayarRepository")
	ctx, span := tr.Start(ctx, "FindOneByFilter")
	defer span.End()

	if err = r.parsePrabayarFilter(tx, prabayarFilter).Take(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		r.Log.Error("PrabayarRepository.FindOneByFilter()", "Exec()", "error", err.Error())
		return nil, err
	}

	return
}

func (r *PrabayarRepository) parsePrabayarFilter(tx *gorm.DB, filter modelrequest.PrabayarFilter) *gorm.DB {
	if filter.UserID != nil {
		tx = tx.Where("user_id = ?", *filter.UserID)
	}
	if filter.PrabayarID != nil {
		tx = tx.Where("prabayar_id = ?", *filter.PrabayarID)
	}
	if len(filter.StageProcesses) != 0 {
		tx = tx.Where("stage_process IN (?)", filter.StageProcesses)
	}
	return tx
}
