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

const (
	TmpTestFilePathDefault  = "/tmp/gollum_test.log"
	TmpTestFilePathFoo    	= "/tmp/gollum_test_foo.log"
	TmpTestFilePathBar    	= "/tmp/gollum_test_bar.log"
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

type ResultFile struct {
	content string
	lines int
}

// GetResultFile returns file content as a string
func GetResultFile(filepath string) (ResultFile, error) {
	result := ResultFile{}

	// open file
	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// read file
	b, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	file.Seek(0, os.SEEK_SET)
	lineCount, err := lineCounter(file)
	if err != nil {
		panic(err)
	}

	// create result
	result.content = string(b)
	result.lines = lineCount

	return result, nil
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
				count += 1 // one line without "\n"
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
