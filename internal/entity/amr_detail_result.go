package entity

import (
	coreenum "logisfy/core/enum"
	"logisfy/helper/crypto"
	helperprocess "logisfy/helper/process"
	modelresponse "logisfy/internal/model/response"
	"time"
)

type AMRDetailResultEntity struct {
	AMRDetailResultID  uint64    `gorm:"primaryKey;autoIncrement" json:"amr_detail_result_id"`
	AMRDetailID        uint64    `gorm:"not null;index" json:"amr_detail_id"`
	VDrop              bool      `json:"v_drop"`
	VLoss              bool      `json:"v_loss"`
	CosPhiKecil        bool      `json:"cos_phi_kecil"`
	ILoss              bool      `json:"i_loss"`
	InGreaterIMax      bool      `json:"in_greater_i_max"`
	OverI              bool      `json:"over_i"`
	OverV              bool      `json:"over_v"`
	ReversePower       bool      `json:"reverse_power"`
	UnbalanceI         bool      `json:"unbalance_i"`
	ILowVLow           bool      `json:"i_low_v_low"`
	CurrentLoop        bool      `json:"current_loop"`
	ActivePLoss        bool      `json:"active_p_loss"`
	Freeze             bool      `json:"freeze"`
	TotalWeightedValue float64   `json:"total_weighted_value"`
	CreatedAt          time.Time `gorm:"autoCreateTime;not null"`

	AMRDetailEntity AMRDetailEntity `gorm:"-"`
}

type Report struct {
	AMRDetailID         uint64
	LocationCodeEncrypt []byte
	LocationType        coreenum.CTXEnumLocationType
	Tariff              string
	AMRDetailResultEntity
}

type SummaryAMRDetailResult struct {
	TotalVDrop         int64
	TotalVLoss         int64
	TotalCosPhiKecil   int64
	TotalILoss         int64
	TotalOverI         int64
	TotalOverV         int64
	TotalUnbalanceI    int64
	TotalILowVLow      int64
	TotalCurrentLoop   int64
	TotalActivePLoss   int64
	TotalFreeze        int64
	TotalInGreaterIMax int64
	TotalReversePower  int64
}

func (t AMRDetailResultEntity) TableName() string {
	return "amr_detail_result"
}

func (t AMRDetailResultEntity) CheckFound() bool {
	return t.AMRDetailResultID > 0
}

func (t AMRDetailResultEntity) Create(entityAMRDetails []*AMRDetailEntity, entityParamConfig *AMRConfigEntity, entityWeightConfig *AMRWeightConfigEntity) (amrDetailResults []*AMRDetailResultEntity) {
	var (
		vDrop         bool
		vLoss         bool
		cosPhiKecil   bool
		iLoss         bool
		inGreaterIMax bool
		overI         bool
		overV         bool
		reversePower  bool
		unbalanceI    bool
		iLowVLow      bool
		currentLoop   bool
		activePLoss   bool
		freeze        bool
	)

	for _, amrDetail := range entityAMRDetails {
		vDrop = helperprocess.HasVDrop(amrDetail.VoltageType,
			entityParamConfig.VDropVTM, entityParamConfig.VDropVTR, entityParamConfig.VDropITM, entityParamConfig.VDropITR,
			amrDetail.VoltageL1, amrDetail.VoltageL2, amrDetail.VoltageL3,
			amrDetail.CurrentL1, amrDetail.CurrentL2, amrDetail.CurrentL3)
		vLoss = helperprocess.HasVLoss(amrDetail.VoltageType,
			entityParamConfig.VLossVTM, entityParamConfig.VLossVTR, entityParamConfig.VLossITM, entityParamConfig.VLossITR,
			amrDetail.Phase,
			amrDetail.VoltageL1, amrDetail.VoltageL2, amrDetail.VoltageL3,
			amrDetail.CurrentL1, amrDetail.CurrentL2, amrDetail.CurrentL3)
		cosPhiKecil = helperprocess.HasCosPhiKecil(amrDetail.VoltageType,
			amrDetail.MeasurementType,
			entityParamConfig.CosPhiKecilUpperLimitTM, entityParamConfig.CosPhiKecilUpperLimitTR, entityParamConfig.CosPhiKecilITM, entityParamConfig.CosPhiKecilITR,
			amrDetail.PowerFactorL1, amrDetail.PowerFactorL2, amrDetail.PowerFactorL3,
			amrDetail.CurrentL1, amrDetail.CurrentL2, amrDetail.CurrentL3)
		iLoss = helperprocess.HasILoss(amrDetail.VoltageType,
			entityParamConfig.ILossITM, entityParamConfig.ILossITR, entityParamConfig.ILossIMaxTM, entityParamConfig.ILossIMaxTR,
			amrDetail.CurrentL1, amrDetail.CurrentL2, amrDetail.CurrentL3, amrDetail.CurrentMax)
		inGreaterIMax = helperprocess.HasInGreaterIMax(amrDetail.VoltageType,
			entityParamConfig.InGreaterIMaxInTM, entityParamConfig.InGreaterIMaxInTR,
			amrDetail.CurrentN, amrDetail.CurrentMax, amrDetail.CurrentMin)
		overI = helperprocess.HasOverCurrent(amrDetail.VoltageType, entityParamConfig.OverCurrentIMaxTM,
			entityParamConfig.OverCurrentIMaxTR, amrDetail.Power, amrDetail.CurrentMax)
		overV = helperprocess.HasOverVoltage(amrDetail.VoltageType, entityParamConfig.OverVoltageVMaxTM, entityParamConfig.OverVoltageVMaxTR,
			amrDetail.VoltageMax)
		reversePower = helperprocess.HasReversePower(amrDetail.VoltageType,
			entityParamConfig.ReversePowerVTM, entityParamConfig.ReversePowerVTR, entityParamConfig.ReversePowerITM, entityParamConfig.ReversePowerITR,
			amrDetail.ActivePowerL1, amrDetail.ActivePowerL2, amrDetail.ActivePowerL3, amrDetail.CurrentL1, amrDetail.CurrentL2, amrDetail.CurrentL3)
		unbalanceI = helperprocess.HasUnbalanceCurrent(amrDetail.MeasurementType, amrDetail.VoltageType,
			entityParamConfig.IUnbalanceTolTM, entityParamConfig.IUnbalanceTolTR, entityParamConfig.IUnbalanceITM, entityParamConfig.IUnbalanceITR,
			amrDetail.CurrentL1, amrDetail.CurrentL2, amrDetail.CurrentL3)
		iLowVLow = helperprocess.HasILowVLow(amrDetail.VoltageType,
			entityParamConfig.ILowVLowTM, entityParamConfig.ILowVLowTR,
			amrDetail.CurrentL1, amrDetail.CurrentL2, amrDetail.CurrentL3,
			amrDetail.VoltageL1, amrDetail.VoltageL2, amrDetail.VoltageL3)
		currentLoop = helperprocess.HasCurrentLoop(amrDetail.CurrentL1, amrDetail.CurrentL2, amrDetail.CurrentL3,
			amrDetail.CurrentAngleL1, amrDetail.CurrentAngleL2, amrDetail.CurrentAngleL3, amrDetail.Power)
		activePLoss = helperprocess.HasActivePowerLost(amrDetail.BillReffKwh,
			entityParamConfig.PLossI,
			amrDetail.ActivePowerL1, amrDetail.ActivePowerL2, amrDetail.ActivePowerL3,
			amrDetail.CurrentL1, amrDetail.CurrentL2, amrDetail.CurrentL3)
		freeze = helperprocess.HasFreeze(amrDetail.VoltageMax)

		amrDetailResult := &AMRDetailResultEntity{
			AMRDetailID:   amrDetail.AMRDetailID,
			VDrop:         vDrop,
			VLoss:         vLoss,
			CosPhiKecil:   cosPhiKecil,
			ILoss:         iLoss,
			InGreaterIMax: inGreaterIMax,
			OverI:         overI,
			OverV:         overV,
			ReversePower:  reversePower,
			UnbalanceI:    unbalanceI,
			ILowVLow:      iLowVLow,
			CurrentLoop:   currentLoop,
			ActivePLoss:   activePLoss,
			Freeze:        freeze,
			TotalWeightedValue: totalWeightedAMR(vDrop, vLoss, cosPhiKecil, iLoss, inGreaterIMax, overI, overV, reversePower,
				unbalanceI, iLowVLow, currentLoop, activePLoss, freeze, amrDetail.Power, entityWeightConfig),
		}
		amrDetailResults = append(amrDetailResults, amrDetailResult)
	}
	return
}

