package api

import (
	"fmt"
	"net"
	"os"

	log "github.com/sirupsen/logrus"

	node "github.com/PUMATeam/catapult-node/pb"
	"github.com/PUMATeam/catapult-node/service"

	"google.golang.org/grpc"
)

func init() {
	// TODO make configurable
	f, err := os.OpenFile("catapult-node.log", os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)
}

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
