package api

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
)

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Server) renewAccessToken(ctx *fiber.Ctx) error {
	var req renewAccessTokenRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	if err := server.validator.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(errorResponse(err))
	}

	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(errorResponse(err))
	}

	session, err := server.store.GetSession(ctx.Context(), refreshPayload.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx.Status(fiber.StatusNotFound).JSON(errorResponse(err))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(errorResponse(err))
	}

	if session.IsBlocked {
		err := errors.New("session is blocked")
		return ctx.Status(fiber.StatusUnauthorized).JSON(errorResponse(err))
	}

	if session.Username != refreshPayload.Username {
		err := errors.New("incorrect session user")
		return ctx.Status(fiber.StatusUnauthorized).JSON(errorResponse(err))
	}

	if session.RefreshToken != req.RefreshToken {
		err := errors.New("mismatched session token")
		return ctx.Status(fiber.StatusUnauthorized).JSON(errorResponse(err))
	}

	if time.Now().After(session.ExpiresAt) {
		err := errors.New("session expired")
		return ctx.Status(fiber.StatusUnauthorized).JSON(errorResponse(err))
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		refreshPayload.Username,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(errorResponse(err))
	}

	res := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiresAt,
	}
	return ctx.JSON(res)
}
