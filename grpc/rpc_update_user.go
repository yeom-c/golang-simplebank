package grpc

import (
	"context"
	"database/sql"
	"time"

	db "github.com/yeom-c/golang-simplebank/db/sqlc"
	"github.com/yeom-c/golang-simplebank/pb"
	"github.com/yeom-c/golang-simplebank/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	arg := db.UpdateUserParams{
		Username: req.GetUsername(),
		FullName: sql.NullString{String: req.GetFullName(), Valid: req.FullName != nil},
		Email:    sql.NullString{String: req.GetEmail(), Valid: req.Email != nil},
	}

	if req.Password != nil {
		hashedPassword, err := util.HashPassword(req.GetPassword())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
		}

		arg.HashedPassword = sql.NullString{String: hashedPassword, Valid: true}
		arg.PasswordChangedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	user, err := server.store.UpdateUser(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	res := &pb.UpdateUserResponse{
		User: convertUser(user),
	}
	return res, nil
}
