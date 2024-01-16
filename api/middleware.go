package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yeom-c/golang-simplebank/token"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authPayload"
)

func authMiddleware(tokenMaker token.Maker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authorizationHeader := c.Get(authorizationHeaderKey)
		if len(authorizationHeader) == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": errors.New("authorization header is not provided").Error(),
			})
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": errors.New("invalid authorization header format").Error(),
			})
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fmt.Errorf("unsupported authorization type %s", authorizationType).Error(),
			})
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		userCtx := context.Background()
		userCtx = context.WithValue(userCtx, authorizationPayloadKey, payload)
		c.SetUserContext(userCtx)

		return c.Next()
	}
}
