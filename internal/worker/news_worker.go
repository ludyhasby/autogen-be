package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"logisfy/core"
	helperconverter "logisfy/helper/converter"
	"logisfy/internal/entity"
	modelresponse "logisfy/internal/model/response"
	"logisfy/internal/repository"
	"net/http"
	"time"

	"github.com/hasifpri/dancok"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type NewsWorker struct {
	DB             *gorm.DB
	Log            *slog.Logger
	Location       *time.Location
	NewsRepository *repository.NewsRepository
	NewsAPI        string
	CronExpr       string
}

func NewNewsWorker(db *gorm.DB, log *slog.Logger, location *time.Location, newsRepository *repository.NewsRepository, newsAPI, cronExpr string) *NewsWorker {
	return &NewsWorker{
		DB:             db,
		Log:            log,
		Location:       location,
		NewsRepository: newsRepository,
		NewsAPI:        newsAPI,
		CronExpr:       cronExpr,
	}
}

const defaultInMinutes = 60 * 24

func (worker *NewsWorker) FetchNewNews(ctx context.Context) {
	slog.Info("worker news worker started...")
	queryInfo := core.QueryInfo{
		SelectParameter: dancok.SelectParameter{
			PageDescriptor: dancok.PageDescriptor{
				PageIndex: 1,
				PageSize:  -1,
			},
		},
	}
	client := &http.Client{Timeout: 15 * time.Second}

	runJob := func() {
		slog.Info("Running FetchNewNews job...")

		// init tracer
		tr := otel.Tracer("worker.NewsWorker")
		var (
			span trace.Span
			news modelresponse.NewsFetchResponse
			err  error
		)
		spanCtx, span := tr.Start(ctx, "FetchNewNews()")
		defer span.End()

		req, errReq := http.NewRequestWithContext(spanCtx, http.MethodGet, "https://www.cnnindonesia.com/api/v3/search?query=pln&idtype=1&start=0&limit=20", nil)
		if errReq != nil {
			worker.Log.Error("NewsWorker.FetchNewNews()", "http.NewRequestWithContext()", "error", errReq.Error())
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			worker.Log.Error("NewsWorker.FetchNewNews()", "client.Do()", "error", err.Error())
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			worker.Log.Error("NewsWorker.FetchNewNews()", "HTTP status", "status", resp.StatusCode)
			return
		}

		if err = json.NewDecoder(resp.Body).Decode(&news); err != nil {
			worker.Log.Error("NewsWorker.FetchNewNews()", "json.NewDecoder().Decode()", "error", err.Error())
			return
		}
		nNew := len(news.Data)

		tx := worker.DB.WithContext(spanCtx)

		entityNewsList, nExisting, _, _, err := worker.NewsRepository.List(tx, queryInfo)
		if err != nil {
			worker.Log.Error("NewsWorker.FetchNewNews()", "NewsRepository.List()", "error", err.Error())
			return
		}

		var txErr error
		if tx = tx.Begin(); tx.Error != nil {
			worker.Log.Error("NewsWorker.FetchNewNews()", "tx.Begin()", "error", tx.Error.Error())
			return
		}
		defer func() {
			if txErr != nil {
				tx.Rollback()
			}
		}()

		nDeleted := max(nNew+int(nExisting)-20, 0)
		if nDeleted > len(entityNewsList) {
			nDeleted = len(entityNewsList)
		}
		if nDeleted > 0 && nNew > 0 {
			for i := 0; i < nDeleted; i++ {
				entityNews := entityNewsList[len(entityNewsList)-1-i]
				if txErr = worker.NewsRepository.Delete(tx, &entityNews); txErr != nil {
					worker.Log.Error("NewsWorker.FetchNewNews()", "NewsRepository.Delete()", "error", txErr.Error())
					return
				}
			}
		}

		entityNewNews := (&entity.NewsEntity{}).ConvertToEntity(news, entityNewsList[:len(entityNewsList)-nDeleted], worker.Location)

		if len(entityNewNews) > 0 {
			if txErr = worker.NewsRepository.CreateBatch(tx, entityNewNews); txErr != nil {
				worker.Log.Error("NewsWorker.FetchNewNews()", "NewsRepository.CreateBatch()", "error", txErr.Error())
				return
			}
		}

		if txErr = tx.Commit().Error; txErr != nil {
			worker.Log.Error("NewsWorker.FetchNewNews()", "tx.Commit()", "error", txErr.Error())
			return
		}
	}

	runJob()
	for {
		sleepMinutes, err := helperconverter.CronExprToIntervalMinutes(worker.CronExpr, defaultInMinutes)
		if err != nil {
			worker.Log.Error("NewsWorker.FetchNewNews()", "helperconverter.CronExprToIntervalMinutes()", "error", err.Error())
			return
		}

		sleepDuration := time.Duration(sleepMinutes) * time.Minute
		nextRun := time.Now().Add(sleepDuration)
		slog.Info(
			"worker FetchNewNews sleeping until next run",
			"time", nextRun.Format(time.RFC3339),
		)
		select {
		case <-ctx.Done():
			slog.Info("Worker FetchNewNews shutting down...")
			return
		case <-time.After(sleepDuration):
			runJob()
		}
	}
}
