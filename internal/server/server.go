package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"aipsa-backend/internal/config"
	"aipsa-backend/internal/middleware"
	"aipsa-backend/internal/service"
	"aipsa-backend/pkg/response"
)

type Server struct {
	app    *fiber.App
	cfg    *config.Config
	logger *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})

	services, err := service.NewDependencies(cfg, logger)
	if err != nil {
		return nil, err
	}

	setupMiddleware(app, cfg)
	setupRoutes(app, services, cfg)

	return &Server{
		app:    app,
		cfg:    cfg,
		logger: logger,
	}, nil
}

func setupMiddleware(app *fiber.App, cfg *config.Config) {
	app.Use(recover.New())
	app.Use(requestid.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Internal-Key"},
		AllowCredentials: true,
	}))

	app.Use(helmet.New())

	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
	}))

	app.Use(middleware.Logger())
}

func setupRoutes(app *fiber.App, services *service.Dependencies, cfg *config.Config) {
	app.Get("/health", services.Health.Handler)

	v1 := app.Group("/v1")

	auth := v1.Group("/auth")
	auth.Post("/register", services.Auth.Register)
	auth.Post("/login", services.Auth.Login)
	auth.Post("/refresh", services.Auth.RefreshToken)
	auth.Post("/logout", services.Auth.Logout)

	protected := v1.Group("", middleware.Auth(cfg))

	protected.Get("/me", services.Users.GetMe)
	protected.Patch("/me", services.Users.UpdateMe)

	projects := protected.Group("/projects")
	projects.Get("", services.Projects.List)
	projects.Post("", services.Projects.Create)
	projects.Get("/:projectId", services.Projects.Get)
	projects.Patch("/:projectId", services.Projects.Update)
	projects.Delete("/:projectId", services.Projects.Delete)

	projects.Get("/:projectId/database", services.Database.GetInfo)
	projects.Post("/:projectId/database/reset-password", services.Database.ResetPassword)

	projects.Get("/:projectId/sql/history", services.Database.GetSQLHistory)
	projects.Post("/:projectId/sql/execute", services.Database.ExecuteSQL)

	keys := protected.Group("/keys")
	keys.Get("", services.APIKeys.List)
	keys.Post("", services.APIKeys.Create)
	keys.Delete("/:id", services.APIKeys.Delete)

	internal := app.Group("/internal", middleware.InternalAuth(cfg))
	internal.Post("/projects/provision", services.Database.Provision)
	internal.Post("/projects/delete", services.Database.DeleteProject)
}

func (s *Server) Start() error {
	s.logger.Info("starting server", "address", s.cfg.ServerHost+":"+s.cfg.ServerPort)
	return s.app.Listen(s.cfg.ServerHost + ":" + s.cfg.ServerPort)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

func errorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(response.ErrorResponse(message))
}
