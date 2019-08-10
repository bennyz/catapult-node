package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"

	node "github.com/PUMATeam/catapult-node/pb"
	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
)

const (
	firecrackerBinary = "./firecracker"
	vmDataPath        = "/var/vms"
	vmLogs            = "fc-logs"
)

func runVMM(ctx context.Context, vmCfg *node.VmConfig) error {
	vmLog := filepath.Join(vmLogs, fmt.Sprintf("%s.log", vmCfg.GetVmID().GetValue()))
	vmMetrics := filepath.Join(vmLogs, fmt.Sprintf("%s-metrics.log", vmCfg.GetVmID().GetValue()))

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
	os.Remove(socketPath)

	logFifo := createFifo(vmCfg.GetVmID().GetValue(), "log")
	metricsFifo := createFifo(vmCfg.GetVmID().GetValue(), "metrics")

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
		// TODO move to a constant
		// TODO extract
		LogLevel:    "Debug",
		LogFifo:     logFifo,
		MetricsFifo: metricsFifo,
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
	go readPipe(logFifo, vmLog)
	go readPipe(metricsFifo, vmMetrics)
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

// TODO make better
func createFifo(vmId, typ string) string {
	if _, err := os.Stat(vmLogs); err != nil {
		os.Mkdir(vmLogs, os.ModeDir)
	}

	logPipe := filepath.Join(vmLogs, fmt.Sprintf("%s-%s.fifo", vmId, typ))
	os.Remove(logPipe)
	err := syscall.Mkfifo(logPipe, 0600)
	if err != nil {
		log.Errorf("Couldn't create named pipe", err)
	}
	return logPipe
}

func readPipe(path, out string) {
	pipe, _ := os.OpenFile(path, os.O_RDONLY, os.ModeNamedPipe)
	output, _ := os.Create(filepath.Join(vmLogs, out))
	var buff bytes.Buffer
	defer output.Close()
	defer pipe.Close()

	for {
		written, err := io.Copy(&buff, pipe)
		if err != nil {
			log.Error(err)
		}
		log.Infof("Written %d bytes", written)
		if buff.Len() > 0 {
			buff.WriteTo(output)
			output.Sync()
		}

		time.Sleep(1 * time.Second)
	}
}
