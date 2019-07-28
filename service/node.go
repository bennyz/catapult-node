package service

import (
	"context"
	"log"

	node "github.com/PUMATeam/catapult-node/pb"

	"github.com/golang/protobuf/ptypes/empty"
)

type NodeService struct{}

func (ns *NodeService) StartVM(ctx context.Context, cfg *node.VmConfig) (*node.Response, error) {
	log.Println("StartVM called cfg: ", cfg)

	// TODO remove error from response
	return &node.Response{
		Status: node.Response_SUCCESSFUL,
	}, nil
}

func (ns *NodeService) StopVM(context.Context, *node.UUID) (*node.Response, error) {
	log.Println("StopVM called")
	return nil, nil
}

func (ns *NodeService) ListVMs(context.Context, *empty.Empty) (*node.VmList, error) {
	log.Println("ListVMs called")
	return nil, nil
}
