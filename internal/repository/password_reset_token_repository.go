package repository

import (
	"log/slog"
	"logisfy/internal/entity"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type PasswordResetTokenRepository struct {
	*Repository[entity.PasswordResetTokenEntity]
	Log *slog.Logger
}

func NewPasswordResetTokenRepository(
	log *slog.Logger,
) *PasswordResetTokenRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.PasswordResetTokenEntity{}).TableName(), "created_at")
	repo := NewRepositoryImpl[entity.PasswordResetTokenEntity](sqlGenerator)
	return &PasswordResetTokenRepository{
		Repository: repo,
		Log:        log,
	}
}

func (r *PasswordResetTokenRepository) UpdatePasswordResetTokenByUserID(
	tx *gorm.DB,
	userID uint64,
	isActive bool,
) error {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.PasswordResetTokenRepository")
	ctx, span := tr.Start(ctx, "UpdatePasswordResetTokenByUserID")
	defer span.End()

	err := tx.WithContext(ctx).
		Model(&entity.PasswordResetTokenEntity{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Update("is_active", isActive).Error

	if err != nil {
		r.Log.Error(
			"failed to update password reset token",
			"method", "PasswordResetTokenRepository.UpdatePasswordResetTokenByUserID",
			"sub_method", "tx.Model().Where().Update()",
			"user_id", userID,
			"is_active", isActive,
			"error", err,
		)
		return err
	}
	return nil
}
