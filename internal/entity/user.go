package entity

import (
	coreenum "logisfy/core/enum"
	helperconverter "logisfy/helper/converter"
	helperhash "logisfy/helper/hash"
	modelrequest "logisfy/internal/model/request"
	modelresponse "logisfy/internal/model/response"
	"time"

	"gorm.io/gorm"
)

type UserEntity struct {
	UserID      uint64               `gorm:"primaryKey;autoIncrement" json:"user_id"`
	Role        coreenum.CTXEnumRole `gorm:"not null;default:USER" json:"role"`
	Name        string               `gorm:"not null" json:"name"`
	Email       string               `gorm:"not null" json:"email"`
	Password    string               `gorm:"not null" json:"password"`
	UP3         string               `gorm:"null" json:"UP3"`
	UnitInduk   string               `gorm:"null" json:"UI"`
	IsActive    bool                 `gorm:"not null;default:false" json:"is_active"`
	CreatedAt   time.Time            `gorm:"autoCreateTime;not null"`
	UpdatedAt   time.Time            `gorm:"autoUpdateTime;not null"`
	LastLoginAt *time.Time           `gorm:"null"`
	DeletedAt   gorm.DeletedAt       `gorm:"index"`
}

func (t UserEntity) TableName() string {
	return "users"
}

func (t *UserEntity) CheckFound() bool {
	return t != nil && t.UserID > 0
}

func (t UserEntity) Create(req *modelrequest.RegisterUserReq, role coreenum.CTXEnumRole) *UserEntity {
	return &UserEntity{
		Role:      role,
		Name:      req.Name,
		Email:     req.Email,
		UP3:       req.UP3,
		UnitInduk: req.UnitInduk,
		Password:  helperhash.HashPassword(req.Password),
		IsActive:  false,
	}
}

func (t UserEntity) ConvertToList(entities []UserEntity) (resp []modelresponse.ListUser) {
	for _, entity := range entities {
		oneUser := modelresponse.ListUser{
			UserID:    entity.UserID,
			Name:      entity.Name,
			Email:     entity.Email,
			UP3:       entity.UP3,
			UnitInduk: entity.UnitInduk,
			IsActive:  entity.IsActive,
			CreatedAt: helperconverter.ConvertTimeToString(&entity.CreatedAt),
		}
		if entity.LastLoginAt != nil {
			oneUser.LastLogin = helperconverter.ConvertTimeToString(entity.LastLoginAt)
		}
		resp = append(resp, oneUser)
	}
	return
}
