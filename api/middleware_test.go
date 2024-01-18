package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/yeom-c/golang-simplebank/token"
)

func addAuthorization(t *testing.T, request *http.Request, tokenMaker token.Maker, authorizationType string, username string, duration time.Duration) {
	token, payload, err := tokenMaker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, token)
	request.Header.Set(fiber.HeaderAuthorization, authorizationHeader)
}

func TestAuthMiddleware(t *testing.T) {
	testCases := []struct {
		name      string
		setupAuth func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkRes  func(t *testing.T, res *http.Response)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, "Bearer", "user", time.Minute)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusOK, res.StatusCode)
			},
		},
		{
			name: "NoAuthorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
		{
			name: "UnsupportedAuthorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, "Unsupported", "user", time.Minute)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
		{
			name: "InvalidAuthorizationFormat",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, "", "user", time.Minute)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
		{
			name: "ExpiredToken",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, "Bearer", "user", -time.Minute)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			server := newTestServer(t, nil)

			authPath := "/auth"
			server.app.Use(authMiddleware(server.tokenMaker))
			server.app.Get(authPath, func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{})
			})

			req := httptest.NewRequest(http.MethodGet, authPath, nil)
			tc.setupAuth(t, req, server.tokenMaker)

			res, err := server.app.Test(req)
			require.NoError(t, err)
			tc.checkRes(t, res)
		})
	}
}
