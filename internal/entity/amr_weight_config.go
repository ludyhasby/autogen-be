package entity

import (
	modelrequest "logisfy/internal/model/request"
	"time"
)

type AMRWeightConfigEntity struct {
	AMRWeightConfigID uint64    `gorm:"primaryKey;autoIncrement" json:"amr_weight_config_id"`
	AMRID             uint64    `gorm:"not null;index" json:"amr_id"`
	VIndirectDrop     float64   `gorm:"default:20" json:"v_indirect_drop"`
	VDirectDrop       float64   `gorm:"default:7" json:"v_direct_drop"`
	VLoss             float64   `gorm:"default:7" json:"v_loss"`
	CosPhiKecil       float64   `gorm:"default:10" json:"cos_phi_kecil"`
	ILoss             float64   `gorm:"default:1" json:"i_loss"`
	InGreatedImax     float64   `gorm:"default:10" json:"in_greated_imax"`
	OverCurrent       float64   `gorm:"default:15" json:"over_current"`
	OverVoltage       float64   `gorm:"default:1" json:"over_voltage"`
	ReversePower      float64   `gorm:"default:7" json:"reverse_power"`
	UnbalanceI        float64   `gorm:"default:3" json:"unbalance_i"`
	ILowVLow          float64   `gorm:"default:4" json:"i_low_v_low"`
	CurrentLoop       float64   `gorm:"default:20" json:"current_loop"`
	ActivePLoss       float64   `gorm:"default:7" json:"active_p_loss"`
	Freeze            float64   `gorm:"default:20" json:"freeze"`
	CreatedAt         time.Time `gorm:"autoCreateTime;not null"`

	AMREntity AMREntity `gorm:"-"`
}

func (t AMRWeightConfigEntity) TableName() string {
	return "amr_weight_config"
}

func (t AMRWeightConfigEntity) CheckFound() bool {
	return t.AMRWeightConfigID > 0
}

func (t AMRWeightConfigEntity) Create(req *modelrequest.CreateWeightConfigAMRReq, amrID uint64) (amrWeightConfigEntity *AMRWeightConfigEntity) {
	amrWeightConfigEntity = &AMRWeightConfigEntity{
		AMRID: amrID,
	}
	if req.VIndirectDrop != nil {
		amrWeightConfigEntity.VIndirectDrop = *req.VIndirectDrop
	}
	if req.VDirectDrop != nil {
		amrWeightConfigEntity.VDirectDrop = *req.VDirectDrop
	}
	if req.VLoss != nil {
		amrWeightConfigEntity.VLoss = *req.VLoss
	}
	if req.CosPhiKecil != nil {
		amrWeightConfigEntity.CosPhiKecil = *req.CosPhiKecil
	}
	if req.ILoss != nil {
		amrWeightConfigEntity.ILoss = *req.ILoss
	}
	if req.ReversePower != nil {
		amrWeightConfigEntity.ReversePower = *req.ReversePower
	}
	if req.UnbalanceI != nil {
		amrWeightConfigEntity.UnbalanceI = *req.UnbalanceI
	}
	if req.ILowVLow != nil {
		amrWeightConfigEntity.ILowVLow = *req.ILowVLow
	}
	if req.CurrentLoop != nil {
		amrWeightConfigEntity.CurrentLoop = *req.CurrentLoop
	}
	if req.ActivePLoss != nil {
		amrWeightConfigEntity.ActivePLoss = *req.ActivePLoss
	}
	if req.Freeze != nil {
		amrWeightConfigEntity.Freeze = *req.Freeze
	}
	return
}
