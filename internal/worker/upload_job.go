package worker

import (
	"context"
	coreenum "logisfy/core/enum"
)

type UploadJob struct {
	UseCaseName     coreenum.CTXEnumUseCase
	UseCaseDetailID uint64
	UserID          uint64
	FilePath        string
	Ctx             context.Context
}

type RowJob struct {
	Cols      []string
	RowNumber int64
}
