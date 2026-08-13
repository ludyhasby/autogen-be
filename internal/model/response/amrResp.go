package response

import (
	coreenum "logisfy/core/enum"
)

type UploadAMRResp struct {
	AMRID     uint64 `json:"amr_id"`
	DataCount int64  `json:"data_count"`
}

type CreateParamConfigAMRResp struct {
	AMRConfigID uint64 `json:"amr_config_id"`
}
type CreateWeightConfigAMRResp struct {
	AMRWeightConfigID uint64 `json:"amr_weight_config_id"`
}
type ListAMRResp struct {
	List       []ListAMR `json:"list"`
	TotalItems int32     `json:"total_items"`
	TotalPages int32     `json:"total_pages"`
	Page       int32     `json:"page"`
	PageSize   int32     `json:"page_size"`
}

type ListAMR struct {
	AMRID         uint64                       `json:"amr_id"`
	Filename      string                       `json:"filename"`
	Extension     coreenum.CTXEnumExtension    `json:"extension"`
	RowNumbers    int64                        `json:"row_numbers"`
	ReadDate      string                       `json:"read_date"`
	StageProcess  coreenum.CTXEnumStageProcess `json:"stage_process"`
	CreatedAt     string                       `json:"created_at"`
	UpdatedAt     string                       `json:"updated_at"`
	AutoDeletedAt string                       `json:"auto_deleted_at"`
	FailedReason  string                       `json:"failed_reason"`
}
type GenerateReportAMRResp struct {
	AMRID uint64 `json:"amr_id"`
}
type ListReportAMRResp struct {
	List       []ListReport `json:"list"`
	TotalItems int32        `json:"total_items"`
	TotalPages int32        `json:"total_pages"`
	Page       int32        `json:"page"`
	PageSize   int32        `json:"page_size"`
}
type ListReport struct {
	AMRDetailID        uint64                       `json:"amr_detail_id"`
	LocationCode       string                       `json:"location_code"`
	LocationType       coreenum.CTXEnumLocationType `json:"location_type"`
	Tariff             string                       `json:"tariff"`
	VDrop              bool                         `json:"v_drop"`
	VLoss              bool                         `json:"v_loss"`
	CosPhiKecil        bool                         `json:"cos_phi_kecil"`
	ILoss              bool                         `json:"i_loss"`
	InGreaterIMax      bool                         `json:"in_greater_i_max"`
	OverI              bool                         `json:"over_i"`
	OverV              bool                         `json:"over_v"`
	ReversePower       bool                         `json:"reverse_power"`
	UnbalanceI         bool                         `json:"unbalance_i"`
	ILowVLow           bool                         `json:"i_low_v_low"`
	CurrentLoop        bool                         `json:"current_loop"`
	ActivePLoss        bool                         `json:"active_p_loss"`
	Freeze             bool                         `json:"freeze"`
	TotalWeightedValue float64                      `json:"total_weighted_value"`
}
type FindAMRResp struct {
	AMRID         uint64                       `json:"amr_id"`
	Filename      string                       `json:"filename"`
	Extension     coreenum.CTXEnumExtension    `json:"extension"`
	RowNumbers    int64                        `json:"row_numbers"`
	ReadDate      string                       `json:"read_date"`
	StageProcess  coreenum.CTXEnumStageProcess `json:"stage_process"`
	CreatedAt     string                       `json:"created_at"`
	AutoDeletedAt string                       `json:"auto_deleted_at"`
}
type FindAMRParamConfigResp struct {
	AMRConfigID             uint64  `json:"amr_config_id"`
	AMRID                   uint64  `json:"amr_id"`
	VDropVTM                float64 `json:"v_drop_v_tm"`
	VDropVTR                float64 `json:"v_drop_v_tr"`
	VDropITM                float64 `json:"v_drop_i_tm"`
	VDropITR                float64 `json:"v_drop_i_tr"`
	VLossVTM                float64 `json:"v_loss_v_tm"`
	VLossVTR                float64 `json:"v_loss_v_tr"`
	VLossITM                float64 `json:"v_loss_i_tm"`
	VLossITR                float64 `json:"v_loss_i_tr"`
	CosPhiKecilITM          float64 `json:"cos_phi_kecil_i_tm"`
	CosPhiKecilITR          float64 `json:"cos_phi_kecil_i_tr"`
	CosPhiKecilUpperLimitTM float64 `json:"cos_phi_kecil_upper_limit_tm"`
	CosPhiKecilUpperLimitTR float64 `son:"cos_phi_kecil_upper_limit_tr"`
	ILossITM                float64 `json:"i_loss_i_tm"`
	ILossITR                float64 `json:"i_loss_i_tr"`
	ILossIMaxTM             float64 `json:"i_loss_i_max_tm"`
	ILossIMaxTR             float64 `json:"i_loss_i_max_tr"`
	InGreaterIMaxInTM       float64 `json:"in_greater_i_max_in_tm"`
	InGreaterIMaxInTR       float64 `json:"in_greater_i_max_in_tr"`
	OverCurrentIMaxTM       float64 `json:"over_current_i_max_tm"`
	OverCurrentIMaxTR       float64 `json:"over_current_i_max_tr"`
	OverVoltageVMaxTM       float64 `json:"over_voltage_v_max_tm"`
	OverVoltageVMaxTR       float64 `json:"over_voltage_v_max_tr"`
	ReversePowerVTM         float64 `json:"reverse_power_v_tm"`
	ReversePowerVTR         float64 `json:"reverse_power_v_tr"`
	ReversePowerITM         float64 `json:"reverse_power_i_tm"`
	ReversePowerITR         float64 `json:"reverse_power_i_tr"`
	IUnbalanceTolTM         float64 `json:"i_unbalance_tol_tm"`
	IUnbalanceTolTR         float64 `json:"i_unbalance_tol_tr"`
	IUnbalanceITM           float64 `json:"i_unbalance_i_tm"`
	IUnbalanceITR           float64 `json:"i_unbalance_i_tr"`
	PLossI                  float64 `json:"p_loss_i"`
	ILowVLowTM              float64 `json:"i_low_v_low_tm"`
	ILowVLowTR              float64 `json:"i_low_v_low_tr"`
	MinIndicatorAmount      float64 `json:"min_indicator_amount"`
	MinWeight               float64 `json:"min_weight"`
	NShowRecommendation     int     `json:"n_show_recommendation"`
	CreatedAt               string  `json:"created_at"`
}

