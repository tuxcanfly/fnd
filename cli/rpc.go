package cli

import (
	"net"
	"strconv"

	"google.golang.org/grpc"
)

func DialRPC() (*grpc.ClientConn, error) {
	rpcHost := "127.0.0.1"
	rpcPort := 9098
	return grpc.Dial(net.JoinHostPort(rpcHost, strconv.Itoa(rpcPort)), grpc.WithInsecure())
}
