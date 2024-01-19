package grpc

import (
	"log"
	"net"

	db "github.com/yeom-c/golang-simplebank/db/sqlc"
	"github.com/yeom-c/golang-simplebank/pb"
	"github.com/yeom-c/golang-simplebank/token"
	"github.com/yeom-c/golang-simplebank/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	pb.UnimplementedSimpleBankServer
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker()
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	return server, nil
}

func (s *Server) Start(address string) error {
	grpcServer := grpc.NewServer()
	pb.RegisterSimpleBankServer(grpcServer, s)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	log.Printf("starting gRPC server on %s", listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil {
		return err
	}

	return nil
}
