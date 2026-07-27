package config

import (
	"fmt"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func NewFiber(env *Env) *fiber.App {
	var app = fiber.New(fiber.Config{
		AppName:      env.AppName,
		ErrorHandler: NewErrorHandler(),
		Prefork:      env.PreFork,
		BodyLimit:    200 * 1024 * 1024,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Use(otelfiber.Middleware(otelfiber.WithSpanNameFormatter(func(ctx *fiber.Ctx) string {
		return fmt.Sprintf("%s - %s - %s", env.AppName, ctx.Method(), ctx.Route().Path)
	})))

	return app
}

func NewErrorHandler() fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}
		return ctx.Status(code).JSON(fiber.Map{
			"code":    code,
			"message": err.Error(),
		})
	}
}
