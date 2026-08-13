package worker

import (
	"context"
	"log/slog"
	"logisfy/core"
	helperconverter "logisfy/helper/converter"
	"logisfy/internal/repository"
	"strconv"
	"time"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type AMRWorker struct {
	DB            *gorm.DB
	Log           *slog.Logger
	AMRRepository *repository.AMRRepository
	Location      *time.Location
	NumberBatch   int
	CronExpr      string
}

func NewAMRWorker(db *gorm.DB, log *slog.Logger, amrRepository *repository.AMRRepository, location *time.Location, numberBatch int, cronExpr string) *AMRWorker {
	return &AMRWorker{
		DB:            db,
		Log:           log,
		AMRRepository: amrRepository,
		Location:      location,
		NumberBatch:   numberBatch,
		CronExpr:      cronExpr,
	}
}
func (worker *AMRWorker) RunDeletionWorker(ctx context.Context) {
	worker.Log.Info("worker amr deletion started...")
	sleepMinutes, err := helperconverter.CronExprToIntervalMinutes(worker.CronExpr, defaultInMinutes)
	if err != nil {
		worker.Log.Error("AMRWorker.RunDeletionWorker() failed parsing cron", "error", err.Error())
		return
	}
	sleepDuration := time.Duration(sleepMinutes) * time.Minute

	runJob := func() {
		worker.Log.Info("Running Delete Job...")

		tr := otel.Tracer("worker.AMRWorker")
		spanCtx, span := tr.Start(ctx, "Delete")
		defer span.End()

		now := time.Now().In(worker.Location)

		// get all amr that auto delete < now batch
		var lastAMRID uint64

		baseQueryInfo := core.QueryInfo{
			SelectParameter: dancok.SelectParameter{
				PageDescriptor: dancok.PageDescriptor{
					PageIndex: 1,
					PageSize:  int32(worker.NumberBatch),
				},
				SortDescriptors: []dancok.SortDescriptor{
					dancok.SortDescriptor{
						FieldName:     "amr_id",
						SortDirection: dancok.Ascending,
					},
				},
			},
		}

		tx := worker.DB.WithContext(spanCtx)
		for {
			finalQueryInfo := baseQueryInfo
			finalQueryInfo.SelectParameter.FilterDescriptors = []dancok.FilterDescriptor{
				{
					FieldName: "auto_deleted_at",
					Operator:  dancok.IsLessThanOrEqualDate,
					Value:     helperconverter.ConvertTimeToString(&now),
					Condition: dancok.And},
				{
					FieldName: "amr_id",
					Operator:  dancok.IsMoreThan,
					Value:     strconv.FormatUint(lastAMRID, 10),
					Condition: dancok.And},
			}

			amrEntities, _, _, _, err := worker.AMRRepository.List(tx, finalQueryInfo)
			if err != nil {
				worker.Log.Error("AMRWorker.RunDeletionWorker()", "AMRRepository.List()", "error", err.Error())
				return
			}
			if len(amrEntities) == 0 {
				break
			}
			lastAMRID = amrEntities[len(amrEntities)-1].AMRID

			if err = worker.AMRRepository.DeleteBatch(tx, amrEntities); err != nil {
				worker.Log.Error("AMRWorker.RunDeletionWorker()", "AMRRepository.DeleteBatch()", "error", err.Error())
				return
			}
		}

		if err = worker.AMRRepository.Vacuum(tx); err != nil {
			worker.Log.Error("AMRWorker.RunDeletionWorker()", "AMRRepository.Vacuum()",
				"error", err.Error(),
			)
		}
	}

	runJob()
	for {
		nextRun := time.Now().Add(sleepDuration)
		worker.Log.Info(
			"worker AMRWorker sleeping until next run",
			"time", nextRun.Format(time.RFC3339),
		)

		timer := time.NewTimer(sleepDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			worker.Log.Info("worker AMRWorker shutting down...")
			return
		case <-timer.C:
			runJob()
		}
	}
}
