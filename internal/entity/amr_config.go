package entity

import (
	helperconverter "logisfy/helper/converter"
	modelrequest "logisfy/internal/model/request"
	modelresponse "logisfy/internal/model/response"
	"time"
)

type AMRConfigEntity struct {
	AMRConfigID             uint64    `gorm:"primaryKey;autoIncrement" json:"amr_config_id"`
	AMRID                   uint64    `gorm:"not null;index" json:"amr_id"`
	VDropVTM                float64   `gorm:"default:56" json:"v_drop_v_tm"`
	VDropVTR                float64   `gorm:"default:180" json:"v_drop_v_tr"`
	VDropITM                float64   `gorm:"default:0.5" json:"v_drop_i_tm"`
	VDropITR                float64   `gorm:"default:0.5" json:"v_drop_i_tr"`
	VLossVTM                float64   `gorm:"default:0" json:"v_loss_v_tm"`
	VLossVTR                float64   `gorm:"default:0" json:"v_loss_v_tr"`
	VLossITM                float64   `gorm:"default:0" json:"v_loss_i_tm"`
	VLossITR                float64   `gorm:"default:0" json:"v_loss_i_tr"`
	CosPhiKecilITM          float64   `gorm:"default:0.02" json:"cos_phi_kecil_i_tm"`
	CosPhiKecilITR          float64   `gorm:"default:0.02" json:"cos_phi_kecil_i_tr"`
	CosPhiKecilUpperLimitTM float64   `gorm:"default:1" json:"cos_phi_kecil_upper_limit_tm"`
	CosPhiKecilUpperLimitTR float64   `gorm:"default:1" json:"cos_phi_kecil_upper_limit_tr"`
	ILossITM                float64   `gorm:"default:-1" json:"i_loss_i_tm"`
	ILossITR                float64   `gorm:"default:-1" json:"i_loss_i_tr"`
	ILossIMaxTM             float64   `gorm:"default:0" json:"i_loss_i_max_tm"`
	ILossIMaxTR             float64   `gorm:"default:0" json:"i_loss_i_max_tr"`
	InGreaterIMaxInTM       float64   `gorm:"default:1" json:"in_greater_i_max_in_tm"`
	InGreaterIMaxInTR       float64   `gorm:"default:10" json:"in_greater_i_max_in_tr"`
	OverCurrentIMaxTM       float64   `gorm:"default:5" json:"over_current_i_max_tm"`
	OverCurrentIMaxTR       float64   `gorm:"default:5" json:"over_current_i_max_tr"`
	OverVoltageVMaxTM       float64   `gorm:"default:62" json:"over_voltage_v_max_tm"`
	OverVoltageVMaxTR       float64   `gorm:"default:241" json:"over_voltage_v_max_tr"`
	ReversePowerVTM         float64   `gorm:"default:0" json:"reverse_power_v_tm"`
	ReversePowerVTR         float64   `gorm:"default:0" json:"reverse_power_v_tr"`
	ReversePowerITM         float64   `gorm:"default:0.5" json:"reverse_power_i_tm"`
	ReversePowerITR         float64   `gorm:"default:0.7" json:"reverse_power_i_tr"`
	IUnbalanceTolTM         float64   `gorm:"default:0.5" json:"i_unbalance_tol_tm"`
	IUnbalanceTolTR         float64   `gorm:"default:0.5" json:"i_unbalance_tol_tr"`
	IUnbalanceITM           float64   `gorm:"default:0.5" json:"i_unbalance_i_tm"`
	IUnbalanceITR           float64   `gorm:"default:1" json:"i_unbalance_i_tr"`
	PLossI                  float64   `gorm:"default:0.5" json:"p_loss_i"`
	ILowVLowTM              float64   `gorm:"default:2" json:"i_low_v_low_tm"`
	ILowVLowTR              float64   `gorm:"default:8" json:"i_low_v_low_tr"`
	MinIndicatorAmount      float64   `gorm:"default:1" json:"min_indicator_amount"`
	MinWeight               float64   `gorm:"default:3" json:"min_weight"`
	NShowRecommendation     int       `gorm:"default:50" json:"n_show_recommendation"`
	CreatedAt               time.Time `gorm:"autoCreateTime;not null"`

	AMREntity AMREntity `gorm:"-"`
}

