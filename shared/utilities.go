// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shared

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
)

var simpleEscapeChars = strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")
var jsonEscapeChars = strings.NewReplacer("\\", "\\\\", "\"", "\\\"")

// FilesByDate implements the Sort interface by Date for os.FileInfo arrays
type FilesByDate []os.FileInfo

// Len returns the number of files in the array
func (files FilesByDate) Len() int {
	return len(files)
}

// Swap exchanges the values stored at indexes a and b
func (files FilesByDate) Swap(a, b int) {
	files[a], files[b] = files[b], files[a]
}

// Less compares the date of the files stored at a and b as in
// "[a] modified < [b] modified". If both files were created in the same second
// the file names are compared by using a lexicographic string compare.
func (files FilesByDate) Less(a, b int) bool {
	timeA, timeB := files[a].ModTime().UnixNano(), files[b].ModTime().UnixNano()
	if timeA == timeB {
		return files[a].Name() < files[b].Name()
	}
	return timeA < timeB
}

// ListFilesByDateMatching gets all files from a directory that match a given
// regular expression pattern and orders them by modification date (ascending).
// Directories and symlinks are excluded from the returned list.
func ListFilesByDateMatching(directory string, pattern string) ([]os.FileInfo, error) {
	filteredFiles := []os.FileInfo{}
	filter, err := regexp.Compile(pattern)
	if err != nil {
		return filteredFiles, err
	}

	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return filteredFiles, err
	}

	sort.Sort(FilesByDate(files))

	for _, file := range files {
		if file.IsDir() || file.Mode()&os.ModeSymlink == os.ModeSymlink {
			continue // ### continue, skip symlinks and directories ###
		}
		if filter.MatchString(file.Name()) {
			filteredFiles = append(filteredFiles, file)
		}
	}

	return filteredFiles, nil
}

// SplitPath separates a file path into directory, filename (without extension)
// and file extension (with dot). If no directory could be derived "." is
// returned as a directory. If no file extension could be derived "" is returned
// as a file extension.
func SplitPath(filePath string) (dir string, base string, ext string) {
	dir = filepath.Dir(filePath)
	ext = filepath.Ext(filePath)
	base = filepath.Base(filePath)
	base = base[:len(base)-len(ext)]
	return dir, base, ext
}

// Unescape replaces occurrences of \\n, \\r and \\t with real escape codes.
func Unescape(text string) string {
	return simpleEscapeChars.Replace(text)
}

// EscapeJSON replaces occurrences of \ and " with escaped versions.
func EscapeJSON(text string) string {
	return jsonEscapeChars.Replace(text)
}

// MaxI returns the maximum out of two integers
func MaxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Max3I returns the maximum out of three integers
func Max3I(a, b, c int) int {
	max := a
	if b > max {
		max = b
	}
	if c > max {
		max = c
	}
	return max
}

// MinI returns the minimum out of two integers
func MinI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Min3I returns the minimum out of three integers
func Min3I(a, b, c int) int {
	min := a
	if b < min {
		min = b
	}
	if c < min {
		min = c
	}
	return min
}

// ItoLen returns the length of an unsingned integer when converted to a string
func ItoLen(number uint64) int {
	switch {
	case number < 10:
		return 1
	default:
		return int(math.Log10(float64(number)) + 1)
	}
}

// Itob writes an unsigned integer to the start of a given byte buffer.
func Itob(number uint64, buffer []byte) error {
	numberLen := ItoLen(number)
	bufferLen := len(buffer)

	if numberLen > bufferLen {
		return fmt.Errorf("Number too large for buffer")
	}

	for i := numberLen - 1; i >= 0; i-- {
		buffer[i] = '0' + byte(number%10)
		number /= 10
	}

	return nil
}

// Itobe writes an unsigned integer to the end of a given byte buffer.
func Itobe(number uint64, buffer []byte) error {
	for i := len(buffer) - 1; i >= 0; i-- {
		buffer[i] = '0' + byte(number%10)
		number /= 10

		// Check here because the number 0 has to be written, too
		if number == 0 {
			return nil
		}
	}

	return fmt.Errorf("Number too large for buffer")
}

// Btoi is a fast byte buffer to unsigned int parser that reads until the first
// non-number character is found. It returns the number value as well as the
// length of the number string encountered.
// If a number could not be parsed the returned length will be 0
func Btoi(buffer []byte) (uint64, int) {
	number := uint64(0)
	index := 0
	bufferLen := len(buffer)

	for index < bufferLen && buffer[index] >= '0' && buffer[index] <= '9' {
		parsed := uint64(buffer[index] - '0')
		number = number*10 + parsed
		index++
	}

	return number, index
}

// RecoverShutdown will trigger a shutdown via interrupt if a panic was issued.
// Typically used as "defer RecoverShutdown()".
func RecoverShutdown() {
	if r := recover(); r != nil {
		log.Print("Panic triggered shutdown: ", r)
		log.Print(string(debug.Stack()))

		fmt.Println("PANIC: ", r)
		fmt.Println(string(debug.Stack()))

		// Send interrupt = clean shutdown
		// TODO: Hack
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(os.Interrupt)
	}
}

