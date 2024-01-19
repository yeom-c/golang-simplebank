package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/yeom-c/golang-simplebank/api"
	db "github.com/yeom-c/golang-simplebank/db/sqlc"

	grpcApi "github.com/yeom-c/golang-simplebank/grpc"
	"github.com/yeom-c/golang-simplebank/util"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config: ", err)
	}
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	store := db.NewStore(conn)
	startGRPCServer(config, store)
}

func startFiberServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot start http server:", err)
	}
}

func startGRPCServer(config util.Config, store db.Store) {
	server, err := grpcApi.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create grpc server:", err)
	}

	err = server.Start(config.GRPCServerAddress)
	if err != nil {
		log.Fatal("cannot start grpc server:", err)
	}
}