func (t AMRConfigEntity) TableName() string {
	return "amr_config"
}

func (t AMRConfigEntity) CheckFound() bool {
	return t.AMRConfigID > 0
}

func (t AMRConfigEntity) Create(req *modelrequest.CreateParamConfigAMRReq, amrID uint64) (amrConfigEntity *AMRConfigEntity) {
	amrConfigEntity = &AMRConfigEntity{
		AMRID: amrID,
	}
	if req.VDropVTM != nil {
		amrConfigEntity.VDropVTM = *req.VDropVTM
	}
	if req.VDropVTR != nil {
		amrConfigEntity.VDropVTR = *req.VDropVTR
	}
	if req.VDropITM != nil {
		amrConfigEntity.VDropITM = *req.VDropITM
	}
	if req.VDropITR != nil {
		amrConfigEntity.VDropITR = *req.VDropITR
	}
	if req.VLossVTM != nil {
		amrConfigEntity.VLossVTM = *req.VLossVTM
	}
	if req.VLossVTR != nil {
		amrConfigEntity.VLossVTR = *req.VLossVTR
	}
	if req.VLossITM != nil {
		amrConfigEntity.VLossITM = *req.VLossITM
	}
	if req.VLossITR != nil {
		amrConfigEntity.VLossITR = *req.VLossITR
	}
	if req.ILossVTM != nil {
		amrConfigEntity.ILossIMaxTM = *req.ILossVTM
	}
	if req.ILossVTR != nil {
		amrConfigEntity.ILossIMaxTR = *req.ILossVTR
	}
	if req.ILossITM != nil {
		amrConfigEntity.ILossITM = *req.ILossITM
	}
	if req.ILossITR != nil {
		amrConfigEntity.ILossITR = *req.ILossITR
	}
	if req.CosPhiKecilITM != nil {
		amrConfigEntity.CosPhiKecilITM = *req.CosPhiKecilITM
	}
	if req.CosPhiKecilITR != nil {
		amrConfigEntity.CosPhiKecilITR = *req.CosPhiKecilITR
	}
	if req.CosPhiKecilIMaxTM != nil {
		amrConfigEntity.CosPhiKecilUpperLimitTM = *req.CosPhiKecilIMaxTM
	}
	if req.CosPhiKecilIMaxTR != nil {
		amrConfigEntity.CosPhiKecilUpperLimitTR = *req.CosPhiKecilIMaxTR
	}
	if req.InGreaterIMaxInTM != nil {
		amrConfigEntity.InGreaterIMaxInTM = *req.InGreaterIMaxInTM
	}
	if req.InGreaterIMaxInTR != nil {
		amrConfigEntity.InGreaterIMaxInTR = *req.InGreaterIMaxInTR
	}
	if req.OverCurrentIMaxTM != nil {
		amrConfigEntity.OverCurrentIMaxTM = *req.OverCurrentIMaxTM
	}
	if req.OverCurrentIMaxTR != nil {
		amrConfigEntity.OverCurrentIMaxTR = *req.OverCurrentIMaxTR
	}
	if req.OverVoltageVMaxTM != nil {
		amrConfigEntity.OverVoltageVMaxTM = *req.OverVoltageVMaxTM
	}
	if req.OverVoltageVMaxTR != nil {
		amrConfigEntity.OverVoltageVMaxTR = *req.OverVoltageVMaxTR
	}
	if req.ReversePowerVTM != nil {
		amrConfigEntity.ReversePowerVTM = *req.ReversePowerVTM
	}
	if req.ReversePowerVTR != nil {
		amrConfigEntity.ReversePowerVTR = *req.ReversePowerVTR
	}
	if req.ReversePowerITM != nil {
		amrConfigEntity.ReversePowerITM = *req.ReversePowerITM
	}
	if req.ReversePowerITR != nil {
		amrConfigEntity.ReversePowerITR = *req.ReversePowerITR
	}
	if req.IUnbalanceTolTM != nil {
		amrConfigEntity.IUnbalanceTolTM = *req.IUnbalanceTolTM
	}
	if req.IUnbalanceTolTR != nil {
		amrConfigEntity.IUnbalanceTolTR = *req.IUnbalanceTolTR
	}
	if req.IUnbalanceITM != nil {
		amrConfigEntity.IUnbalanceITM = *req.IUnbalanceITM
	}
	if req.IUnbalanceITR != nil {
		amrConfigEntity.IUnbalanceITR = *req.IUnbalanceITR
	}
	if req.PLossI != nil {
		amrConfigEntity.PLossI = *req.PLossI
	}
	if req.ILowVLowTM != nil {
		amrConfigEntity.ILowVLowTM = *req.ILowVLowTM
	}
	if req.ILowVLowTR != nil {
		amrConfigEntity.ILowVLowTR = *req.ILowVLowTR
	}
	if req.MinIndicatorAmount != nil {
		amrConfigEntity.MinIndicatorAmount = *req.MinIndicatorAmount
	}
	if req.MinWeight != nil {
		amrConfigEntity.MinWeight = *req.MinWeight
	}
	if req.NShowRecommendation != nil {
		amrConfigEntity.NShowRecommendation = *req.NShowRecommendation
	}
	return
}

