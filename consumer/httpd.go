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

package consumer

import (
	"bytes"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Httpd consumer plugin
// Configuration example
//
//   - "consumer.Httpd":
//     Enable: true
//     Address: ":80"
//     ReadTimeoutSec: 5
//
// The Httpd consumer defines a simple http listener that will generate a
// message from the body of each POST request.
// This consumer cannot be paused.
//
// Address stores the identifier to bind to.
// This is allowed be any ip address/dns and port like "localhost:5880".
// By default this is set to ":80".
//
// ReadTimeoutSec specifies the maximum duration in seconds before timing out
// read of the request. By default this is set to 3 seconds.
type Httpd struct {
	core.ConsumerBase
	listen         *shared.StopListener
	address        string
	readTimeoutSec time.Duration
	sequence       uint64
}

func init() {
	shared.RuntimeType.Register(Httpd{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Httpd) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	cons.address = conf.GetString("Address", ":80")
	cons.readTimeoutSec = time.Duration(conf.GetInt("ReadTimeoutSec", 3)) * time.Second
	return err
}

// requestHandler will handle a single web request.
func (cons *Httpd) requestHandler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		return // ### return, requires POST ###
	}

	if req.ContentLength <= 0 {
		resp.WriteHeader(http.StatusLengthRequired)
		return // ### return, missing content length ###
	}

	body := bytes.NewBuffer(make([]byte, 0, int(req.ContentLength)))
	n, err := body.ReadFrom(req.Body)
	if err != nil || n <= 0 {
		resp.WriteHeader(http.StatusBadRequest)
		return // ### return, missing body ###
	}

	cons.Enqueue(body.Bytes(), atomic.AddUint64(&cons.sequence, 1))
	resp.WriteHeader(http.StatusCreated)
}

func (cons *Httpd) serve() {
	defer cons.WorkerDone()

	srv := http.Server{
		Addr:        cons.address,
		Handler:     http.HandlerFunc(cons.requestHandler),
		ReadTimeout: cons.readTimeoutSec,
	}

	err := srv.Serve(cons.listen)
	_, isStopRequest := err.(shared.StopRequestError)
	if err != nil && !isStopRequest {
		Log.Error.Print("httpd: ", err)
	}
}

// Consume opens a new http server listen on specified ip and port (address)
func (cons *Httpd) Consume(workers *sync.WaitGroup) {
	listen, err := shared.NewStopListener(cons.address)
	if err != nil {
		Log.Error.Print("httpd: ", err)
		return // ### return, could not connect ###
	}

	cons.listen = listen
	cons.AddMainWorker(workers)

	go cons.serve()
	defer cons.listen.Close()

	cons.DefaultControlLoop(nil)
}
