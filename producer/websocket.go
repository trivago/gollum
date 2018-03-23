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

package producer

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tnet"
	"github.com/trivago/tgo/tsync"
)

// Websocket producer plugin
//
// The websocket producer opens up a websocket.
//
// Parameters
//
// - Address: This value defines the host and port to bind to.
// This is allowed be any ip address/dns and port like "localhost:5880".
// By default this parameter is set to ":81".
//
// - Path: This value defines the url path to listen for.
// By default this parameter is set to "/"
//
// - ReadTimeoutSec: This value specifies the maximum duration in seconds before timing out
// read of the request.
// By default this parameter is set to "3" seconds.
//
// - IgnoreOrigin: Ignore origin check from websocket server.
// By default this parameter is set to "false".
//
// Examples
//
// This example starts a default Websocket producer on port 8080:
//
//  WebsocketOut:
//    Type: producer.Websocket
//    Address: ":8080"
//
type Websocket struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	listen                *tnet.StopListener
	readTimeoutSec        time.Duration `config:"ReadTimeoutSec" default:"3" metric:"sec"`
	upgrader              websocket.Upgrader
	clients               [2]clientList
	address               string `config:"Address" default:":81"`
	path                  string `config:"Path" default:"/"`
	clientIdx             uint32
	ignoreOrigin          bool `config:"IgnoreOrigin" default:"false"`
}

type clientList struct {
	conns     []*websocket.Conn
	doneCount uint32
}

func init() {
	core.TypeRegistry.Register(Websocket{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Websocket) Configure(conf core.PluginConfigReader) {
	prod.SetStopCallback(prod.close)

	prod.upgrader = websocket.Upgrader{}

	if prod.ignoreOrigin {
		prod.upgrader.CheckOrigin = func(r *http.Request) bool { return prod.ignoreOrigin }
	}
}

func (prod *Websocket) handleConnection(conn *websocket.Conn) {
	idx := atomic.AddUint32(&prod.clientIdx, 1) >> 31

	prod.clients[idx].conns = append(prod.clients[idx].conns, conn)
	prod.clients[idx].doneCount++
	conn.SetReadDeadline(time.Time{})

	// Keep alive until connection is closed
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			conn.Close()
			break
		}
	}
}

func (prod *Websocket) pushMessage(msg *core.Message) {
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
		spin := tsync.NewSpinner(tsync.SpinPriorityHigh)

		for prod.clients[currentIdx].doneCount != count {
			spin.Yield()
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
		if err := client.WriteMessage(websocket.TextMessage, msg.GetPayload()); err != nil {
			activeConns.conns = append(activeConns.conns[:i], activeConns.conns[i+1:]...)
			if closeErr := client.Close(); closeErr == nil {
				prod.Logger.Error(err)
			}
			i--
		}
	}
}

func (prod *Websocket) upgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := prod.upgrader.Upgrade(w, r, nil)
	if err != nil {
		prod.Logger.Error("Websocket: ", err)
		// Return here to not track invalid connections
		return
	}
	prod.handleConnection(conn)
}

func (prod *Websocket) serve() {
	defer prod.WorkerDone()

	listen, err := tnet.NewStopListener(prod.address)
	if err != nil {
		prod.Logger.Error(err)
		return // ### return, could not connect ###
	}

	http.HandleFunc(prod.path, prod.upgrade)

	srv := http.Server{
		ReadTimeout: prod.readTimeoutSec,
	}

	prod.listen = listen

	err = srv.Serve(prod.listen)
	_, isStopRequest := err.(tnet.StopRequestError)
	if err != nil && !isStopRequest {
		prod.Logger.Error(err)
	}
}

func (prod *Websocket) close() {
	prod.DefaultClose()
	prod.listen.Close()

	for _, client := range prod.clients[0].conns {
		client.Close()
	}
	for _, client := range prod.clients[1].conns {
		client.Close()
	}
}

// Produce writes to stdout or stderr.
func (prod *Websocket) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	go prod.serve()
	prod.MessageControlLoop(prod.pushMessage)
}
