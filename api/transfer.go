package api

import (
	"database/sql"
	"fmt"

	"github.com/gofiber/fiber/v2"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
)

type transferRequest struct {
	FromAccountID int64  `json:"from_account_id" validate:"required,min=1"`
	ToAccountID   int64  `json:"to_account_id" validate:"required,min=1"`
	Amount        int64  `json:"amount" validate:"required,gt=0"`
	Currency      string `json:"currency" validate:"required,currency"`
}

func (server *Server) createTransfer(ctx *fiber.Ctx) error {
	var req transferRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	if err := server.validator.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	code, err := server.validAccount(ctx, req.FromAccountID, req.Currency)
	if err != nil {
		return ctx.Status(code).JSON(errorResponse(err))
	}

	code, err = server.validAccount(ctx, req.ToAccountID, req.Currency)
	if err != nil {
		return ctx.Status(code).JSON(errorResponse(err))
	}

	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	result, err := server.store.TransferTx(ctx.Context(), arg)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(errorResponse(err))
	}

	return ctx.JSON(result)
}

func (server *Server) validAccount(ctx *fiber.Ctx, accountID int64, currency string) (int, error) {
	account, err := server.store.GetAccount(ctx.Context(), accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fiber.StatusNotFound, err
		}
		return fiber.StatusInternalServerError, err
	}

	if account.Currency != currency {
		return fiber.StatusBadRequest, fmt.Errorf("account [%d] currency mismatch: %s vs %s", accountID, account.Currency, currency)
	}

	return fiber.StatusOK, nil
}
