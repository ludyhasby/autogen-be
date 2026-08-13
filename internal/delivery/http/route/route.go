package route

import (
	"errors"
	handler "logisfy/internal/delivery/http"
	"logisfy/internal/delivery/http/middleware"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type RouteConfig struct {
	App         *fiber.App
	SecretKey   string
	AuthHandler *handler.AuthHandler
	AMRHandler  *handler.AMRHandler
	NewsHandler *handler.NewsHandler
}

func (c *RouteConfig) Setup() {
	c.SetupGuestRoute()
	c.SetupUserRoute()
	c.SetupAdminRoute()
}

func (c *RouteConfig) SetupGuestRoute() {
	c.App.Get("/health", func(c *fiber.Ctx) error {
		tr := otel.Tracer("panel-ektensi/http")
		ctx := c.UserContext()

		ctx, span := tr.Start(ctx, "HealthHandler")
		defer span.End()

		span.SetAttributes(
			attribute.String("http.route", "/health"),
			attribute.String("app.component", "health"),
		)

		// error
		if false {
			err := errors.New("simulated error")
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		return c.SendString("OK")
	})
	publicRoute := c.App.Group("/public")
	// Auth
	authRoute := publicRoute.Group("/auth")
	authRoute.Post("/register", c.AuthHandler.Register)
	authRoute.Post("/login", c.AuthHandler.Login)

	// News
	newsRoute := publicRoute.Group("/news")
	newsRoute.Get("/", c.NewsHandler.List)
}

func (c *RouteConfig) SetupUserRoute() {
	userRoute := c.App.Group("/user", middleware.AuthUser(c.SecretKey))
	userRoute.Get("/", c.AuthHandler.Find)
	// AMR
	amrRoute := userRoute.Group("/amr")
	amrRoute.Post("/", c.AMRHandler.Upload)
	amrRoute.Get("/", c.AMRHandler.List)
	amrRoute.Get("/template", c.AMRHandler.DownloadTemplate)
	amrRoute.Get("/:amr_id", c.AMRHandler.FindAMR)
	amrRoute.Delete("/:amr_id", c.AMRHandler.Delete)
	amrRoute.Get("/:amr_id/report", c.AMRHandler.Report)
	amrRoute.Get("/:amr_id/summary", c.AMRHandler.Summary)
	amrRoute.Post("/:amr_id/generate-report", c.AMRHandler.GenerateReport)
	amrRoute.Post("/:amr_id/param-config", c.AMRHandler.CreateParamConfig)
	amrRoute.Get("/:amr_id/param-config", c.AMRHandler.FindParamConfig)
	amrRoute.Post("/:amr_id/weight-config", c.AMRHandler.CreateWeightConfig)
	amrRoute.Get("/:amr_id/weight-config", c.AMRHandler.FindWeightConfig)
	amrRoute.Get("/:amr_id/export", c.AMRHandler.Export)
	amrRoute.Get("/:amr_id/export-recommendation", c.AMRHandler.ExportRecommendation)
}

func (c *RouteConfig) SetupAdminRoute() {
	adminRoute := c.App.Group("/admin", middleware.AuthAdmin(c.SecretKey))
	// Auth
	authRoute := adminRoute.Group("/auth")
	authRoute.Get("/", c.AuthHandler.List)
	authRoute.Delete("/:user_id", c.AuthHandler.Delete)
	authRoute.Put("/:user_id/activation", c.AuthHandler.Activation)
	authRoute.Put("/:user_id/deactivation", c.AuthHandler.DeActivation)
}
