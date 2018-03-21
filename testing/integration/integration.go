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
	"syscall"
	"time"
)

const (
	tmpTestFilePathDefault = "/tmp/gollum_test.log"
	tmpTestFilePathFoo     = "/tmp/gollum_test_foo.log"
	tmpTestFilePathBar     = "/tmp/gollum_test_bar.log"
)

// StartGollum starts the gollum binary and expects the test to call StopGollum.
func StartGollum(config string, arg ...string) (*exec.Cmd, error) {
	if config != "" {
		arg = append(arg, "-c="+getTestConfigPath(config))
	}

	cmd := GetGollumCmd(0, arg...)
	cmd.Stdout = bytes.NewBuffer([]byte{})
	err := cmd.Start()

	time.Sleep(time.Second)

	return cmd, err
}

// StopGollum stops the gollum command started with StartGollum
func StopGollum(cmd *exec.Cmd) *bytes.Buffer {
	cmd.Process.Signal(syscall.SIGTERM)
	cmd.Wait()
	buffer := (cmd.Stdout).(*bytes.Buffer)
	return buffer
}

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

func executeGollumAndGetCmd(timeout time.Duration, config string, inputs []string, arg ...string) (cmd *exec.Cmd) {
	if config != "" {
		arg = append(arg, "-c="+getTestConfigPath(config))
	}

	var stdin io.WriteCloser
	var err error

	hasInputValues := len(inputs) > 0

	cmd = GetGollumCmd(timeout, arg...)
	cmd.Stdout = bytes.NewBuffer([]byte{})

	if hasInputValues {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			panic(err)
		}
	}

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	if hasInputValues {
		for _, input := range inputs {
			io.WriteString(stdin, input+"\n")
		}
	}

	return cmd
}

// GetGollumCmd returns gollum Command
func GetGollumCmd(timeout time.Duration, arg ...string) *exec.Cmd {
	var cmd *exec.Cmd

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		time.AfterFunc(timeout, cancel)

		cmd = exec.CommandContext(ctx, "./gollum", arg...)
	} else {
		cmd = exec.Command("./gollum", arg...)
	}

	cmd.Dir = getGollumPath()

	return cmd
}

type resultFile struct {
	content string
	lines   int
}

// getResultFile returns file content as a string
func getResultFile(filepath string) (resultFile, error) {
	fileContent, lineCount, err := getResultFileData(filepath)

	// create result
	result := resultFile{}
	result.content = fileContent
	result.lines = lineCount

	return result, err
}

func getResultFileData(filepath string) (string, int, error) {
	for try := 0; ; try++ {
		fileContent, lineCount, err := getResultFileDataWithError(filepath)
		if err == nil || try >= 3 {
			return fileContent, lineCount, err
		}
		time.Sleep(time.Second)
	}
}

func getResultFileDataWithError(filepath string) (string, int, error) {
	// open file
	file, err := os.Open(filepath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	// read file
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return "", 0, err
	}

	file.Seek(0, os.SEEK_SET)
	lineCount, err := lineCounter(file)
	if err != nil {
		return "", 0, err
	}

	return string(b), lineCount, nil
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}
	hasChars := false

	for {
		c, err := r.Read(buf)

		if len(buf[:c]) > 0 {
			hasChars = true
		}

		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			if count == 0 && hasChars == true {
				count++ // one line without "\n"
			}
			return count, nil

		case err != nil:
			return count, err
		}
	}
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
