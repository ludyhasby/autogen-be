package request

import (
	"io"
	"logisfy/core"
)

type UploadAMRReq struct {
	File      io.Reader `json:"-"`
	Size      int64     `json:"-"`
	Filename  string    `json:"-"`
	Extension string    `validate:"required,oneof=.xlsx .xls" json:"-"`
}
type CreateParamConfigAMRReq struct {
	AMRID               string   `json:"-"`
	VDropVTM            *float64 `json:"v_drop_v_tm"`
	VDropVTR            *float64 `json:"v_drop_v_tr"`
	VDropITM            *float64 `json:"v_drop_i_tm"`
	VDropITR            *float64 `json:"v_drop_i_tr"`
	VLossVTM            *float64 `json:"v_loss_v_tm"`
	VLossVTR            *float64 `json:"v_loss_v_tr"`
	VLossITM            *float64 `json:"v_loss_i_tm"`
	VLossITR            *float64 `json:"v_loss_i_tr"`
	ILossVTM            *float64 `json:"i_loss_v_tm"`
	ILossVTR            *float64 `json:"i_loss_v_tr"`
	ILossITM            *float64 `json:"i_loss_i_tm"`
	ILossITR            *float64 `json:"i_loss_i_tr"`
	CosPhiKecilITM      *float64 `json:"cos_phi_kecil_i_tm"`
	CosPhiKecilITR      *float64 `json:"cos_phi_kecil_i_tr"`
	CosPhiKecilIMaxTM   *float64 `json:"cos_phi_kecil_i_max_tm"`
	CosPhiKecilIMaxTR   *float64 `json:"cos_phi_kecil_i_max_tr"`
	InGreaterIMaxInTM   *float64 `json:"in_greater_i_max_in_tm"`
	InGreaterIMaxInTR   *float64 `json:"in_greater_i_max_in_tr"`
	OverCurrentIMaxTM   *float64 `json:"over_current_i_max_tm"`
	OverCurrentIMaxTR   *float64 `json:"over_current_i_max_tr"`
	OverVoltageVMaxTM   *float64 `json:"over_voltage_v_max_tm"`
	OverVoltageVMaxTR   *float64 `json:"over_voltage_v_max_tr"`
	ReversePowerVTM     *float64 `json:"reverse_power_v_tm"`
	ReversePowerVTR     *float64 `json:"reverse_power_v_tr"`
	ReversePowerITM     *float64 `json:"reverse_power_i_tm"`
	ReversePowerITR     *float64 `json:"reverse_power_i_tr"`
	IUnbalanceTolTM     *float64 `json:"i_unbalance_tol_tm"`
	IUnbalanceTolTR     *float64 `json:"i_unbalance_tol_tr"`
	IUnbalanceITM       *float64 `json:"i_unbalance_i_tm"`
	IUnbalanceITR       *float64 `json:"i_unbalance_i_tr"`
	PLossI              *float64 `json:"p_loss_i"`
	ILowVLowTM          *float64 `json:"i_low_v_low_tm"`
	ILowVLowTR          *float64 `json:"i_low_v_low_tr"`
	MinIndicatorAmount  *float64 `json:"min_indicator_amount"`
	MinWeight           *float64 `json:"min_weight"`
	NShowRecommendation *int     `validate:"omitempty,min=1,max=2500" msg:"jumlah rekomendasi harus di rentang 1-2500" json:"n_show_recommendation"`
}
type CreateWeightConfigAMRReq struct {
	AMRID         string   `json:"-"`
	VIndirectDrop *float64 `json:"v_indirect_drop"`
	VDirectDrop   *float64 `json:"v_direct_drop"`
	VLoss         *float64 `json:"v_loss"`
	CosPhiKecil   *float64 `json:"cos_phi_kecil"`
	ILoss         *float64 `json:"i_loss"`
	ReversePower  *float64 `json:"reverse_power"`
	UnbalanceI    *float64 `json:"unbalance_i"`
	ILowVLow      *float64 `json:"i_low_v_low"`
	CurrentLoop   *float64 `json:"current_loop"`
	ActivePLoss   *float64 `json:"active_p_loss"`
	Freeze        *float64 `json:"freeze"`
}

type ListAMRReq struct {
	QueryInfo core.QueryInfo `json:"-"`
}
type GenerateReportAMRReq struct {
	AMRID string `json:"-"`
}
type ListReportAMRReq struct {
	AMRID     string         `json:"-"`
	QueryInfo core.QueryInfo `json:"-"`
}
type FindAMRReq struct {
	AMRID string `json:"-" validate:"required"`
}
type FindAMRParamConfigReq struct {
	AMRID string `json:"-" validate:"required"`
}
type FindAMRWeightConfigReq struct {
	AMRID string `json:"-" validate:"required"`
}
type DeleteAMRReq struct {
	AMRID string `json:"-" validate:"required"`
}
type SummaryAMRReq struct {
	AMRID string `json:"-"`
}
type ExportAMRReq struct {
	AMRID     string         `json:"-"`
	QueryInfo core.QueryInfo `json:"-"`
}
type ExportRecommendationAMRReq struct {
	AMRID     string         `json:"-"`
	QueryInfo core.QueryInfo `json:"-"`
}
