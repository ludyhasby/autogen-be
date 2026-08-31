package entity

import (
	modelrequest "logisfy/internal/model/request"
	"time"

	"github.com/google/uuid"
)

type PasswordResetTokenEntity struct {
	PasswordResetTokenID uint64    `gorm:"primaryKey;autoIncrement" json:"password_reset_token_id"`
	UserID               uint64    `gorm:"index;not null" json:"user_id"`
	Email                string    `gorm:"not null" json:"email"`
	SessionKey           string    `gorm:"uniqueIndex;not null" json:"session_key"`
	IsActive             *bool     `gorm:"not null" json:"is_active"`
	ExpiresAt            time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt            time.Time `gorm:"autoCreateTime;not null"`
}

func (t PasswordResetTokenEntity) TableName() string {
	return "password_reset_token"
}

func (t *PasswordResetTokenEntity) CheckFound() bool {
	return t != nil && t.PasswordResetTokenID > 0
}

func (t PasswordResetTokenEntity) Create(req *modelrequest.ForgotPasswordReq, userID uint64, isActive bool) *PasswordResetTokenEntity {
	return &PasswordResetTokenEntity{
		UserID:     userID,
		Email:      req.Email,
		SessionKey: uuid.New().String(),
		IsActive:   &isActive,
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}
}
