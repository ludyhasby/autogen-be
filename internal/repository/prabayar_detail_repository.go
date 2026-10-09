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

type PrabayarDetailRepository struct {
	*Repository[entity.PrabayarDetailEntity]
	Log   *slog.Logger
	SqlDB *sql.DB // digunakan untuk CopyIn via pgx
}

func NewPrabayarDetailRepository(
	log *slog.Logger,
	sqlDB *sql.DB,
) *PrabayarDetailRepository {
	sqlGenerator := dancok.NewSqlGenerator((&entity.PrabayarDetailEntity{}).TableName(), "created_at")
	repo := NewRepositoryImpl[entity.PrabayarDetailEntity](sqlGenerator)
	return &PrabayarDetailRepository{
		Repository: repo,
		Log:        log,
		SqlDB:      sqlDB,
	}
}

func (r *PrabayarDetailRepository) CopyIn(ctx context.Context, entities []*entity.PrabayarDetailEntity, now time.Time) error {
	if len(entities) == 0 {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if r.SqlDB == nil {
		return fmt.Errorf("CopyIn: SqlDB belum di-inject ke PrabayarDetailRepository")
	}

	conn, err := r.SqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("CopyIn: gagal acquire conn: %w", err)
	}
	defer conn.Close()

	columns := []string{
		"prabayar_id",
		"idpel_crypt",
		"no_meter_crypt",
		"type_meter",
		"tariff",
		"power",
		"read_date",
		"voltage",
		"current",
		"cosphi",
		"seal_condition",
		"lcd_condition",
		"keypad_condition",
		"terminal_amount",
		"temper_indicator",
		"relay_indicator",
		"created_at",
	}

	rows := make([][]any, 0, len(entities))
	for _, e := range entities {
		rows = append(rows, []any{
			e.PrabayarID,
			e.IDPelCrypt,
			e.NoMeterCrypt,
			e.Tariff,
			e.Power,
			e.ReadDate,
			e.Voltage,
			e.Current,
			e.Cosphi,
			e.SealCondition,
			e.LCDCondition,
			e.KeypadCondition,
			e.TerminalAmount,
			e.TemperIndicator,
			e.RelayIndicator,
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
			pgx.Identifier{"prabayar_id"},
			columns,
			pgx.CopyFromRows(rows),
		)
		return err
	})
}

func (r *PrabayarDetailRepository) DeleteByPrabayarID(tx *gorm.DB, prabayarID uint64) error {
	ctx := tx.Statement.Context

	tr := otel.Tracer("repository.PrabayarDetailRepository")
	_, span := tr.Start(ctx, "DeleteByPrabayarID()")
	defer span.End()

	if err := tx.
		Where("prabayar_id = ?", prabayarID).
		Delete(&entity.PrabayarDetailEntity{}).Error; err != nil {
		r.Log.Error("PrabayarDetailRepository.DeleteByPrabayarID()", "Delete()", "error", err.Error())
		return err
	}
	return nil
}