func totalWeightedAMR(vDrop, vLoss, cosPhiKecil, iLoss, inGreaterIMax, overI, overV, reversePower,
	unbalanceI, iLowVLow, currentLoop, activePLoss, freeze bool,
	power int64, entityWeight *AMRWeightConfigEntity) float64 {
	return helperprocess.VDropWeighted(vDrop, power, entityWeight.VIndirectDrop, entityWeight.VDirectDrop) +
		helperprocess.CommonWeighted(vLoss, entityWeight.VLoss) +
		helperprocess.CommonWeighted(cosPhiKecil, entityWeight.CosPhiKecil) +
		helperprocess.CommonWeighted(iLoss, entityWeight.ILoss) +
		helperprocess.CommonWeighted(inGreaterIMax, entityWeight.InGreatedImax) +
		helperprocess.CommonWeighted(overI, entityWeight.OverCurrent) +
		helperprocess.CommonWeighted(overV, entityWeight.OverVoltage) +
		helperprocess.CommonWeighted(reversePower, entityWeight.ReversePower) +
		helperprocess.CommonWeighted(unbalanceI, entityWeight.UnbalanceI) +
		helperprocess.CommonWeighted(iLowVLow, entityWeight.ILowVLow) +
		helperprocess.CommonWeighted(currentLoop, entityWeight.CurrentLoop) +
		helperprocess.CommonWeighted(activePLoss, entityWeight.ActivePLoss) +
		helperprocess.CommonWeighted(freeze, entityWeight.Freeze)
}

func (t AMRDetailResultEntity) decrypt(data []byte, crypto crypto.Crypto) string {
	if len(data) > 0 {
		ec, _ := crypto.Decrypt(data)
		return string(ec)
	}

	return ""
}

func (t AMRDetailResultEntity) ConvertToReport(reports []Report, crypto crypto.Crypto) (resp []modelresponse.ListReport) {
	for _, report := range reports {
		locationCode := t.decrypt(report.LocationCodeEncrypt, crypto)
		resp = append(resp, modelresponse.ListReport{
			AMRDetailID:        report.AMRDetailID,
			LocationCode:       locationCode,
			LocationType:       report.LocationType,
			Tariff:             report.Tariff,
			VDrop:              report.VDrop,
			VLoss:              report.VLoss,
			CosPhiKecil:        report.CosPhiKecil,
			ILoss:              report.ILoss,
			InGreaterIMax:      report.InGreaterIMax,
			OverI:              report.OverI,
			OverV:              report.OverV,
			ReversePower:       report.ReversePower,
			UnbalanceI:         report.UnbalanceI,
			ILowVLow:           report.ILowVLow,
			CurrentLoop:        report.CurrentLoop,
			ActivePLoss:        report.ActivePLoss,
			Freeze:             report.Freeze,
			TotalWeightedValue: report.TotalWeightedValue,
		})
	}
	return
}
