package main

import (
	"../../protos"
	"../../todos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"upper.io/db.v3/sqlite"
)

const (
	port = ":9090"
)

// ConnectionURL implements a SQLite connection struct.
type ConnectionURL struct {
	Database string
	Options  map[string]string
}

var settings = sqlite.ConnectionURL{
	Database: `data/todos.db`, // Path to database file.
}

func main() {

	dbSession, err := sqlite.Open(settings)
	if err != nil {
		log.Fatalf("db.Open(): %q\n", err)
	}
	defer dbSession.Close() // Remember to close the database session.

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()

	// weitere Services kann man hier registrieren
	todo.RegisterTodoServiceServer(grpcServer, todos.Register(dbSession))

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
