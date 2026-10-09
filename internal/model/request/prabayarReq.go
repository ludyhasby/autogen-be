package request

import (
	"io"
	coreenum "logisfy/core/enum"
)

type UploadPrabayarReq struct {
	File        io.Reader `json:"-"`
	Size        int64     `json:"-"`
	Filename    string    `json:"-"`
	Extension   string    `validate:"required,oneof=.xlsx .xls" json:"-"`
	ContentType string    `json:"-"`
}

type PrabayarFilter struct {
	UserID         *uint64
	PrabayarID     *uint64
	StageProcesses []coreenum.CTXEnumStageProcess
}
