package entity

import (
	coreenum "logisfy/core/enum"
	helperconverter "logisfy/helper/converter"
	modelrequest "logisfy/internal/model/request"
	modelresponse "logisfy/internal/model/response"
	"time"
)

type AMREntity struct {
	AMRID         uint64                       `gorm:"primaryKey;autoIncrement" json:"amr_id"`
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

func (t AMREntity) TableName() string {
	return "amr"
}

func (t AMREntity) CheckFound() bool {
	return t.AMRID > 0
}

func (t AMREntity) Create(userID uint64, location *time.Location, deletionHours int, req *modelrequest.UploadAMRReq, dataCount int64, stageProcess coreenum.CTXEnumStageProcess) *AMREntity {
	now := time.Now().In(location)
	return &AMREntity{
		UserID:        userID,
		Filename:      req.Filename,
		Extension:     coreenum.CTXEnumExtension(req.Extension),
		RowNumbers:    dataCount,
		ReadDate:      time.Now(),
		StageProcess:  stageProcess,
		AutoDeletedAt: now.Add(time.Duration(deletionHours) * time.Hour),
	}

}

func (t AMREntity) ConvertToList(entities []*AMREntity) (resp []modelresponse.ListAMR) {
	for _, entity := range entities {
		readDateStr := helperconverter.ConvertTimeToString(&entity.ReadDate)
		createdAtStr := helperconverter.ConvertTimeToString(&entity.CreatedAt)
		updatedAtStr := helperconverter.ConvertTimeToString(&entity.UpdatedAt)
		autoDeletedAtStr := helperconverter.ConvertTimeToString(&entity.AutoDeletedAt)
		resp = append(resp, modelresponse.ListAMR{
			AMRID:         entity.AMRID,
			Filename:      entity.Filename,
			Extension:     entity.Extension,
			RowNumbers:    entity.RowNumbers,
			ReadDate:      readDateStr,
			StageProcess:  entity.StageProcess,
			CreatedAt:     createdAtStr,
			UpdatedAt:     updatedAtStr,
			AutoDeletedAt: autoDeletedAtStr,
			FailedReason:  entity.FailedReason,
		})
	}
	return
}
