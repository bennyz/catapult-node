package service

import (
	"bufio"
	"context"
	"fmt"
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

	vmLogFifo := filepath.Join(vmLogs, vmCfg.GetVmID().GetValue()+"-log.fifo")
	vmMetricsFifo := filepath.Join(vmLogs, vmCfg.GetVmID().GetValue()+"-metrics.fifo")

	if _, err := os.Stat(vmDataPath); err != nil {
		os.Mkdir(vmDataPath, os.ModeDir)
	}

	if _, err := os.Stat(vmLogs); err != nil {
		os.Mkdir(vmLogs, os.ModeDir)
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
		LogFifo:     vmLogFifo,
		MetricsFifo: vmMetricsFifo,
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
	go readPipe(vmLogFifo, vmLog)
	go readPipe(vmMetricsFifo, vmMetrics)

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

func readPipe(path, out string) {
	pipe, err := os.OpenFile(path, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		log.Error(err)
	}

	var output *os.File
	if _, err := os.Stat(out); err != nil {
		output, err = os.Create(out)
		if err != nil {
			log.Errorf("Failed to create log file %s, %s", out, err)
		}
	}

	defer output.Close()
	defer pipe.Close()

	reader := bufio.NewReader(pipe)
	writer := bufio.NewWriter(output)

	for {
		line, err := reader.ReadBytes('\n')
		if err == nil {
			writer.Write(line)
			writer.Flush()
		}

		time.Sleep(1 * time.Second)
	}
}
