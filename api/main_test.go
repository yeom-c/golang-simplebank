package api

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
	"github.com/yeom-c/golang-simplebank/util"
)

func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	return server
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
