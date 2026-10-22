package xray

import (
	"context"
	"fmt"
	"github.com/AnotherProksY/Happ-TUI/core/utils"
	"log"
	"os/exec"
	"sync"
	"syscall"
)

type XrayCore struct {
	mu             sync.Mutex
	cmd            *exec.Cmd
	cancel         context.CancelFunc
	running        bool
	channel_closed bool
	Exited         chan error
}

// Start runs xray with config on stdin. A root core runs it as the
// unprivileged xray user with only caps as ambient capabilities.
func (x *XrayCore) Start(stdinPipe []byte, caps ...uintptr) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.running {
		return fmt.Errorf("command is already running")
	}

	xraybin, err := utils.GetXrayBin()
	if err != nil {
		return fmt.Errorf("failed to start xray %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, xraybin, "run")

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
	if err := utils.Unprivileged(cmd, caps...); err != nil {
		cancel()
		return err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to get stdin %w", err)
	}

	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}

	go func() {
		defer stdin.Close()
		_, _ = stdin.Write(stdinPipe)
	}()

	x.cmd = cmd
	x.cancel = cancel
	x.running = true

	go func() {
		err := cmd.Wait()
		x.mu.Lock()
		defer x.mu.Unlock()
		if ctx.Err() == nil {
			select {
			case x.Exited <- err:
				if x.running {
					close(x.Exited)
					x.channel_closed = true
				}
				x.running = false
			default:
				// Channel is full or no reader, don't block
			}
		}
	}()

	return nil
}

func (x *XrayCore) Stop() {
	x.mu.Lock()
	defer x.mu.Unlock()
	if !x.channel_closed {
		close(x.Exited)
		x.channel_closed = true
	}
	if x.cmd != nil && x.cmd.Process != nil {
		err := x.cmd.Process.Kill()
		if err != nil {
			log.Println("error killing process", err)
		}
	}
	if x.cancel != nil {
		x.cancel()
	}
	x.running = false
}

func (x *XrayCore) IsRunning() bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.running
}
