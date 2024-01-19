package grpc

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rakyll/statik/fs"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"
	_ "github.com/yeom-c/golang-simplebank/doc/statik"
	"github.com/yeom-c/golang-simplebank/pb"
	"github.com/yeom-c/golang-simplebank/token"
	"github.com/yeom-c/golang-simplebank/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
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

func (s *Server) StartGateway(address string) error {
	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	grpcMux := runtime.NewServeMux(jsonOption)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := pb.RegisterSimpleBankHandlerServer(ctx, grpcMux, s)
	if err != nil {
		return err
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/", grpcMux)

	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	swaggerHandler := http.StripPrefix("/swagger", http.FileServer(statikFS))
	httpMux.Handle("/swagger/", swaggerHandler)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	log.Printf("starting HTTP gateway server on %s", listener.Addr().String())
	err = http.Serve(listener, httpMux)
	if err != nil {
		return err
	}

	return nil
}
