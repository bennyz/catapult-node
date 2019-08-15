package service

import (
	"context"
	"fmt"

	"github.com/firecracker-microvm/firecracker-go-sdk"

	log "github.com/sirupsen/logrus"

	node "github.com/PUMATeam/catapult-node/pb"

	"github.com/golang/protobuf/ptypes/empty"
)

type NodeService struct {
	Machines map[string]*firecracker.Machine
}

// StartVM starts a firecracker VM with the provided configuration
func (ns *NodeService) StartVM(ctx context.Context, cfg *node.VmConfig) (*node.Response, error) {
	log.Debug("Starting VM", cfg.GetVmID().GetValue())
	fch := &fcHandler{
		vmID: cfg.GetVmID().GetValue(),
	}

	m, err := fch.runVMM(context.Background(), cfg, log.Logger{})
	if err != nil {
		return &node.Response{
			Status: node.Response_FAILED,
		}, err
	}

	ns.Machines[cfg.GetVmID().GetValue()] = m

	go fch.readPipe("log")
	go fch.readPipe("metrics")

	return &node.Response{
		Status: node.Response_SUCCESSFUL,
	}, nil
}

func (ns *NodeService) StopVM(ctx context.Context, uuid *node.UUID) (*node.Response, error) {
	log.Debug("StopVM called on VM ", uuid.GetValue())
	if v, ok := ns.Machines[uuid.GetValue()]; !ok {
		log.Errorf("VM %s not found", uuid.GetValue())
		return &node.Response{
			Status: node.Response_FAILED,
		}, fmt.Errorf("VM %s not found", uuid.GetValue())
	} else {
		err := v.StopVMM()
		if err != nil {
			log.Error("Failed to stop VM ", uuid.GetValue())
			return &node.Response{
				Status: node.Response_FAILED,
			}, err
		}
	}

	log.Infof("Stopped VM %s", uuid.GetValue())

	return &node.Response{
		Status: node.Response_SUCCESSFUL,
	}, nil
}

func (ns *NodeService) ListVMs(context.Context, *empty.Empty) (*node.VmList, error) {
	log.Debug("ListVMs called")
	vmList := new(node.VmList)
	uuid := &node.UUID{
		Value: "poop",
	}
	vmList.VmID = uuid
	return vmList, nil
}
