package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/phuthien0308/ordering-base/simplelb"
	"github.com/phuthien0308/ordering-base/simplelb/test/server/gen"
	"github.com/phuthien0308/ordering-base/simplelog"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var exampleScheme = "simplelb"
var exampleServiceName = "sampleserver"

func main() {
	zapLogger, _ := zap.NewDevelopment()
	simplelb.Register(&simplelog.SimpleZapLogger{Logger: zapLogger}, &samplePuller{}, 10*time.Second)

	address := fmt.Sprintf("%s://%s", exampleScheme, exampleServiceName)

	cc, err := grpc.NewClient(address,
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	client := gen.NewHelloServiceClient(cc)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		resp, err := client.Hello(context.Background(), &gen.HelloRequest{Hello: line})
		if err != nil {
			panic(err)
		}
		fmt.Printf("You entered: %s\n", resp)
	}
	cc.Close()
}

type samplePuller struct {
}

// Pull implements [simplelb.AddressPuller].
func (s *samplePuller) Pull(ctx context.Context, serviceName string) ([]simplelb.Address, error) {
	return []simplelb.Address{simplelb.Address("localhost:8080"), simplelb.Address("localhost:8081")}, nil
}