// DontPanic can be used instead of RecoverShutdown when using a function
// without any parameters. E.g. go DontPanic(myFunction)
func DontPanic(callback func()) {
	defer RecoverShutdown()
	callback()
}

// ParseAddress takes an address and tries to extract the protocol from it.
// Protocols may be prepended by the "protocol://" notation.
// If no protocol is given, "tcp" is assumed.
// The first parameter returned is the address, the second denotes the protocol.
// The protocol is allways returned as lowercase string.
func ParseAddress(addressString string) (address, protocol string) {
	protocolIdx := strings.Index(addressString, "://")
	if protocolIdx == -1 {
		return addressString, "tcp"
	}

	return addressString[protocolIdx+3:], strings.ToLower(addressString[:protocolIdx])
}

// SplitAddress splits an address of the form "protocol://host:port" into its
// components. If no protocol is given, the default protocol is used.
// This function uses net.SplitHostPort.
func SplitAddress(addressString string, defaultProtocol string) (protocol, host, port string, err error) {
	protocol = defaultProtocol
	address := addressString
	protocolIdx := strings.Index(addressString, "://")

	if protocolIdx > -1 {
		protocol = addressString[:protocolIdx]
		address = addressString[protocolIdx+3:]
	}

	host, port, err = net.SplitHostPort(address)
	return strings.ToLower(protocol), host, port, err
}

// GetMissingMethods checks if a given object implements all methods of a
// given interface. It returns the interface coverage [0..1] as well as an array
// of error messages. If the interface is correctly implemented the coverage is
// 1 and the error message array is empty.
func GetMissingMethods(objType reflect.Type, ifaceType reflect.Type) (float32, []interface{}) {
	var missing []interface{}
	if objType.Implements(ifaceType) {
		return 1.0, missing
	}

	methodCount := ifaceType.NumMethod()
	for mIdx := 0; mIdx < methodCount; mIdx++ {
		ifaceMethod := ifaceType.Method(mIdx)
		objMethod, exists := objType.MethodByName(ifaceMethod.Name)
		signatureMismatch := false

		switch {
		case !exists:
			missing = append(missing, fmt.Sprintf("Missing: \"%s\" %v", ifaceMethod.Name, ifaceMethod.Type))
			continue // ### continue, error found ###

		case ifaceMethod.Type.NumOut() != objMethod.Type.NumOut():
			signatureMismatch = true

		case ifaceMethod.Type.NumIn()+1 != objMethod.Type.NumIn():
			signatureMismatch = true

		default:
			for oIdx := 0; !signatureMismatch && oIdx < ifaceMethod.Type.NumOut(); oIdx++ {
				signatureMismatch = ifaceMethod.Type.Out(oIdx) != objMethod.Type.Out(oIdx)
			}
			for iIdx := 0; !signatureMismatch && iIdx < ifaceMethod.Type.NumIn(); iIdx++ {
				signatureMismatch = ifaceMethod.Type.In(iIdx) != objMethod.Type.In(iIdx+1)
			}
		}

		if signatureMismatch {
			missing = append(missing, fmt.Sprintf("Invalid: \"%s\" %v is not %v", ifaceMethod.Name, objMethod.Type, ifaceMethod.Type))
		}
	}

	return float32(methodCount-len(missing)) / float32(methodCount), missing
}

// IndexN returns the nth occurrence of sep in s or -1 if there is no nth
// occurrence of sep.
func IndexN(s, sep string, n int) int {
	sepIdx := 0
	for i := 0; i < n; i++ {
		nextIdx := strings.Index(s[sepIdx:], sep)
		if nextIdx == -1 {
			return -1 // ### return, not found ###
		}
		sepIdx += nextIdx + 1
	}
	return sepIdx - 1
}

// LastIndexN returns the nth occurrence of sep in s or -1 if there is no nth
// occurrence of sep. Searching is going from the end of the string to the start.
func LastIndexN(s, sep string, n int) int {
	if n == 0 {
		return -1 // ### return, nonsense ###
	}
	sepIdx := len(s)
	for i := 0; i < n; i++ {
		sepIdx = strings.LastIndex(s[:sepIdx], sep)
		if sepIdx == -1 {
			return -1 // ### return, not found ###
		}
	}
	return sepIdx
}

// IsDisconnectedError returns true if the given error is related to a
// disconnected socket.
func IsDisconnectedError(err error) bool {
	if err == io.EOF {
		return true // ### return, closed stream ###
	}

	netErr, isNetErr := err.(*net.OpError)
	if isNetErr {
		errno, isErrno := netErr.Err.(syscall.Errno)
		if isErrno {
			switch errno {
			default:
			case syscall.ECONNRESET:
				return true // ### return, close connection ###
			}
		}
	}

	return false
}
