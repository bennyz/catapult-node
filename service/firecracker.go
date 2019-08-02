package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"

	node "github.com/PUMATeam/catapult-node/pb"
	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
)

const (
	firecrackerBinary = "./firecracker"
	vmDataPath        = "/var/vms/"
)

func runVMM(ctx context.Context, vmCfg *node.VmConfig) error {
	if _, err := os.Stat(vmDataPath); err != nil {
		os.Mkdir(vmDataPath, os.ModeDir)
	}

	_, err := os.Stat(firecrackerBinary)
	if os.IsNotExist(err) {
		return fmt.Errorf("Binary %q does not exist: %v", firecrackerBinary, err)
	}

	if err != nil {
		return fmt.Errorf("Failed to stat binary, %q: %v", firecrackerBinary, err)
	}
	socketPath := filepath.Join(vmDataPath, vmCfg.GetVmID().GetValue())
	cfg := firecracker.Config{
		KernelImagePath: vmCfg.GetKernelImage(),
		SocketPath:      socketPath,
		Drives: []models.Drive{{
			DriveID:      firecracker.String("1"),
			PathOnHost:   firecracker.String(vmCfg.GetRootFileSystem()),
			IsRootDevice: firecracker.Bool(true),
			IsReadOnly:   firecracker.Bool(false),
		}},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  vmCfg.GetVcpus(),
			MemSizeMib: vmCfg.GetMemory(),
		},
		LogLevel: "Debug",

		// TODO extract
		LogFifo: filepath.Join(vmDataPath, vmCfg.GetVmID().GetValue()+".log"),
	}

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

	// TODO why is this needed?
	defer m.StopVMM()
	installSignalHandlers(ctx, m)

	if err := m.Wait(ctx); err != nil {
		return fmt.Errorf("wait returned an error %s", err)
	}

	return nil
}

func installSignalHandlers(ctx context.Context, m *firecracker.Machine) {
	go func() {
		// Clear some default handlers installed by the firecracker SDK:
		signal.Reset(os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

		for {
			switch s := <-c; {
			case s == syscall.SIGTERM || s == os.Interrupt:
				log.Printf("Caught SIGINT, requesting clean shutdown")
				m.Shutdown(ctx)
			case s == syscall.SIGQUIT:
				log.Printf("Caught SIGTERM, forcing shutdown")
				m.StopVMM()
			}
		}
	}()
}
