package service

import (
	"context"
	"log"

	node "github.com/PUMATeam/catapult-node/pb"

	"github.com/golang/protobuf/ptypes/empty"
)

type NodeService struct{}

func (ns *NodeService) StartVM(context.Context, *node.VmConfig) (*node.Response, error) {
	log.Println("StartVM called")
	return nil, nil
}

func (ns *NodeService) StopVM(context.Context, *node.UUID) (*node.Response, error) {
	log.Println("StopVM called")
	return nil, nil
}

func (ns *NodeService) ListVMs(context.Context, *empty.Empty) (*node.VmList, error) {
	log.Println("ListVMs called")
	return nil, nil
}
