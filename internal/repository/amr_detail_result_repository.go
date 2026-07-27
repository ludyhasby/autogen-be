package repository

import (
	"fmt"
	"log/slog"
	"logisfy/core"
	"logisfy/internal/entity"
	"math"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type AMRDetailResultRepository struct {
	*Repository[entity.AMRDetailResultEntity]
	Log *slog.Logger
}

func NewAMRDetailResultRepository(
	log *slog.Logger,
) *AMRDetailResultRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.AMRDetailResultEntity{}).TableName(), "created_at")
	repo := NewRepositoryImpl[entity.AMRDetailResultEntity](sqlGenerator)
	return &AMRDetailResultRepository{
		Repository: repo,
		Log:        log,
	}
}

func (r *AMRDetailResultRepository) Report(tx *gorm.DB, amrID uint64, param core.QueryInfo) (results []entity.Report, totalItems int32, totalPages int32, pageSize int32, err error) {
	ctx := tx.Statement.Context

	// init tracer
	tr := otel.Tracer("repository.AMRDetailResultRepository")
	ctx, span := tr.Start(ctx, "Report()")
	defer span.End()

	// default sort
	sortDefault := dancok.SortDescriptor{
		FieldName:     "total_weighted_value",
		SortDirection: dancok.Descending,
	}
	if len(param.SelectParameter.SortDescriptors) == 0 {
		param.SelectParameter.SortDescriptors = append(param.SelectParameter.SortDescriptors, sortDefault)
	}

	param.SelectParameter.FilterDescriptors = append(
		param.SelectParameter.FilterDescriptors,
		dancok.FilterDescriptor{
			FieldName: "amr_id",
			Operator:  dancok.IsEqual,
			Value:     fmt.Sprintf("%d", amrID),
		},
	)

	conditionJoin := (&entity.AMRDetailEntity{}).TableName() + " AS ad ON " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".amr_detail_id = ad.amr_detail_id"
	selectData := "ad.amr_detail_id, ad.amr_id, ad.location_code_encrypt, ad.location_type, ad.tariff, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".v_drop, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".v_loss, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".cos_phi_kecil, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".i_loss, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".in_greater_i_max, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".over_i, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".over_v, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".reverse_power, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".unbalance_i, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".i_low_v_low, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".current_loop, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".active_p_loss, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".freeze, " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".total_weighted_value"

	sqlQuery, sqlQueryCount := r.queryGenerator.GenerateJoin(
		param.SelectParameter,
		(&entity.AMRDetailResultEntity{}).TableName(),
		conditionJoin,
		selectData,
	)

	queryResult := []entity.Report{}
	var queryTotal int32

	err = tx.WithContext(ctx).Raw(sqlQuery).Scan(&queryResult).Error
	if err != nil {
		r.Log.Error("failed get list report", "method", "AMRDetailResultRepository.Report()", "sub_method", "tx.Raw()", "Err", err.Error())
		return
	}
	err = tx.WithContext(ctx).Raw(sqlQueryCount).Scan(&queryTotal).Error
	if err != nil {
		r.Log.Error("failed get count report", "method", "AMRDetailResultRepository.Report()", "sub_method", "tx.Raw()", "Err", err.Error())
		return
	}

	// output
	results = queryResult
	totalItems = queryTotal
	pageSize = param.SelectParameter.PageDescriptor.PageSize
	totalPages = int32(math.Ceil(float64(totalItems) / float64(pageSize)))
	return
}
