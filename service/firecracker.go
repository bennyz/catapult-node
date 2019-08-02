package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"

	"github.com/PUMATeam/catapult-node/util"

	node "github.com/PUMATeam/catapult-node/pb"
	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	uuid "github.com/satori/go.uuid"
)

const firecrackerBinary = "./firecracker"
const unixSocketDir = "fc-sockets"

func runVMM(ctx context.Context, vmCfg *node.VmConfig) error {
	_, err := os.Stat(firecrackerBinary)
	if os.IsNotExist(err) {
		return fmt.Errorf("Binary %q does not exist: %v", firecrackerBinary, err)
	}

	if err != nil {
		return fmt.Errorf("Failed to stat binary, %q: %v", firecrackerBinary, err)
	}

	cfg := firecracker.Config{
		SocketPath:      "/tmp/fc-sock",
		KernelImagePath: "hello-vmlinux.bin",
		Drives: []models.Drive{{
			DriveID:      firecracker.String("1"),
			PathOnHost:   firecracker.String("hello-rootfs.ext4"),
			IsRootDevice: firecracker.Bool(true),
			IsReadOnly:   firecracker.Bool(false),
		}},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  vmCfg.GetVcpus(),
			MemSizeMib: vmCfg.GetMemory(),
		},
	}
	socketPath := createSocketPath(util.StringToUUID(vmCfg.GetVmID().Value))
	cmd := firecracker.VMCommandBuilder{}.
		WithBin(firecrackerBinary).
		WithSocketPath(socketPath).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		Build(ctx)

	m, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
	if err != nil {
		return fmt.Errorf("Failed creating machine: %s", err)
	}

	if err := m.Start(ctx); err != nil {
		return fmt.Errorf("failed to start machine: %v", err)
	}

	return nil
}

func createSocketPath(vmID uuid.UUID) string {
	if err := os.Mkdir(unixSocketDir, os.ModeDir); err != nil {
		log.Println("Failed to create unix socket dir", err)
	}

	socketPath := filepath.Join(unixSocketDir, vmID.String())
	return socketPath
}
