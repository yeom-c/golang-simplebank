package api

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
)

type createAccountRequest struct {
	Owner    string `json:"owner" validate:"required"`
	Currency string `json:"currency" validate:"required,currency"`
}

func (server *Server) createAccount(ctx *fiber.Ctx) error {
	var req createAccountRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	if err := server.validator.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	arg := db.CreateAccountParams{
		Owner:    req.Owner,
		Currency: req.Currency,
		Balance:  0,
	}
	account, err := server.store.CreateAccount(ctx.Context(), arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "foreign_key_violation", "unique_violation":
				return ctx.Status(fiber.StatusForbidden).JSON(errorResponse(err))
			}
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(errorResponse(err))
	}

	return ctx.JSON(account)
}

type listAccountRequest struct {
	PageID   int32 `query:"page_id" validate:"required,min=1"`
	PageSize int32 `query:"page_size" validate:"required,min=5,max=10"`
}

func (server *Server) listAccount(ctx *fiber.Ctx) error {
	var req listAccountRequest
	if err := ctx.QueryParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	if err := server.validator.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	arg := db.ListAccountsParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}
	accounts, err := server.store.ListAccounts(ctx.Context(), arg)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(errorResponse(err))
	}

	return ctx.JSON(accounts)
}

type getAccountRequest struct {
	ID int64 `params:"id" validate:"required,min=1"`
}

func (server *Server) getAccount(ctx *fiber.Ctx) error {
	var req getAccountRequest
	if err := ctx.ParamsParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	if err := server.validator.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	account, err := server.store.GetAccount(ctx.Context(), req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx.Status(fiber.StatusNotFound).JSON(errorResponse(err))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(errorResponse(err))
	}

	return ctx.JSON(account)
}

type deleteAccountRequest struct {
	ID int64 `params:"id" validate:"required,min=1"`
}

func (server *Server) deleteAccount(ctx *fiber.Ctx) error {
	var req deleteAccountRequest
	if err := ctx.ParamsParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	if err := server.validator.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	err := server.store.DeleteAccount(ctx.Context(), req.ID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(errorResponse(err))
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
