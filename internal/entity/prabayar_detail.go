package entity

import (
	"time"
)

type PrabayarDetailEntity struct {
	PrabayarDetailID uint64    `gorm:"primaryKey;autoIncrement" json:"prabayar_detail_id"`
	PrabayarID       uint64    `gorm:"not null;index" json:"prabayar_id"`
	IDPelCrypt       []byte    `gorm:"not null" json:"idpel_crypt"`
	NoMeterCrypt     []byte    `gorm:"not null" json:"no_meter_crypt"`
	Tariff           string    `gorm:"not null" json:"tariff"`
	Power            float64   `gorm:"not null" json:"power"`
	ReadDate         time.Time `gorm:"not null" json:"read_date"`
	Voltage          float64   `gorm:"not null" json:"voltage"`
	Current          float64   `gorm:"not null" json:"current"`
	Cosphi           float64   `gorm:"not null" json:"cosphi"`
	SealCondition    string    `gorm:"not null" json:"seal_condition"`
	LCDCondition     string    `gorm:"not null" json:"lcd_condition"`
	KeypadCondition  string    `gorm:"not null" json:"keypad_condition"`
	TerminalAmount   int       `gorm:"not null" json:"terminal_amount"`
	TemperIndicator  string    `gorm:"not null" json:"temper_indicator"`
	RelayIndicator   string    `gorm:"not null" json:"relay_indicator"`
	CreatedAt        time.Time `gorm:"autoCreateTime;not null" json:"created_at"`

	PrabayarEntity PrabayarEntity `gorm:"-"`
}

func (t PrabayarDetailEntity) TableName() string {
	return "prabayar_detail"
}

func (t PrabayarDetailEntity) CheckFound() bool {
	return t.PrabayarDetailID > 0
}
