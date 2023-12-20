package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeom-c/golang-simplebank/util"
)

func createAccountTransfer(t *testing.T) Transfer {
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	arg := CreateTransferParams{
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        util.RandomMoney(),
	}

	transfer, err := testQueries.CreateTransfer(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	require.Equal(t, arg.FromAccountID, transfer.FromAccountID)
	require.Equal(t, arg.ToAccountID, transfer.ToAccountID)
	require.Equal(t, arg.Amount, transfer.Amount)

	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.CreatedAt)

	return transfer
}

func TestCreateTransfer(t *testing.T) {
	createAccountTransfer(t)
}

func TestGetTransfer(t *testing.T) {
	transfer := createAccountTransfer(t)
	getTransfer, err := testQueries.GetTransfer(context.Background(), transfer.ID)
	require.NoError(t, err)
	require.NotEmpty(t, getTransfer)

	require.Equal(t, transfer.ID, getTransfer.ID)
	require.Equal(t, transfer.FromAccountID, getTransfer.FromAccountID)
	require.Equal(t, transfer.ToAccountID, getTransfer.ToAccountID)
	require.Equal(t, transfer.Amount, getTransfer.Amount)

	require.WithinDuration(t, transfer.CreatedAt, getTransfer.CreatedAt, time.Second)
}

func TestUpdateTransfer(t *testing.T) {
	transfer := createAccountTransfer(t)
	arg := UpdateTransferParams{
		ID:     transfer.ID,
		Amount: util.RandomMoney(),
	}

	updatedTransfer, err := testQueries.UpdateTransfer(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedTransfer)

	require.Equal(t, transfer.ID, updatedTransfer.ID)
	require.Equal(t, transfer.FromAccountID, updatedTransfer.FromAccountID)
	require.Equal(t, transfer.ToAccountID, updatedTransfer.ToAccountID)
	require.Equal(t, arg.Amount, updatedTransfer.Amount)

	require.WithinDuration(t, transfer.CreatedAt, updatedTransfer.CreatedAt, time.Second)
}

func TestDeleteTransfer(t *testing.T) {
	transfer := createAccountTransfer(t)
	err := testQueries.DeleteTransfer(context.Background(), transfer.ID)
	require.NoError(t, err)

	getTransfer, err := testQueries.GetTransfer(context.Background(), transfer.ID)
	require.Error(t, err)
	require.Empty(t, getTransfer)
}

func TestListTransfers(t *testing.T) {
	for i := 0; i < 10; i++ {
		createAccountTransfer(t)
	}

	arg := ListTransfersParams{
		Limit:  5,
		Offset: 5,
	}

	transfers, err := testQueries.ListTransfers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
	}

	arg.Limit = -1
	transfers2, err := testQueries.ListTransfers(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, transfers2)
}
