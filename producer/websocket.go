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

package producer

import (
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"golang.org/x/net/websocket"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
)

// Websocket producer plugin
// Configuration example
//
//   - "producer.Websocket":
//     Enable:  true
//     Address: ":80"
//     Path:    "/"
//
type Websocket struct {
	core.ProducerBase
	addr      string
	path      string
	server    websocket.Server
	clients   [2]clientList
	clientIdx uint32
	shutdown  bool
}

type clientList struct {
	conns     []*websocket.Conn
	doneCount uint32
}

func init() {
	shared.RuntimeType.Register(Websocket{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Websocket) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	prod.addr = conf.GetString("Address", ":5881")
	prod.path = conf.GetString("Path", "/")

	return nil
}

func (prod *Websocket) handshake(conf *websocket.Config, req *http.Request) error {
	if prod.shutdown {
		return fmt.Errorf("shutting down")
	}
	return nil
}

func (prod *Websocket) addConnection(conn *websocket.Conn) {
	idx := atomic.AddUint32(&prod.clientIdx, 1) >> 31

	prod.clients[idx].conns = append(prod.clients[idx].conns, conn)
	prod.clients[idx].doneCount++
	buffer := make([]byte, 8)

	// Keep alive until connection is closed
	for {
		if _, err := conn.Read(buffer); err != nil {
			conn.Close()
			break
		}
	}
}

func (prod *Websocket) pushMessage(msg core.Message) {
	prod.Formatter().PrepareMessage(msg)
	messageText := prod.Formatter().String()

	if prod.clientIdx&0x7FFFFFFF > 0 {
		// There are new clients available
		currentIdx := prod.clientIdx >> 31
		activeIdx := (currentIdx + 1) & 1

		// Store away the current client list and reset it
		activeConns := &prod.clients[activeIdx]
		oldConns := activeConns.conns
		activeConns.conns = activeConns.conns[:]
		activeConns.doneCount = 0

		// Switch new and current client list
		if currentIdx == 0 {
			currentIdx = atomic.SwapUint32(&prod.clientIdx, 1<<31)
		} else {
			currentIdx = atomic.SwapUint32(&prod.clientIdx, 0)
		}

		// Wait for new list writer to finish
		count := currentIdx & 0x7FFFFFFF
		currentIdx = currentIdx >> 31
		for prod.clients[currentIdx].doneCount != count {
			runtime.Gosched()
		}

		// Add new connections to old connections
		newConns := &prod.clients[currentIdx]
		newConns.conns = append(oldConns, newConns.conns...)
	}

	// Process the active connections
	activeIdx := ((prod.clientIdx >> 31) + 1) & 1
	activeConns := &prod.clients[activeIdx]

	for i := 0; i < len(activeConns.conns); i++ {
		client := activeConns.conns[i]
		if _, err := client.Write([]byte(messageText)); err != nil {
			activeConns.conns = append(activeConns.conns[:i], activeConns.conns[i+1:]...)
			if closeErr := client.Close(); closeErr == nil {
				Log.Error.Print("Websocket: ", err)
			}
			i--
		}
	}
}

func (prod *Websocket) flush() {
	prod.shutdown = true
	for prod.NextNonBlocking(prod.pushMessage) {
		runtime.Gosched()
	}

	for _, client := range prod.clients[0].conns {
		client.Close()
	}
	for _, client := range prod.clients[1].conns {
		client.Close()
	}
}

// Produce writes to stdout or stderr.
func (prod Websocket) Produce(workers *sync.WaitGroup) {
	prod.shutdown = false
	prod.server = websocket.Server{
		Handshake: prod.handshake,
		Handler:   prod.addConnection,
	}

	go func() {
		http.Handle(prod.path, prod.server)
		err := http.ListenAndServe(prod.addr, nil)
		if err != nil {
			Log.Error.Print("Websocket: ", err)
		}
	}()

	prod.AddMainWorker(workers)
	defer func() {
		prod.flush()
		prod.WorkerDone()
	}()
	prod.DefaultControlLoop(prod.pushMessage, nil)
}
