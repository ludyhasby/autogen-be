package response

type RegisterUserResp struct {
	UserID uint64 `json:"id"`
}
type AuthUserResp struct {
	Token string `json:"token"`
	Role  string `json:"role"`
}
type UserActivationResp struct {
	UserID uint64 `json:"user_id"`
}
type ListUserResp struct {
	List       []ListUser `json:"list"`
	TotalItems int32      `json:"total_items"`
	TotalPages int32      `json:"total_pages"`
	Page       int32      `json:"page"`
	PageSize   int32      `json:"page_size"`
}
type ListUser struct {
	UserID    uint64 `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	UP3       string `json:"up3"`
	UnitInduk string `json:"unit_induk"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	LastLogin string `json:"last_login"`
}
type UserDeActivationResp struct {
	UserID uint64 `json:"user_id"`
}
type DeleteUserResp struct {
	UserID uint64 `json:"user_id"`
}
type FindUserResp struct {
	UserID    uint64 `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	UP3       string `json:"up3"`
	UnitInduk string `json:"unit_induk"`
}
type ForgotPasswordResp struct {
	Email string `json:"email"`
}
type ForgotPasswordMail struct {
	Email        string
	Name         string
	ExpiredAtStr string
	URL          string
}
type FindResetPasswordTokenResp struct {
	Token string `json:"token"`
}
type ResetPasswordResp struct {
	Token string `json:"token"`
}
type UserActivationMail struct {
	Email string
	Name  string
	URL   string
}
