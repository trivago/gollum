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

package tnet

import (
	"io"
	"net"
	"strings"
	"syscall"
)

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
