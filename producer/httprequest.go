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
	"bufio"
	"bytes"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"net/http"
	"sync"
)

// HTTPRequest producer plugin
// Configuration example
//
//   - "producer.HTTPRequest":
//     Enable:  true
//     RawData: true
//     Encoding: "text/plain; charset=utf-8"
//     Address: "localhost:80"
//
// The HTTPRequest producers sends messages as HTTP packet to a given webserver.
// This producer uses a fuse breaker when a request fails with an error
// code > 400 or the connection is down.
//
// Address defines the webserver to send http requests to. Set to "localhost:80"
// by default.
//
// RawData switches between creating POST data from the incoming message (false)
// and passing the message as HTTP request without changes (true).
// This setting is enabled by default.
//
// Encoding defines the payload encoding when RawData is set to false.
// Set to "text/plain; charset=utf-8" by default.
type HTTPRequest struct {
	core.ProducerBase
	host       string
	port       string
	protocol   string
	address    string
	encoding   string
	rawPackets bool
	listen     *shared.StopListener
}

func init() {
	shared.TypeRegistry.Register(HTTPRequest{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *HTTPRequest) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	address := conf.GetString("Address", "localhost:80")
	prod.protocol, prod.host, prod.port, err = shared.SplitAddress(address, "http")
	if err != nil {
		return err
	}

	if prod.host == "" {
		prod.host = "localhost"
	}

	prod.address = fmt.Sprintf("%s://%s:%s", prod.protocol, prod.host, prod.port)
	prod.encoding = conf.GetString("Encoding", "text/plain; charset=utf-8")
	prod.rawPackets = conf.GetBool("RawData", true)
	prod.SetCheckFuseCallback(prod.isHostUp)
	return nil
}

func (prod *HTTPRequest) isHostUp() bool {
	resp, err := http.Get(prod.address)
	return err != nil && resp != nil && resp.StatusCode < 400
}

func (prod *HTTPRequest) sendReq(msg core.Message) {
	var (
		req *http.Request
		err error
	)

	data, _ := prod.ProducerBase.Format(msg)
	requestData := bytes.NewBuffer(data)

	if prod.rawPackets {
		// Pass raw request
		req, err = http.ReadRequest(bufio.NewReader(requestData))

		if req != nil {
			req.URL.Host = prod.address
			req.RequestURI = ""
			req.URL.Scheme = prod.protocol
		}
	} else {
		// Convert to POST request
		req, err = http.NewRequest("post", prod.address, requestData)
		if req != nil {
			req.Header.Add("content-type", prod.encoding)
		}
	}

	if err != nil {
		Log.Error.Print("HTTPRequest invalid request", err)
		prod.Drop(msg)
		return // ### return, malformed request ###
	}

	go func() {
		if _, err := http.DefaultClient.Do(req); err != nil {
			Log.Error.Print("HTTPRequest send failed: ", err)
			if !prod.isHostUp() {
				prod.Control() <- core.PluginControlFuseBurn
			}
			prod.Drop(msg)
		} else {
			prod.Control() <- core.PluginControlFuseActive
		}
	}()
}

func (prod *HTTPRequest) close() {
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.sendReq)
}

// Produce writes to stdout or stderr.
func (prod *HTTPRequest) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.sendReq)
}
