// +build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

const (
	maxRunTime          = 20 * time.Second
	maxStartupWaitTime  = 2 * time.Second
	maxShutdownWaitTime = 10 * time.Second
	maxFetchResultTime  = 3 * time.Second
)

// GollumHandle provides access to a gollum process
type GollumHandle struct {
	cmd    *exec.Cmd
	out    *syncBuffer
	in     io.WriteCloser
	cancel context.CancelFunc
}

// StartGollum launches gollum and waits until the given start indicator shows
// up in the stdout buffer.
func StartGollum(config string, indicator StartIndicator, arg ...string) (*GollumHandle, error) {
	if config != "" {
		arg = append(arg, "-c="+getTestConfigPath(config))
	}

	handle, err := newGollumHandle(arg...)
	if err != nil {
		return nil, err
	}

	err = handle.cmd.Start()
	if err != nil {
		return nil, err
	}

	return handle, indicator.HasStarted(handle)
}

func newGollumHandle(arg ...string) (*GollumHandle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), maxRunTime)
	out := newSyncBuffer(4096)

	cmd := exec.CommandContext(ctx, "./gollum", arg...)
	cmd.Dir = getGollumPath()
	cmd.Stdout = out

	in, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}

	return &GollumHandle{
		cmd:    cmd,
		out:    out,
		in:     in,
		cancel: cancel,
	}, nil
}

// SendStdIn sends the given strings to StdIn.
// The first parameter defines the time to wait for the arguments to be
// processed by the pipeline.
func (g *GollumHandle) SendStdIn(wait time.Duration, args ...string) error {
	if !g.IsRunning() {
		return fmt.Errorf("Gollum process is not running")
	}

	for _, arg := range args {
		if _, err := io.WriteString(g.in, arg+"\n"); err != nil {
			return err
		}
	}

	// Make sure everything has been passed to gollum
	time.Sleep(wait)
	return nil
}

// ReadStdOut returns the current state of the stdout buffer
func (g *GollumHandle) ReadStdOut() string {
	return g.out.Read()
}

// StopAfter calls Stop after the given duration.
// This function blocks while waiting.
func (g *GollumHandle) StopAfter(d time.Duration) error {
	if !g.IsRunning() {
		return nil
	}

	time.Sleep(d)
	return g.Stop()
}

// Stop tries to gracefully stop gollum by CTRL+C followed by Wait()
func (g *GollumHandle) Stop() error {
	if !g.IsRunning() {
		return nil
	}

	g.cmd.Process.Signal(syscall.SIGINT)
	return g.Wait()
}

// Wait waits for gollum to stop without explicitly sending a stop signal.
// This function may time out but is blocking while waiting.
func (g *GollumHandle) Wait() error {
	if !g.IsRunning() {
		return nil
	}

	done := make(chan struct{})
	go func() {
		g.cmd.Wait()
		g.in.Close()
		close(done)
	}()

	select {
	case <-done:
		return nil

	case <-time.After(maxShutdownWaitTime):
		g.cmd.Process.Kill()
		return fmt.Errorf("Timed out waiting for gollum to stop")
	}
}

// IsRunning checks if the gollung cmd is running
func (g *GollumHandle) IsRunning() bool {
	if g.cmd.ProcessState != nil {
		return !g.cmd.ProcessState.Exited()
	}

	_, err := os.FindProcess(g.cmd.Process.Pid)
	return err == nil
}

func getGollumPath() (pwd string) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	dir, err := filepath.Abs(pwd + "/../../")
	if err != nil {
		panic(err)
	}

	return dir
}

func getTestConfigPath(configFile string) string {
	baseDir := getGollumPath()

	dir, err := filepath.Abs(baseDir + "/testing/configs/")
	if err != nil {
		panic(err)
	}

	return dir + "/" + configFile
}
