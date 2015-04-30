// Copyright 2015 trivago GmbH
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
	"log"
	"math"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
)

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
		number = number*10 + uint64(buffer[index]-'0')
		index++
	}

	return number, index
}

// RecoverShutdown will trigger a shutdown via interrupt if a panic was issued.
// Typically used as "defer RecoverShutdown()".
func RecoverShutdown() {
	if r := recover(); r != nil {
		log.Println(r)
		log.Println(string(debug.Stack()))

		// Send interrupt = clean shutdown
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(os.Interrupt)
	}
}

// ParseAddress takes an address and tries to extract the protocol from it.
// Protocols may be prepended by the "protocol://" notation.
// If no protocol is given, "tcp" is assumed.
// The first parameter returned is the address, the second denotes the protocol.
// The protocol is allways returned as lowercase string.
func ParseAddress(addr string) (address string, protocol string) {
	protocolIdx := strings.Index(addr, "://")
	if protocolIdx == -1 {
		return addr, "tcp"
	}

	return addr[protocolIdx+3:], strings.ToLower(addr[:protocolIdx])
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
