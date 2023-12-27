package api

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	mockdb "github.com/yeom-c/golang-simplebank/db/mock"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
	"github.com/yeom-c/golang-simplebank/util"
	"go.uber.org/mock/gomock"
)

func TestGetAccount(t *testing.T) {
	account := randomAccount()

	testCases := []struct {
		name       string
		accountID  any
		buildStubs func(store *mockdb.MockStore)
		checkRes   func(t *testing.T, res *http.Response)
	}{
		{
			name:      "OK",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusOK, res.StatusCode)
				requireBodyMatchAccount(t, res.Body, account)
			},
		},
		{
			name:      "NotFound",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusNotFound, res.StatusCode)
			},
		},
		{
			name:      "InternalError",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
			},
		},
		{
			name:      "InvalidID",
			accountID: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name:      "InvalidReq",
			accountID: "INVALID",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			// start test server and send request
			server := NewServer(store)

			url := fmt.Sprintf("/accounts/%v", tc.accountID)
			req := httptest.NewRequest(fiber.MethodGet, url, nil)

			res, err := server.app.Test(req)
			require.NoError(t, err)

			tc.checkRes(t, res)
		})
	}
}

func randomAccount() db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		Owner:    util.RandomOwner(),
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}
}

func requireBodyMatchAccount(t *testing.T, body io.Reader, account db.Account) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAccount db.Account
	err = json.Unmarshal(data, &gotAccount)
	require.NoError(t, err)
	require.Equal(t, account, gotAccount)
}
