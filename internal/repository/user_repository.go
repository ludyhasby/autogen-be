package repository

import (
	"errors"
	"log/slog"
	"logisfy/core"
	coreenum "logisfy/core/enum"
	"logisfy/internal/entity"
	"math"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type UserRepository struct {
	*Repository[entity.UserEntity]
	Log *slog.Logger
}

func NewUserRepository(
	log *slog.Logger,
) *UserRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.UserEntity{}).TableName(), "created_at")
	repo := NewRepositoryImpl[entity.UserEntity](sqlGenerator)
	return &UserRepository{
		Repository: repo,
		Log:        log,
	}
}

func (r *UserRepository) FindByEmail(tx *gorm.DB, email string) (result *entity.UserEntity, errRes error) {

	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.UserRepository")
	ctx, span := tr.Start(ctx, "FindByEmail()")
	defer span.End()

	if err := tx.Where("email=? AND deleted_at IS NULL", email).Take(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		r.Log.Error("UserRepository.FindByEmail()", "Exec()", "error", err.Error())
		errRes = err
		return
	}
	return
}

func (r *UserRepository) FindByUserID(tx *gorm.DB, userID uint64) (result *entity.UserEntity, errRes error) {

	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.UserRepository")
	ctx, span := tr.Start(ctx, "FindByUserID()")
	defer span.End()

	if err := tx.Where("user_id=? AND deleted_at IS NULL", userID).Take(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		r.Log.Error("UserRepository.FindByUserID()", "Exec()", "error", err.Error())
		errRes = err
		return
	}
	return
}

func (r *UserRepository) List(tx *gorm.DB, param core.QueryInfo) (results []entity.UserEntity, totalItems int32, totalPages int32, pageSize int32, err error) {
	ctx := tx.Statement.Context

	// init tracer
	tr := otel.Tracer("repository.UserRepository")
	ctx, span := tr.Start(ctx, "List()")
	defer span.End()

	//default deleted_at
	filterDeleted := dancok.FilterDescriptor{
		Value:     "0",
		Operator:  dancok.IsNULL,
		FieldName: "deleted_at",
		Condition: dancok.And,
	}
	param.SelectParameter.FilterDescriptors = append(param.SelectParameter.FilterDescriptors, filterDeleted)
	// filter only show the users
	filterUser := dancok.FilterDescriptor{
		Value:     string(coreenum.CTXEnumRoleUser),
		Operator:  dancok.IsEqual,
		FieldName: "role",
		Condition: dancok.And,
	}
	param.SelectParameter.FilterDescriptors = append(param.SelectParameter.FilterDescriptors, filterUser)

	// default sort
	sortDefault := dancok.SortDescriptor{
		FieldName:     r.queryGenerator.DefaultFieldForSort,
		SortDirection: dancok.Descending,
	}
	if len(param.SelectParameter.SortDescriptors) == 0 {
		param.SelectParameter.SortDescriptors = append(param.SelectParameter.SortDescriptors, sortDefault)
	}

	sqlQuery, sqlQueryCount := r.queryGenerator.Generate(param.SelectParameter, (&entity.UserEntity{}).TableName())

	queryResult := []entity.UserEntity{}
	var queryTotal int32

	err = tx.Raw(sqlQuery).Scan(&queryResult).Error
	if err != nil {
		r.Log.Error("failed get list user", "method", "UserRepository.List()", "sub_method", "tx.Raw()", "Err", err.Error())
		return
	}
	err = tx.Raw(sqlQueryCount).Scan(&queryTotal).Error
	if err != nil {
		r.Log.Error("failed get list account", "method", "UserRepository.List()", "sub_method", "tx.Raw()", "Err", err.Error())
		return
	}

	// output
	results = queryResult
	totalItems = queryTotal
	pageSize = param.SelectParameter.PageDescriptor.PageSize
	totalPages = int32(math.Ceil(float64(totalItems) / float64(pageSize)))
	return
}
