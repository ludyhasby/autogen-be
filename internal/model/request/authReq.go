package request

import "logisfy/core"

type RegisterUserReq struct {
	Name      string `json:"name" validate:"required" msg:"nama tidak boleh kosong"`
	Email     string `json:"email" validate:"required,email" msg:"format email salah, pastikan email belum pernah di daftarkan"`
	Password  string `json:"password" validate:"required,min=8" msg:"password minimal 8 karakter, berisi 1 huruf kapital, dan 1 angka"`
	UP3       string `json:"UP3"`
	UnitInduk string `json:"unit_induk"`
}
type AuthUserReq struct {
	Email      string `json:"email" validate:"required,email" msg:"format email salah"`
	Password   string `json:"password" validate:"required" msg:"password harus diisi"`
	RememberMe *bool  `json:"remember_me"`
}
type UserActivationReq struct {
	UserID string `validate:"required" json:"-" msg:"user id tidak valid"`
}
type ListUserReq struct {
	QueryInfo core.QueryInfo `json:"-"`
}
type UserDeActivationReq struct {
	UserID string `validate:"required" json:"-" msg:"user id tidak valid"`
}
type DeleteUserReq struct {
	UserID string `validate:"required" json:"-" msg:"user id tidak valid"`
}
