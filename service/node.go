package service

import (
	"context"
	"log"

	node "github.com/PUMATeam/catapult-node/pb"

	"github.com/golang/protobuf/ptypes/empty"
)

type NodeService struct{}

// StartVM starts a firecracker VM with the provided configuration
func (ns *NodeService) StartVM(ctx context.Context, cfg *node.VmConfig) (*node.Response, error) {
	log.Println("StartVM called cfg: ", cfg)

	if err := runVMM(ctx, cfg); err != nil {
		return &node.Response{
			Status: node.Response_FAILED,
		}, err
	}

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
