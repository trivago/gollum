// Copyright 2015-2018 trivago N.V.
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

package consumer

import (
	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
	"io"
	"net"
	"syscall"
	"time"
)

const (
	proxyClientBufferGrowSize = 256
)

type proxyClient struct {
	core.AsyncMessageSource

	proxy     *Proxy
	conn      net.Conn
	connected bool
	logger    logrus.FieldLogger
}

func listenToProxyClient(conn net.Conn, proxy *Proxy) {
	defer tgo.RecoverShutdown()
	defer conn.Close()

	conn.SetDeadline(time.Time{})

	client := proxyClient{
		proxy:     proxy,
		conn:      conn,
		connected: true,
		logger:    proxy.Logger,
	}

	client.read()
}

func (client *proxyClient) hasDisconnected(err error) bool {
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

func (client *proxyClient) EnqueueResponse(msg core.Message) {
	_, err := client.conn.Write(msg.GetPayload())
	if err != nil && err != io.EOF {
		if client.hasDisconnected(err) {
			client.connected = false // ### return, connection closed ###
		}
		client.logger.Error("Write failed: ", err)
	}
}

func (client *proxyClient) sendMessage(data []byte) {
	client.proxy.Enqueue(data)
}

func (client *proxyClient) read() {
	buffer := tio.NewBufferedReader(proxyClientBufferGrowSize, client.proxy.flags, client.proxy.offset, client.proxy.delimiter)

	for client.proxy.IsActive() && client.connected {
		err := buffer.ReadAll(client.conn, client.sendMessage)

		// Handle read errors
		if err != nil && err != io.EOF {
			if client.hasDisconnected(err) {
				return // ### return, connection closed ###
			}
			client.logger.Error("Read failed: ", err)
		}
	}
}