type FindAMRWeightConfigResp struct {
	AMRWeightConfigID uint64  `json:"amr_weight_config_id"`
	AMRID             uint64  `json:"amr_id"`
	VIndirectDrop     float64 `json:"v_indirect_drop"`
	VDirectDrop       float64 `json:"v_direct_drop"`
	VLoss             float64 `json:"v_loss"`
	CosPhiKecil       float64 `json:"cos_phi_kecil"`
	ILoss             float64 `json:"i_loss"`
	InGreatedImax     float64 `json:"in_greated_imax"`
	OverCurrent       float64 `json:"over_current"`
	OverVoltage       float64 `json:"over_voltage"`
	ReversePower      float64 `json:"reverse_power"`
	UnbalanceI        float64 `json:"unbalance_i"`
	ILowVLow          float64 `json:"i_low_v_low"`
	CurrentLoop       float64 `json:"current_loop"`
	ActivePLoss       float64 `json:"active_p_loss"`
	Freeze            float64 `json:"freeze"`
	CreatedAt         string  `json:"created_at"`
}
type DeleteAMRResp struct {
	AMRID uint64 `json:"amr_id"`
}
type SummaryAMRResp struct {
	AMRID              uint64 `json:"amr_id"`
	Filename           string `json:"filename"`
	ReadDate           string `json:"read_date"`
	CreatedAt          string `json:"created_at"`
	AutoDeletedAt      string `json:"auto_deleted_at"`
	RowNumbers         int64  `json:"row_numbers"`
	TotalVDrop         int64  `json:"total_v_drop"`
	TotalVLoss         int64  `json:"total_v_loss"`
	TotalCosPhiKecil   int64  `json:"total_cos_phi_kecil"`
	TotalILoss         int64  `json:"total_i_loss"`
	TotalOverI         int64  `json:"total_over_i"`
	TotalOverV         int64  `json:"total_over_v"`
	TotalUnbalanceI    int64  `json:"total_unbalance_i"`
	TotalILowVLow      int64  `json:"total_i_low_v_low"`
	TotalCurrentLoop   int64  `json:"total_current_loop"`
	TotalActivePLoss   int64  `json:"total_active_p_loss"`
	TotalFreeze        int64  `json:"total_freeze"`
	TotalInGreaterIMax int64  `json:"total_in_greater_i_max"`
	TotalReversePower  int64  `json:"total_reverse_power"`
	AMRParamConfig     FindAMRParamConfigResp
	AMRWeightConfig    FindAMRWeightConfigResp
}
type DownloadAMRTemplateResp struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	XLSXBytes   []byte `json:"-"`
}
type ExportAMRResp struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	XLSXBytes   []byte `json:"-"`
}
type ExportRecommendationAMRResp struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	XLSXBytes   []byte `json:"-"`
}
