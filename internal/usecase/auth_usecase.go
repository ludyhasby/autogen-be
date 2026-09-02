package usecase

import (
	"context"
	"log/slog"
	coreenum "logisfy/core/enum"
	helperexception "logisfy/helper/exception"
	helperhash "logisfy/helper/hash"
	"logisfy/internal/delivery/http/middleware"
	"logisfy/internal/entity"
	modelrequest "logisfy/internal/model/request"
	modelresponse "logisfy/internal/model/response"
	"logisfy/internal/repository"
	"logisfy/internal/worker"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type AuthUseCase struct {
	DB                           *gorm.DB
	Log                          *slog.Logger
	SecretKeyStr                 string
	Validate                     *validator.Validate
	Location                     *time.Location
	FrontEndURL                  string
	MailWorker                   *worker.MailWorker
	UserRepository               *repository.UserRepository
	PasswordResetTokenRepository *repository.PasswordResetTokenRepository
}

func NewAuthUseCase(
	db *gorm.DB,
	log *slog.Logger,
	validate *validator.Validate,
	secretKeyStr string,
	location *time.Location,
	frontEndURL string,
	mailWorker *worker.MailWorker,
	userRepository *repository.UserRepository,
	passwordResetTokenRepository *repository.PasswordResetTokenRepository,
) *AuthUseCase {
	return &AuthUseCase{
		DB:                           db,
		Log:                          log,
		SecretKeyStr:                 secretKeyStr,
		Location:                     location,
		FrontEndURL:                  frontEndURL,
		Validate:                     validate,
		MailWorker:                   mailWorker,
		UserRepository:               userRepository,
		PasswordResetTokenRepository: passwordResetTokenRepository,
	}
}

func (u *AuthUseCase) Register(ctx context.Context, req *modelrequest.RegisterUserReq) (resp modelresponse.RegisterUserResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AuthUseCase")
	ctx, span := tr.Start(ctx, "Register()")
	defer span.End()

	var err error
	// Validator
	if err = u.Validate.Struct(req); err != nil {
		u.Log.Info("AuthUseCase.Register()", "Validate.Struct()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	// Init DB
	tx := u.DB.WithContext(ctx)
	defer func() {
		if exc != nil || err != nil {
			tx.Rollback()
		}
	}()

	// Check existing email
	existingUser, err := u.UserRepository.FindByEmail(tx, req.Email)
	if err != nil {
		u.Log.Error("AuthUseCase.Register()", "UserRepository.FindByEmail()", "error", err.Error())
		exc = helperexception.Internal("error once find user by email", err)
		return
	}
	if existingUser.CheckFound() {
		exc = helperexception.Conflict("email has been used")
		return
	}

	tx = tx.Begin()

	// create user
	entityUser := (&entity.UserEntity{}).Create(req, coreenum.CTXEnumRoleUser)
	// exec repo
	err = u.UserRepository.Create(tx, entityUser)
	if err != nil {
		u.Log.Error("AuthUseCase.Register()", "UserRepository.Create()", "error", err.Error())
		exc = helperexception.Internal("failed to create entity user", err)
		return
	}
	// response
	resp.UserID = entityUser.UserID

	// commit
	if err = tx.Commit().Error; err != nil {
		u.Log.Error("AuthUseCase.Register()", "tx.Commit()", "error", err.Error())
		exc = helperexception.Internal("failed to create entity user", err)
		return
	}
	return
}

func (u *AuthUseCase) Login(ctx context.Context, req *modelrequest.AuthUserReq) (resp modelresponse.AuthUserResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AuthUseCase")
	ctx, span := tr.Start(ctx, "Login()")
	defer span.End()

	var err error
	// Validator
	if err = u.Validate.Struct(req); err != nil {
		u.Log.Info("AuthUseCase.Login()", "Validate.Struct()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	// Init Transaction
	tx := u.DB.WithContext(ctx)

	// FindByAMRID User By Email
	entityUser, err := u.UserRepository.FindByEmail(tx, req.Email)
	if err != nil {
		u.Log.Info("AuthUseCase.Login()", "UserRepository.FindByEmail()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses login", err)
		return
	}
	// Check Email Found
	if !entityUser.CheckFound() {
		exc = helperexception.NotFound("email/password salah")
		return
	}

	// Check Password
	isTrue := helperhash.ComparePass(entityUser.Password, req.Password)
	if !isTrue {
		u.Log.Info("AuthUseCase.Login()", "CheckPassword()", "Err", err)
		exc = helperexception.NotFound("email/password salah")
		return
	}

	// validate status
	if !entityUser.IsActive {
		exc = helperexception.PermissionDenied("akun anda telah ditangguhkan, silakan hubungi admin untuk mengaktifkannya kembali.")
		return
	}

	// Generate JWT Naked
	jwtNaked := middleware.AuthJWT{
		Email:  entityUser.Email,
		Name:   entityUser.Name,
		UserID: strconv.Itoa(int(entityUser.UserID)),
		Role:   entityUser.Role.String(),
	}

	// Generate JWT
	tokenString, err := middleware.GenerateJWT(jwtNaked, u.SecretKeyStr, req.RememberMe)
	if err != nil {
		u.Log.Info("AuthUseCase.LoginUser()", "middleware.GenerateJWT()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses login", err)
		return
	}

	// Update Last Login
	timeNow := time.Now()
	entityUser.LastLoginAt = &timeNow
	err = u.UserRepository.Update(tx, &entityUser)
	if err != nil {
		u.Log.Info("AuthUseCase.LoginUser()", "UserRepository.Update()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses login", err)
		return
	}

	// response
	resp.Token = tokenString
	resp.Role = entityUser.Role.String()
	return
}

func (u *AuthUseCase) Activation(ctx context.Context, req *modelrequest.UserActivationReq) (resp modelresponse.UserActivationResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AuthUseCase")
	ctx, span := tr.Start(ctx, "Activation()")
	defer span.End()

	var err error
	userID, err := strconv.Atoi(req.UserID)
	if err != nil {
		u.Log.Info("AuthUseCase.Activation()", "strconv.Atoi()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	// Init Transaction
	tx := u.DB.WithContext(ctx)

	// FindByAMRID User By User ID
	entityUser, err := u.UserRepository.FindByUserID(tx, uint64(userID))
	if err != nil {
		u.Log.Info("AuthUseCase.Activation()", "UserRepository.FindByUserID()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses pencarian user", err)
		return
	}
	if !entityUser.CheckFound() {
		exc = helperexception.NotFound("user tidak ditemukan")
		return
	}
	// validate status
	if entityUser.IsActive {
		exc = helperexception.PermissionDenied("akun telah aktif.")
		return
	}

	entityUser.IsActive = true
	err = u.UserRepository.Update(tx, entityUser)
	if err != nil {
		u.Log.Info("AuthUseCase.Activation()", "UserRepository.Update()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses aktivasi", err)
		return
	}

	resp.UserID = entityUser.UserID

	userActivationMail := modelresponse.UserActivationMail{
		Email: entityUser.Email,
		Name:  entityUser.Name,
		URL:   u.FrontEndURL,
	}
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				u.Log.Error("AuthUseCase.Activation(): panic recovered in mail worker goroutine", "panic", r)
			}
		}()
		for i := 0; i < 2; i++ {
			if mailErr := u.MailWorker.Activation(bgCtx, "file/template/user-activation.html", userActivationMail); mailErr == nil {
				break
			}
			time.Sleep(2 * time.Second)
		}
	}()
	return
}

func (u *AuthUseCase) List(ctx context.Context, req *modelrequest.ListUserReq) (resp modelresponse.ListUserResp, exc *helperexception.Exception) {
	// init tracer
	tr := otel.Tracer("useCase.AccountUseCase")
	ctx, span := tr.Start(ctx, "List()")
	defer span.End()

	// Init DB
	tx := u.DB.WithContext(ctx)

	// exec repo
	entityUserList, items, totalPages, size, err := u.UserRepository.List(tx, req.QueryInfo)
	if err != nil {
		u.Log.Info("AuthUseCase.List()", "UserRepository.List()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses list", err)
		return
	}

	// resp
	resp.List = (&entity.UserEntity{}).ConvertToList(entityUserList)
	resp.TotalItems = items
	resp.Page = req.QueryInfo.SelectParameter.PageDescriptor.PageIndex
	resp.PageSize = size
	resp.TotalPages = totalPages
	return
}

func (u *AuthUseCase) DeActivation(ctx context.Context, req *modelrequest.UserDeActivationReq) (resp modelresponse.UserDeActivationResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AuthUseCase")
	ctx, span := tr.Start(ctx, "DeActivation()")
	defer span.End()

	var err error
	userID, err := strconv.Atoi(req.UserID)
	if err != nil {
		u.Log.Info("AuthUseCase.DeActivation()", "strconv.Atoi()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	// Init Transaction
	tx := u.DB.WithContext(ctx)

	// FindByAMRID User By User ID
	entityUser, err := u.UserRepository.FindByUserID(tx, uint64(userID))
	if err != nil {
		u.Log.Info("AuthUseCase.DeActivation()", "UserRepository.FindByUserID()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses pencarian user", err)
		return
	}
	if !entityUser.CheckFound() {
		exc = helperexception.NotFound("user tidak ditemukan")
		return
	}
	// validate status
	if !entityUser.IsActive {
		exc = helperexception.PermissionDenied("akun telah non aktif.")
		return
	}

	entityUser.IsActive = false
	err = u.UserRepository.Update(tx, entityUser)
	if err != nil {
		u.Log.Info("AuthUseCase.DeActivation()", "UserRepository.Update()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses de aktivasi", err)
		return
	}

	// response
	resp.UserID = entityUser.UserID
	return
}

func (u *AuthUseCase) Delete(ctx context.Context, req *modelrequest.DeleteUserReq) (resp modelresponse.DeleteUserResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AuthUseCase")
	ctx, span := tr.Start(ctx, "Delete()")
	defer span.End()

	var err error
	userID, err := strconv.Atoi(req.UserID)
	if err != nil {
		u.Log.Info("AuthUseCase.Delete()", "strconv.Atoi()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	// Init Transaction
	tx := u.DB.WithContext(ctx)

	// FindByAMRID User By User ID
	entityUser, err := u.UserRepository.FindByUserID(tx, uint64(userID))
	if err != nil {
		u.Log.Info("AuthUseCase.Delete()", "UserRepository.FindByUserID()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses pencarian user", err)
		return
	}
	if !entityUser.CheckFound() {
		exc = helperexception.NotFound("user tidak ditemukan")
		return
	}
	// validate status
	if entityUser.IsActive {
		exc = helperexception.PermissionDenied("akun sedang aktif, lakukan deaktivasi terlebih dahulu.")
		return
	}

	err = u.UserRepository.Delete(tx, entityUser)
	if err != nil {
		u.Log.Info("AuthUseCase.Delete()", "UserRepository.Delete()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses hapus user", err)
		return
	}

	// response
	resp.UserID = entityUser.UserID
	return
}

func (u *AuthUseCase) Find(ctx context.Context) (resp modelresponse.FindUserResp, exc *helperexception.Exception) {
	// Init Tracer
	tr := otel.Tracer("useCase.AuthUseCase")
	ctx, span := tr.Start(ctx, "Find()")
	defer span.End()

	var err error
	userID, err := strconv.Atoi(ctx.Value(string(coreenum.CTXEnumIDUserID)).(string))
	if err != nil {
		u.Log.Warn("AuthUseCase.Find()", "strconv.Atoi()", "warn", err.Error())
		exc = helperexception.InvalidArgument(err, ctx.Value(string(coreenum.CTXEnumIDUserID)))
		return
	}

	// Init Transaction
	tx := u.DB.WithContext(ctx)

	// FindByAMRID User By User ID
	entityUser, err := u.UserRepository.FindByUserID(tx, uint64(userID))
	if err != nil {
		u.Log.Info("AuthUseCase.Find()", "UserRepository.FindByUserID()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses pencarian user", err)
		return
	}
	if !entityUser.CheckFound() {
		exc = helperexception.NotFound("user tidak ditemukan")
		return
	}
	// response
	resp.UserID = entityUser.UserID
	resp.Name = entityUser.Name
	resp.Email = entityUser.Email
	resp.UP3 = entityUser.UP3
	resp.UnitInduk = entityUser.UnitInduk
	return
}

func (u *AuthUseCase) ForgotPassword(ctx context.Context, req *modelrequest.ForgotPasswordReq) (resp modelresponse.ForgotPasswordResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AuthUseCase")
	ctx, span := tr.Start(ctx, "ForgotPassword()")
	defer span.End()

	var err error
	if err = u.Validate.Struct(req); err != nil {
		u.Log.Info("AuthUseCase.ForgotPassword()", "Validate.Struct()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}
	resp.Email = req.Email

	tx := u.DB.WithContext(ctx)
	entityUser, err := u.UserRepository.FindByEmail(tx, req.Email)
	if err != nil {
		u.Log.Info("AuthUseCase.ForgotPassword()", "UserRepository.FindByEmail()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses forgot password", err)
		return
	}
	if !entityUser.CheckFound() {
		return
	}

	tx = tx.Begin()
	defer func() {
		if exc != nil || err != nil {
			tx.Rollback()
		}
	}()
	if err = u.PasswordResetTokenRepository.UpdatePasswordResetTokenByUserID(tx, entityUser.UserID, entity.PasswordResetTokenDeActive); err != nil {
		u.Log.Info("AuthUseCase.ForgotPassword()", "PasswordResetTokenRepository.UpdatePasswordResetTokenByUserID()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat menonaktifkan token reset password sebelumnya", err)
		return
	}

	passwordResetTokenEntity := (&entity.PasswordResetTokenEntity{}).Create(req, entityUser.UserID, entity.PasswordResetTokenActive)
	if err = u.PasswordResetTokenRepository.Create(tx, passwordResetTokenEntity); err != nil {
		u.Log.Info("AuthUseCase.ForgotPassword()", "PasswordResetTokenRepository.Create()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat membuat token reset password", err)
		return
	}

	expiredStr := passwordResetTokenEntity.ExpiresAt.In(u.Location).Format("02 Jan 2006, 15:04 WIB")
	forgotPassword := modelresponse.ForgotPasswordMail{
		Email:        entityUser.Email,
		Name:         entityUser.Name,
		ExpiredAtStr: expiredStr,
		URL:          u.FrontEndURL + "/reset-password?session-key=" + passwordResetTokenEntity.SessionKey,
	}

	if err = tx.Commit().Error; err != nil {
		u.Log.Error("AuthUseCase.ForgotPassword()", "tx.Commit()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk membuat token reset password", err)
		return
	}

	bgCtx := context.WithoutCancel(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				u.Log.Error("AuthUseCase.ForgotPassword(): panic recovered in mail worker goroutine", "panic", r)
			}
		}()
		for i := 0; i < 2; i++ {
			if mailErr := u.MailWorker.ForgotPassword(bgCtx, "file/template/forgot-password.html", forgotPassword); mailErr == nil {
				break
			}
			time.Sleep(2 * time.Second)
		}
	}()
	return
}

func (u *AuthUseCase) FindResetPasswordToken(ctx context.Context, req *modelrequest.FindResetPasswordTokenReq) (resp modelresponse.FindResetPasswordTokenResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AuthUseCase")
	ctx, span := tr.Start(ctx, "FindResetPasswordToken()")
	defer span.End()

	var err error
	if err = u.Validate.Struct(req); err != nil {
		u.Log.Info("AuthUseCase.FindResetPasswordToken()", "Validate.Struct()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	resetPasswordEntity, err := u.PasswordResetTokenRepository.FindBySessionKey(tx, req.SessionKey)
	if err != nil {
		u.Log.Error("AuthUseCase.FindResetPasswordToken()", "PasswordResetTokenRepository.FindBySessionKey()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari token reset password", err)
		return
	}
	if !resetPasswordEntity.CheckFound() {
		exc = helperexception.NotFound("token tidak valid")
		return
	}
	now := time.Now()
	if resetPasswordEntity.ExpiresAt.In(u.Location).Before(now) {
		exc = helperexception.InvalidArgument("token expired", nil)
		return
	}
	if !*resetPasswordEntity.IsActive {
		exc = helperexception.InvalidArgument("token sudah tidak aktif", nil)
		return
	}

	resp.Token = req.SessionKey
	return
}

func (u *AuthUseCase) ResetPassword(ctx context.Context, req *modelrequest.ResetPasswordReq) (resp modelresponse.ResetPasswordResp, exc *helperexception.Exception) {
	tr := otel.Tracer("useCase.AuthUseCase")
	ctx, span := tr.Start(ctx, "ResetPassword()")
	defer span.End()

	var err error
	if err = u.Validate.Struct(req); err != nil {
		u.Log.Info("AuthUseCase.ResetPassword()", "Validate.Struct()", "error", err.Error())
		exc = helperexception.InvalidArgument(err, req)
		return
	}

	tx := u.DB.WithContext(ctx)
	resetPasswordEntity, err := u.PasswordResetTokenRepository.FindBySessionKey(tx, req.SessionKey)
	if err != nil {
		u.Log.Error("AuthUseCase.ResetPassword()", "PasswordResetTokenRepository.FindBySessionKey()", "error", err.Error())
		exc = helperexception.Internal("gagal untuk mencari token reset password", err)
		return
	}
	if !resetPasswordEntity.CheckFound() {
		exc = helperexception.NotFound("token tidak valid")
		return
	}
	now := time.Now()
	if resetPasswordEntity.ExpiresAt.In(u.Location).Before(now) {
		exc = helperexception.InvalidArgument("token expired", nil)
		return
	}
	if resetPasswordEntity.IsActive == nil || !*resetPasswordEntity.IsActive {
		exc = helperexception.InvalidArgument("token sudah tidak aktif", nil)
		return
	}

	entityUser, err := u.UserRepository.FindByUserID(tx, resetPasswordEntity.UserID)
	if err != nil {
		u.Log.Info("AuthUseCase.ResetPassword()", "UserRepository.FindByUserID()", "Err", err.Error())
		exc = helperexception.Internal("gagal saat proses reset password", err)
		return
	}
	if !entityUser.CheckFound() {
		exc = helperexception.NotFound("user tidak ditemukan")
		return
	}
	if helperhash.ComparePass(entityUser.Password, req.NewPassword) {
		exc = helperexception.InvalidArgument("password baru tidak boleh sama dengan password lama", nil)
		return
	}
	newPasswordHashed := helperhash.HashPassword(req.NewPassword)
	entityUser.Password = newPasswordHashed

	tx = tx.Begin()
	defer func() {
		if exc != nil || err != nil {
			tx.Rollback()
		}
	}()
	if err = u.PasswordResetTokenRepository.UpdatePasswordResetTokenByUserID(tx, entityUser.UserID, entity.PasswordResetTokenDeActive); err != nil {
		u.Log.Error("AuthUseCase.ResetPassword()", "PasswordResetTokenRepository.UpdatePasswordResetTokenByUserID()", "error", err.Error())
		exc = helperexception.Internal("gagal saat proses reset password", err)
		return
	}
	if err = u.UserRepository.Update(tx, entityUser); err != nil {
		u.Log.Error("AuthUseCase.ResetPassword()", "UserRepository.Update()", "error", err.Error())
		exc = helperexception.Internal("gagal saat proses reset password", err)
		return
	}
	if err = tx.Commit().Error; err != nil {
		u.Log.Error("AuthUseCase.ResetPassword()", "tx.Commit()", "error", err.Error())
		exc = helperexception.Internal("gagal saat proses reset password", err)
		return
	}

	resp.Token = req.SessionKey
	return
}
