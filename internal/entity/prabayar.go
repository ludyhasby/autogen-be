package entity

import (
	coreenum "logisfy/core/enum"
	modelrequest "logisfy/internal/model/request"
	"time"
)

type PrabayarEntity struct {
	PrabayarID    uint64                       `gorm:"primaryKey;autoIncrement" json:"prabayar_id"`
	UserID        uint64                       `gorm:"not null;index" json:"user_id"`
	Filename      string                       `gorm:"not null" json:"filename"`
	Extension     coreenum.CTXEnumExtension    `gorm:"not null" json:"extension"`
	RowNumbers    int64                        `gorm:"not null" json:"row_numbers"`
	ReadDate      time.Time                    `gorm:"not null" json:"read_date"`
	StageProcess  coreenum.CTXEnumStageProcess `gorm:"not null;default:CONFIG_SETTING" json:"stage_process"`
	CreatedAt     time.Time                    `gorm:"autoCreateTime;not null"`
	UpdatedAt     time.Time                    `gorm:"autoUpdateTime;not null"`
	AutoDeletedAt time.Time                    `gorm:"null;index" json:"auto_deleted_at"`
	FailedReason  string                       `gorm:"null" json:"failed_reason"`

	UserEntity UserEntity `gorm:"-"`
}

func (t PrabayarEntity) TableName() string {
	return "prabayar"
}

func (t PrabayarEntity) CheckFound() bool {
	return t.PrabayarID > 0
}

func (t PrabayarEntity) Create(userID uint64, location *time.Location, deletionHours int, req *modelrequest.UploadPrabayarReq, dataCount int64, stageProcess coreenum.CTXEnumStageProcess) *PrabayarEntity {
	now := time.Now().In(location)
	return &PrabayarEntity{
		UserID:        userID,
		Filename:      req.Filename,
		Extension:     coreenum.CTXEnumExtension(req.Extension),
		RowNumbers:    dataCount,
		ReadDate:      time.Now(),
		StageProcess:  stageProcess,
		AutoDeletedAt: now.Add(time.Duration(deletionHours) * time.Hour),
	}
}