func (t AMRConfigEntity) ConvertToResp(paramConfigEntity *AMRConfigEntity) (resp modelresponse.FindAMRParamConfigResp) {
	createdAtStr := helperconverter.ConvertTimeToString(&paramConfigEntity.CreatedAt)

	resp.AMRID = paramConfigEntity.AMRID
	resp.AMRConfigID = paramConfigEntity.AMRConfigID
	resp.VDropVTM = paramConfigEntity.VDropVTM
	resp.VDropVTR = paramConfigEntity.VDropVTR
	resp.VDropITM = paramConfigEntity.VDropITM
	resp.VDropITR = paramConfigEntity.VDropITR
	resp.VLossVTM = paramConfigEntity.VLossVTM
	resp.VLossVTR = paramConfigEntity.VLossVTR
	resp.VLossITM = paramConfigEntity.VLossITM
	resp.VLossITR = paramConfigEntity.VLossITR
	resp.CosPhiKecilITM = paramConfigEntity.CosPhiKecilITM
	resp.CosPhiKecilITR = paramConfigEntity.CosPhiKecilITR
	resp.CosPhiKecilUpperLimitTM = paramConfigEntity.CosPhiKecilUpperLimitTM
	resp.CosPhiKecilUpperLimitTR = paramConfigEntity.CosPhiKecilUpperLimitTR
	resp.ILossITM = paramConfigEntity.ILossITM
	resp.ILossITR = paramConfigEntity.ILossITR
	resp.ILossIMaxTM = paramConfigEntity.ILossIMaxTM
	resp.ILossIMaxTR = paramConfigEntity.ILossIMaxTR
	resp.InGreaterIMaxInTM = paramConfigEntity.InGreaterIMaxInTM
	resp.InGreaterIMaxInTR = paramConfigEntity.InGreaterIMaxInTR
	resp.OverCurrentIMaxTM = paramConfigEntity.OverCurrentIMaxTM
	resp.OverCurrentIMaxTR = paramConfigEntity.OverCurrentIMaxTR
	resp.OverVoltageVMaxTM = paramConfigEntity.OverVoltageVMaxTM
	resp.OverVoltageVMaxTR = paramConfigEntity.OverVoltageVMaxTR
	resp.ReversePowerVTM = paramConfigEntity.ReversePowerVTM
	resp.ReversePowerVTR = paramConfigEntity.ReversePowerVTR
	resp.ReversePowerITM = paramConfigEntity.ReversePowerITM
	resp.ReversePowerITR = paramConfigEntity.ReversePowerITR
	resp.IUnbalanceTolTM = paramConfigEntity.IUnbalanceTolTM
	resp.IUnbalanceTolTR = paramConfigEntity.IUnbalanceTolTR
	resp.IUnbalanceITM = paramConfigEntity.IUnbalanceITM
	resp.IUnbalanceITR = paramConfigEntity.IUnbalanceITR
	resp.PLossI = paramConfigEntity.PLossI
	resp.ILowVLowTM = paramConfigEntity.ILowVLowTM
	resp.ILowVLowTR = paramConfigEntity.ILowVLowTR
	resp.MinIndicatorAmount = paramConfigEntity.MinIndicatorAmount
	resp.MinWeight = paramConfigEntity.MinWeight
	resp.NShowRecommendation = paramConfigEntity.NShowRecommendation
	resp.CreatedAt = createdAtStr
	return
}
