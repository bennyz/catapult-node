package api

import (
	"fmt"
	"log"
	"net"

	node "github.com/PUMATeam/catapult-node/pb"
	"github.com/PUMATeam/catapult-node/service"

	"google.golang.org/grpc"
)

// Start starts catapult node server
func Start(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()

	node.RegisterNodeServer(server, &service.NodeService{})
	server.Serve(lis)
}
