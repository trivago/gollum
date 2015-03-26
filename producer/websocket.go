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
	"net/http"
	"runtime"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
)

// Websocket producer plugin
// Configuration example
//
//   - "producer.Websocket":
//     Enable:      true
//     Addr:        ":80"
//     Path:        "/"
//     ChannelSize: 1024
//
//
type Websocket struct {
	core.ProducerBase
	addr        string
	path        string
	messages    chan string
	channelSize int
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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

	prod.addr = conf.GetString("Addr", ":5000")
	prod.path = conf.GetString("Path", "/")
	prod.channelSize = conf.GetInt("ChannelSize", 1024)
	prod.messages = make(chan string, prod.channelSize)

	return nil
}

// Makes sure to close the connection to the client
func readLoop(c *websocket.Conn, signals chan bool) {
	for {
		if _, _, err := c.NextReader(); err != nil {
			c.Close()
			signals <- false
			break
		}
	}
}

func writeLoop(c *websocket.Conn, messages chan string, signals chan bool) {
	for {
		select {
		case msg := <-messages:
			if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				Log.Error.Print("Websocket error: ", err)
			}
		case sig := <-signals:
			if sig == false {
				break
			}
		default:
		}
	}
}

func (prod *Websocket) handler(w http.ResponseWriter, r *http.Request) {
	signals := make(chan bool)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Log.Error.Print(err)
		return
	}

	go readLoop(conn, signals)
	go writeLoop(conn, prod.messages, signals)
}

func (prod Websocket) printMessage(msg core.Message) {
	prod.Formatter().PrepareMessage(msg)
	prod.messages <- prod.Formatter().String()
}

func (prod Websocket) flush() {
	for prod.NextNonBlocking(prod.printMessage) {
		runtime.Gosched()
	}
}

// Produce writes to stdout or stderr.
func (prod Websocket) Produce(workers *sync.WaitGroup) {
	defer func() {
		prod.flush()
		prod.WorkerDone()
	}()

	http.HandleFunc(prod.path, prod.handler)
	// Sometimes we can`t bind on a specified port
	// e.g. another process is using it or it is not allowed by OS
	// In this case an error will be thrown
	go http.ListenAndServe(prod.addr, nil)

	prod.AddMainWorker(workers)
	prod.DefaultControlLoop(prod.printMessage, nil)
}
