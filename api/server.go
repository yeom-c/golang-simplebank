package api

import (
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
)

type Server struct {
	store     db.Store
	app       *fiber.App
	validator *validator.Validate
}

func NewServer(store db.Store) *Server {
	server := &Server{
		store:     store,
		validator: validator.New(validator.WithRequiredStructEnabled()),
	}
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	app.Use(logger.New())

	app.Post("/accounts", server.createAccount)
	app.Get("/accounts", server.listAccount)
	app.Get("/accounts/:id", server.getAccount)
	app.Delete("/accounts/:id", server.deleteAccount)

	server.app = app
	return server
}

func (server *Server) Start(address string) error {
	return server.app.Listen(address)
}

func errorResponse(err error) fiber.Map {
	return fiber.Map{
		"error": err.Error(),
	}
}
