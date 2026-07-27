package entity

import (
	coreenum "logisfy/core/enum"
	"time"
)

type AMRDetailEntity struct {
	AMRDetailID         uint64                          `gorm:"primaryKey;autoIncrement" json:"amr_detail_id"`
	AMRID               uint64                          `json:"amr_id"`
	LocationCodeEncrypt []byte                          `json:"location_code_encrypt"`
	LocationType        coreenum.CTXEnumLocationType    `json:"location_type"`
	TypeMeter           string                          `json:"type_meter"`
	Tariff              string                          `json:"tariff"`
	Power               int64                           `json:"power"`
	Phase               int                             `json:"phase"`
	MeasurementType     coreenum.CTXEnumMeasurementType `json:"measurement_type"`
	CurrentL1           float64                         `json:"current_l1"`
	CurrentL2           float64                         `json:"current_l2"`
	CurrentL3           float64                         `json:"current_l3"`
	CurrentMax          float64                         `json:"current_max"`
	CurrentMin          float64                         `json:"current_min"`
	CurrentN            float64                         `json:"current_n"`
	VoltageL1           float64                         `json:"voltage_l1"`
	VoltageL2           float64                         `json:"voltage_l2"`
	VoltageL3           float64                         `json:"voltage_l3"`
	VoltageMax          float64                         `json:"voltage_max"`
	VoltageType         coreenum.CTXEnumVoltageType     `json:"voltage_type"`
	ActivePowerL1       float64                         `json:"active_power_l1"`
	ActivePowerL2       float64                         `json:"active_power_l2"`
	ActivePowerL3       float64                         `json:"active_power_l3"`
	ActivePowerTotal    float64                         `json:"active_power_total"`
	KWHAbsTotal         float64                         `json:"kwh_abs_total"`
	CurrentAngleL1      float64                         `json:"current_angle_l1"`
	CurrentAngleL2      float64                         `json:"current_angle_l2"`
	CurrentAngleL3      float64                         `json:"current_angle_l3"`
	VoltageAngleL1      float64                         `json:"voltage_angle_l1"`
	VoltageAngleL2      float64                         `json:"voltage_angle_l2"`
	VoltageAngleL3      float64                         `json:"voltage_angle_l3"`
	ApparentPowerL1     float64                         `json:"apparent_power_l1"`
	ApparentPowerL2     float64                         `json:"apparent_power_l2"`
	ApparentPowerL3     float64                         `json:"apparent_power_l3"`
	BillReffKwh         int                             `json:"bill_reff_kwh"`
	ReadDate            time.Time                       `gorm:"not null" json:"read_date"`
	CreatedAt           time.Time                       `gorm:"autoCreateTime" json:"created_at"`
	PowerFactorL1       float64                         `json:"power_factor_l1"`
	PowerFactorL2       float64                         `json:"power_factor_l2"`
	PowerFactorL3       float64                         `json:"power_factor_l3"`

	AMREntity AMREntity `gorm:"-"`
}

func (t AMRDetailEntity) TableName() string {
	return "amr_detail"
}

func (t AMRDetailEntity) CheckFound() bool {
	return t.AMRDetailID > 0
}
