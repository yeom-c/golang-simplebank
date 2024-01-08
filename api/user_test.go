package api

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	mockdb "github.com/yeom-c/golang-simplebank/db/mock"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
	"github.com/yeom-c/golang-simplebank/util"
	"go.uber.org/mock/gomock"
)

type eqCreateUserParamsMatcher struct {
	arg      db.CreateUserParams
	password string
}

func (e eqCreateUserParamsMatcher) Matches(x interface{}) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	err := util.CheckPasswordHash(e.password, arg.HashedPassword)
	if err != nil {
		return false
	}

	e.arg.HashedPassword = arg.HashedPassword
	return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg, password}
}

func TestCreateUser(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct {
		name       string
		body       fiber.Map
		buildStubs func(store *mockdb.MockStore)
		checkRes   func(t *testing.T, res *http.Response)
	}{
		{
			name: "OK",
			body: fiber.Map{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateUserParams{
					Username: user.Username,
					FullName: user.FullName,
					Email:    user.Email,
				}
				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).
					Times(1).
					Return(user, nil)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusOK, res.StatusCode)
				requireBodyMatchUser(t, res.Body, user)
			},
		},
		{
			name: "InternalError",
			body: fiber.Map{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
			},
		},
		{
			name: "DuplicateUsername",
			body: fiber.Map{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, &pq.Error{Code: "23505"})
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusForbidden, res.StatusCode)
			},
		},
		{
			name: "InvalidUsername",
			body: fiber.Map{
				"username":  "INVALID#",
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name: "InvalideEmail",
			body: fiber.Map{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     "INVALID",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name: "TooShortPassword",
			body: fiber.Map{
				"username":  user.Username,
				"password":  "12345",
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkRes: func(t *testing.T, res *http.Response) {
				require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name: "InvalidReq",
			body: fiber.Map{
				"username": 12345,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
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

			url := "/users"
			reqBody, err := json.Marshal(tc.body)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			res, err := server.app.Test(req)
			require.NoError(t, err)

			tc.checkRes(t, res)
		})
	}
}

func randomUser(t *testing.T) (user db.User, password string) {
	password = util.RandomString(6)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user = db.User{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}
	return
}

func requireBodyMatchUser(t *testing.T, body io.Reader, user db.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUser db.User
	err = json.Unmarshal(data, &gotUser)
	require.NoError(t, err)
	require.Equal(t, user.Username, gotUser.Username)
	require.Equal(t, user.FullName, gotUser.FullName)
	require.Equal(t, user.Email, gotUser.Email)
	require.Empty(t, gotUser.HashedPassword)
}
