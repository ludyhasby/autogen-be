package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"logisfy/internal/entity"
	"time"

	"github.com/hasifpri/dancok"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type AMRDetailRepository struct {
	*Repository[entity.AMRDetailEntity]
	Log   *slog.Logger
	SqlDB *sql.DB // digunakan untuk CopyIn via pgx
}

func NewAMRDetailRepository(
	log *slog.Logger,
	sqlDB *sql.DB,
) *AMRDetailRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.AMRDetailEntity{}).TableName(), "created_at")
	repo := NewRepositoryImpl[entity.AMRDetailEntity](sqlGenerator)
	return &AMRDetailRepository{
		Repository: repo,
		Log:        log,
		SqlDB:      sqlDB,
	}
}

func (r *AMRDetailRepository) FindBatchByAMRID(tx *gorm.DB, amrID uint64, lastAMRDetailID uint64, numberBatch int) (results []*entity.AMRDetailEntity, err error) {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.AMRDetailRepository")
	ctx, span := tr.Start(ctx, "FindByAMRID")
	defer span.End()

	if err = tx.Where("amr_id = ? AND amr_detail_id > ?", amrID, lastAMRDetailID).Order("amr_detail_id ASC").Limit(numberBatch).Find(&results).Error; err != nil {
		r.Log.Error("failed find AMR detail by amr id", "method", "AMRDetailRepository.FindByAMRID()", "sub_method", "tx.Where().Find()", "error", err.Error())
		return nil, err
	}
	return
}

func (r *AMRDetailRepository) CopyIn(ctx context.Context, entities []*entity.AMRDetailEntity, now time.Time) error {
	if len(entities) == 0 {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if r.SqlDB == nil {
		return fmt.Errorf("CopyIn: SqlDB belum di-inject ke AMRDetailRepository")
	}

	conn, err := r.SqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("CopyIn: gagal acquire conn: %w", err)
	}
	defer conn.Close()

	columns := []string{
		"amr_id",
		"location_code_encrypt",
		"location_type",
		"type_meter",
		"tariff",
		"power",
		"phase",
		"measurement_type",
		"current_l1",
		"current_l2",
		"current_l3",
		"current_max",
		"current_min",
		"current_n",
		"voltage_l1",
		"voltage_l2",
		"voltage_l3",
		"voltage_max",
		"voltage_type",
		"active_power_l1",
		"active_power_l2",
		"active_power_l3",
		"active_power_total",
		"kwh_abs_total",
		"current_angle_l1",
		"current_angle_l2",
		"current_angle_l3",
		"voltage_angle_l1",
		"voltage_angle_l2",
		"voltage_angle_l3",
		"apparent_power_l1",
		"apparent_power_l2",
		"apparent_power_l3",
		"bill_reff_kwh",
		"read_date",
		"power_factor_l1",
		"power_factor_l2",
		"power_factor_l3",
		"created_at",
	}

	rows := make([][]any, 0, len(entities))
	for _, e := range entities {
		rows = append(rows, []any{
			e.AMRID,
			e.LocationCodeEncrypt,
			e.LocationType,
			e.TypeMeter,
			e.Tariff,
			e.Power,
			e.Phase,
			e.MeasurementType,
			e.CurrentL1,
			e.CurrentL2,
			e.CurrentL3,
			e.CurrentMax,
			e.CurrentMin,
			e.CurrentN,
			e.VoltageL1,
			e.VoltageL2,
			e.VoltageL3,
			e.VoltageMax,
			e.VoltageType,
			e.ActivePowerL1,
			e.ActivePowerL2,
			e.ActivePowerL3,
			e.ActivePowerTotal,
			e.KWHAbsTotal,
			e.CurrentAngleL1,
			e.CurrentAngleL2,
			e.CurrentAngleL3,
			e.VoltageAngleL1,
			e.VoltageAngleL2,
			e.VoltageAngleL3,
			e.ApparentPowerL1,
			e.ApparentPowerL2,
			e.ApparentPowerL3,
			e.BillReffKwh,
			e.ReadDate,
			e.PowerFactorL1,
			e.PowerFactorL2,
			e.PowerFactorL3,
			now,
		})
	}

	return conn.Raw(func(driverConn any) error {
		pgxStdlibConn, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return fmt.Errorf("CopyIn: driver connection bukan *stdlib.Conn, got %T", driverConn)
		}
		pgxConn := pgxStdlibConn.Conn()
		_, err := pgxConn.CopyFrom(
			ctx,
			pgx.Identifier{"amr_detail"},
			columns,
			pgx.CopyFromRows(rows),
		)
		return err
	})
}

func (r *AMRDetailRepository) DeleteByAMRID(tx *gorm.DB, amrID uint64) error {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.AMRDetailRepository")
	_, span := tr.Start(ctx, "DeleteByAMRID()")
	defer span.End()

	if err := tx.
		Where("amr_id = ?", amrID).
		Delete(&entity.AMRDetailEntity{}).Error; err != nil {
		r.Log.Error("AMRDetailRepository.DeleteByAMRID()", "Delete()", "error", err.Error())
		return err
	}
	return nil
}
