package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	mockdb "github.com/yeom-c/golang-simplebank/db/mock"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
	"github.com/yeom-c/golang-simplebank/util"
	"go.uber.org/mock/gomock"
)

func TestCreateTransfer(t *testing.T) {
	amount := int64(10)

	account1 := randomAccount()
	account2 := randomAccount()
	account3 := randomAccount()

	account1.Currency = util.USD
	account2.Currency = util.USD
	account3.Currency = util.EUR

	testCases := []struct {
		name       string
		body       fiber.Map
		buildStubs func(store *mockdb.MockStore)
		checkRes   func(t *testing.T, res *http.Response)
	}{
		{
			name: "OK",
			body: fiber.Map{
				"from_account_id": account1.ID,
				"to_account_id":   account2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(account1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Times(1).Return(account2, nil)

				arg := db.TransferTxParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				}
				store.EXPECT().TransferTx(gomock.Any(), gomock.Eq(arg)).Times(1)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusOK, res.StatusCode)
			},
		},
		{
			name: "FromAccountNotFound",
			body: fiber.Map{
				"from_account_id": account1.ID,
				"to_account_id":   account2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusNotFound, res.StatusCode)
			},
		},
		{
			name: "ToAccountNotFound",
			body: fiber.Map{
				"from_account_id": account1.ID,
				"to_account_id":   account2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(account1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusNotFound, res.StatusCode)
			},
		},
		{
			name: "FromAccountCurrencyMismatch",
			body: fiber.Map{
				"from_account_id": account3.ID,
				"to_account_id":   account2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account3.ID)).Times(1).Return(account3, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name: "ToAccountCurrencyMismatch",
			body: fiber.Map{
				"from_account_id": account1.ID,
				"to_account_id":   account3.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(account1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account3.ID)).Times(1).Return(account3, nil)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name: "InvalidCurrency",
			body: fiber.Map{
				"from_account_id": account1.ID,
				"to_account_id":   account2.ID,
				"amount":          amount,
				"currency":        "INVALID",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name: "NegativeAmount",
			body: fiber.Map{
				"from_account_id": account1.ID,
				"to_account_id":   account2.ID,
				"amount":          -amount,
				"currency":        util.USD,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name: "GetAccountError",
			body: fiber.Map{
				"from_account_id": account1.ID,
				"to_account_id":   account2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(1).Return(db.Account{}, sql.ErrConnDone)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
			},
		},
		{
			name: "TransferTxError",
			body: fiber.Map{
				"from_account_id": account1.ID,
				"to_account_id":   account2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(account1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Times(1).Return(account2, nil)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(1).Return(db.TransferTxResult{}, sql.ErrTxDone)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
			},
		},
		{
			name: "InvalidReq",
			body: fiber.Map{
				"from_account_id": "INVALID",
				"to_account_id":   "INVALID",
				"amount":          "INVALID",
				"currency":        "INVALID",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
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

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/transfers"
			req := httptest.NewRequest(fiber.MethodPost, url, bytes.NewReader(data))
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			res, err := server.app.Test(req)
			require.NoError(t, err)

			tc.checkRes(t, res)
		})
	}
}
