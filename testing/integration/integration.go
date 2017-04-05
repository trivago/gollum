// Package integration performs initialization and validation for integration
// tests.
package integration

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ExecuteGollum execute gollum binary for integration testing
func ExecuteGollum(config string, inputs []string, arg ...string) (out bytes.Buffer, err error) {
	if config != "" {
		arg = append(arg, "-c="+getTestConfigPath(config))
	}

	var stdin io.WriteCloser
	timeout := 2 * time.Second
	hasInputValues := len(inputs) > 0

	cmd := GetGollumCmd(timeout, arg...)
	cmd.Stdout = &out

	if hasInputValues {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return
		}
	}

	err = cmd.Start()
	if err != nil {
		return
	}

	if hasInputValues {
		for _, input := range inputs {
			io.WriteString(stdin, input+"\n")
		}
	}

	// no error handling here because we will get a "signal killed" error
	cmd.Wait()

	return
}

// GetGollumCmd returns gollum Command
func GetGollumCmd(timeout time.Duration, arg ...string) *exec.Cmd {
	var cmd *exec.Cmd

	if timeout > 0 {
		ctx, _ := context.WithTimeout(context.Background(), timeout)
		//defer cancel()

		cmd = exec.CommandContext(ctx, "./gollum", arg...)
	} else {
		cmd = exec.Command("./gollum", arg...)
	}

	cmd.Dir = getGollumPath()

	return cmd
}

// GetFileContentAsString returns file content as a string
func GetFileContentAsString(filepath string) (string, error) {
	// read file
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(b), nil
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
