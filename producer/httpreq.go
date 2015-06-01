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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"net"
	"net/http"
	"sync"
)

// HttpReq producer plugin
// Configuration example
//
//   - "producer.HttpReq":
//     Enable:  true
//     Address: ":80"
//
// The HttpReq producers sends messages that already are valid http request to a
//  given webserver.
//
// Address defines the webserver to send http requests to. Set to ":80", which
// is equal to "localhost:80" by default.
type HttpReq struct {
	core.ProducerBase
	host    string
	port    string
	address string
	listen  *shared.StopListener
}

func init() {
	shared.RuntimeType.Register(HttpReq{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *HttpReq) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	if !conf.HasValue("Address") {
		return core.NewProducerError("No Host configured for producer.HttpReq")
	}

	address := conf.GetString("Address", ":80")
	prod.host, prod.port, err = net.SplitHostPort(address)
	if err != nil {
		return err
	}

	if prod.host == "" {
		prod.host = "localhost"
	}

	prod.address = prod.host + ":" + prod.port
	return nil
}

func (prod *HttpReq) sendReq(msg core.Message) {
	requestData := bytes.NewBuffer(prod.ProducerBase.Format(msg))
	req, err := http.ReadRequest(bufio.NewReader(requestData))
	if err != nil {
		Log.Error.Print("HttpReq invalid request", err)
		return
	}

	req.URL.Host = prod.address
	req.RequestURI = ""
	req.URL.Scheme = "http"

	go func() {
		if _, err := http.DefaultClient.Do(req); err != nil {
			Log.Error.Print("HttpReq send failed: ", err)
		}
	}()
}

// Produce writes to stdout or stderr.
func (prod HttpReq) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	defer prod.WorkerDone()

	prod.DefaultControlLoop(prod.sendReq, nil)
}
