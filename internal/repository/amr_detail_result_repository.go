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

func (r *AMRDetailResultRepository) Report(tx *gorm.DB, amrID uint64, minWeightedValue float64, param core.QueryInfo) (results []entity.Report, totalItems int32, totalPages int32, pageSize int32, err error) {
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
		dancok.FilterDescriptor{
			FieldName: "total_weighted_value",
			Operator:  dancok.IsMoreThanOrEqual,
			Value:     fmt.Sprintf("%f", minWeightedValue),
		},
	)

	conditionJoin := (&entity.AMRDetailEntity{}).TableName() + " AS ad ON " +
		(&entity.AMRDetailResultEntity{}).TableName() + ".amr_detail_id = ad.amr_detail_id"
	selectData := "ad.*, " +
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

func (r *AMRDetailResultRepository) Summary(tx *gorm.DB, amrID uint64) (result entity.SummaryAMRDetailResult, err error) {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.AMRDetailResultRepository")
	ctx, span := tr.Start(ctx, "Summary()")
	defer span.End()

	err = tx.WithContext(ctx).
		Table(entity.AMRDetailResultEntity{}.TableName()+" AS adr").
		Select(`
            COALESCE(SUM(adr.v_drop::int), 0) AS total_v_drop,
            COALESCE(SUM(adr.v_loss::int), 0) AS total_v_loss,
            COALESCE(SUM(adr.cos_phi_kecil::int), 0) AS total_cos_phi_kecil,
            COALESCE(SUM(adr.i_loss::int), 0) AS total_i_loss,
            COALESCE(SUM(adr.over_i::int), 0) AS total_over_i,
            COALESCE(SUM(adr.over_v::int), 0) AS total_over_v,
            COALESCE(SUM(adr.unbalance_i::int), 0) AS total_unbalance_i,
            COALESCE(SUM(adr.i_low_v_low::int), 0) AS total_i_low_v_low,
            COALESCE(SUM(adr.current_loop::int), 0) AS total_current_loop,
            COALESCE(SUM(adr.active_p_loss::int), 0) AS total_active_p_loss,
            COALESCE(SUM(adr.freeze::int), 0) AS total_freeze,
            COALESCE(SUM(adr.in_greater_i_max::int), 0) AS total_in_greater_i_max,
            COALESCE(SUM(adr.reverse_power::int), 0) AS total_reverse_power`).
		Joins(`INNER JOIN `+entity.AMRDetailEntity{}.TableName()+` AS ad ON ad.amr_detail_id = adr.amr_detail_id`).
		Where("ad.amr_id = ?", amrID).
		Scan(&result).Error

	if err != nil {
		r.Log.Error("failed to get AMR detail summary", "method", "AMRDetailResultRepository.Summary()", "sub_method", "tx.Scan()", "amrID", amrID, "error", err.Error())
		return result, err
	}

	return result, nil
}
