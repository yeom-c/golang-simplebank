package api

import (
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
	"github.com/yeom-c/golang-simplebank/token"
	"github.com/yeom-c/golang-simplebank/util"
)

type Server struct {
	config     util.Config
	store      db.Store
	app        *fiber.App
	validator  *validator.Validate
	tokenMaker token.Maker
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	validator := validator.New(validator.WithRequiredStructEnabled())
	validator.RegisterValidation("currency", validCurrency)
	tokenMaker, err := token.NewPasetoMaker()
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:     config,
		store:      store,
		validator:  validator,
		tokenMaker: tokenMaker,
	}
	server.app = fiber.New(fiber.Config{
		JSONEncoder:       json.Marshal,
		JSONDecoder:       json.Unmarshal,
		StreamRequestBody: true,
	})

	server.setupRouter()

	return server, nil
}

func (server *Server) setupRouter() {
	app := server.app

	app.Use(logger.New())

	app.Post("/users", server.createUser)
	app.Post("/users/login", server.loginUser)
	app.Post("/tokens/renew", server.renewAccessToken)

	app.Use(authMiddleware(server.tokenMaker))

	app.Post("/accounts", server.createAccount)
	app.Get("/accounts", server.listAccount)
	app.Get("/accounts/:id", server.getAccount)
	app.Delete("/accounts/:id", server.deleteAccount)

	app.Post("/transfers", server.createTransfer)
}

func (server *Server) Start(address string) error {
	return server.app.Listen(address)
}

func errorResponse(err error) fiber.Map {
	return fiber.Map{
		"error": err.Error(),
	}
}
