package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	gen "github.com/phuthien0308/ordering-base/simplelb/test/server/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	port := os.Getenv("PORT")
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", port))
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	gen.RegisterHelloServiceServer(server, &helloService{port: port})
	fmt.Println("running at port", port)
	if err := server.Serve(lis); err != nil {
		panic(err)
	}
}

type helloService struct {
	gen.UnimplementedHelloServiceServer
	port string
}

func (h *helloService) Hello(ctx context.Context, req *gen.HelloRequest) (*gen.HelloResponse, error) {
	return &gen.HelloResponse{Response: fmt.Sprintf("I am running at port %v, %v", h.port, req.Hello)}, nil
}

func (h *helloService) HelloStream(req *gen.HelloRequest, res grpc.ServerStreamingServer[gen.HelloResponse]) error {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		err := res.Send(&gen.HelloResponse{Response: fmt.Sprintf("I am running at port %v, %v", h.port, req.Hello)})
		if err != nil {
			return err
		}
	}
	return nil
}
