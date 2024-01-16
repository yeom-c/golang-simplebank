package api

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	mockdb "github.com/yeom-c/golang-simplebank/db/mock"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
	"github.com/yeom-c/golang-simplebank/token"
	"github.com/yeom-c/golang-simplebank/util"
	"go.uber.org/mock/gomock"
)

func TestCreateAccount(t *testing.T) {
	user, _ := randomUser(t)
	account := randomAccount(user.Username)

	testCases := []struct {
		name       string
		body       fiber.Map
		setupAuth  func(t *testing.T, req *http.Request, tokenMaker token.Maker)
		buildStubs func(store *mockdb.MockStore)
		checkRes   func(t *testing.T, res *http.Response)
	}{
		{
			name: "OK",
			body: fiber.Map{
				"currency": account.Currency,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateAccountParams{
					Owner:    account.Owner,
					Balance:  0,
					Currency: account.Currency,
				}
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(account, nil)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusOK, res.StatusCode)
				requireBodyMatchAccount(t, res.Body, account)
			},
		},
		{
			name: "UnauthorizedUser",
			body: fiber.Map{
				"currency": account.Currency,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, "unauthorized", user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateAccountParams{
					Owner:    account.Owner,
					Balance:  0,
					Currency: account.Currency,
				}
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(arg)).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
		{
			name: "NoAuthorization",
			body: fiber.Map{
				"currency": account.Currency,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
		{
			name: "InternalError",
			body: fiber.Map{
				"currency": account.Currency,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
			},
		},
		{
			name: "InvalidCurrency",
			body: fiber.Map{
				"currency": "INVALID",
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name: "InvalidReq",
			body: fiber.Map{
				"currency": 123,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
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

			server := newTestServer(t, store)

			url := "/accounts"
			reqBody, err := json.Marshal(tc.body)
			require.NoError(t, err)

			req := httptest.NewRequest(fiber.MethodPost, url, bytes.NewReader(reqBody))
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			tc.setupAuth(t, req, server.tokenMaker)
			res, err := server.app.Test(req)
			require.NoError(t, err)

			tc.checkRes(t, res)
		})
	}
}

func TestListAccount(t *testing.T) {
	user, _ := randomUser(t)

	n := 5
	accounts := make([]db.Account, n)
	for i := 0; i < n; i++ {
		accounts[i] = randomAccount(user.Username)
	}

	type Query struct {
		pageID   int
		pageSize any
	}

	testCases := []struct {
		name       string
		query      Query
		setupAuth  func(t *testing.T, req *http.Request, tokenMaker token.Maker)
		buildStubs func(store *mockdb.MockStore)
		checkRes   func(t *testing.T, res *http.Response)
	}{
		{
			name: "OK",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListAccountsParams{
					Owner:  user.Username,
					Limit:  int32(n),
					Offset: 0,
				}
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(accounts, nil)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusOK, res.StatusCode)
				requireBodyMatchAccounts(t, res.Body, accounts)
			},
		},
		{
			name: "UnauthorizedUser",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, "unauthorized", user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListAccountsParams{
					Owner:  user.Username,
					Limit:  int32(n),
					Offset: 0,
				}
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(arg)).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
		{
			name: "NoAuthorization",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
		{
			name: "InternalError",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, "unauthorized", time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]db.Account{}, sql.ErrConnDone)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
			},
		},
		{
			name: "InvalidPageID",
			query: Query{
				pageID:   -1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, "unauthorized", time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)

			},
		},
		{
			name: "InvalidPageSize",
			query: Query{
				pageID:   1,
				pageSize: -1,
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, "unauthorized", time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)

			},
		},
		{
			name: "InvalidReq",
			query: Query{
				pageID:   1,
				pageSize: "INVALID",
			},
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, "unauthorized", time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any()).
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

			server := newTestServer(t, store)

			url := fmt.Sprintf("/accounts?page_id=%v&page_size=%v", tc.query.pageID, tc.query.pageSize)
			req := httptest.NewRequest(fiber.MethodGet, url, nil)

			tc.setupAuth(t, req, server.tokenMaker)
			res, err := server.app.Test(req)
			require.NoError(t, err)

			tc.checkRes(t, res)
		})
	}
}

func TestGetAccount(t *testing.T) {
	user, _ := randomUser(t)
	account := randomAccount(user.Username)

	testCases := []struct {
		name       string
		accountID  any
		setupAuth  func(t *testing.T, req *http.Request, tokenMaker token.Maker)
		buildStubs func(store *mockdb.MockStore)
		checkRes   func(t *testing.T, res *http.Response)
	}{
		{
			name:      "OK",
			accountID: account.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
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
			name:      "UnauthorizedUser",
			accountID: account.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, "unauthorized", time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
		{
			name:      "NoAuthorization",
			accountID: account.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusUnauthorized, res.StatusCode)
			},
		},
		{
			name:      "NotFound",
			accountID: account.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
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
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
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
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
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
			setupAuth: func(t *testing.T, req *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, req, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
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
			server := newTestServer(t, store)

			url := fmt.Sprintf("/accounts/%v", tc.accountID)
			req := httptest.NewRequest(fiber.MethodGet, url, nil)

			tc.setupAuth(t, req, server.tokenMaker)
			res, err := server.app.Test(req)
			require.NoError(t, err)

			tc.checkRes(t, res)
		})
	}
}

func randomAccount(owner string) db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		Owner:    owner,
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

func requireBodyMatchAccounts(t *testing.T, body io.Reader, accounts []db.Account) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAccounts []db.Account
	err = json.Unmarshal(data, &gotAccounts)
	require.NoError(t, err)
	require.Equal(t, accounts, gotAccounts)
}
