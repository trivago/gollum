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

// Http consumer plugin
// Configuration example
//
//   - "consumer.Http":
//     Enable: true
//     Address: ":80"
//     ReadTimeoutSec: 3
//     WithHeaders: true
//     Stream:
//       - "http"
//
// Address stores the identifier to bind to.
// This is allowed be any ip address/dns and port like "localhost:5880".
// By default this is set to ":80".
//
// ReadTimeoutSec specifies the maximum duration in seconds before timing out
// read of the request. By default this is set to 3 seconds.
//
// WithHeaders can be set to false to only read the HTTP body instead of passing
// the while HTTP message. By default this setting is set to true.
type Http struct {
	core.ConsumerBase
	listen         *shared.StopListener
	address        string
	sequence       uint64
	readTimeoutSec time.Duration
	withHeaders    bool
}

func init() {
	shared.TypeRegistry.Register(Http{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Http) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	cons.address = conf.GetString("Address", ":80")
	cons.readTimeoutSec = time.Duration(conf.GetInt("ReadTimeoutSec", 3)) * time.Second
	cons.withHeaders = conf.GetBool("WithHeaders", true)
	return err
}

// requestHandler will handle a single web request.
func (cons *Http) requestHandler(resp http.ResponseWriter, req *http.Request) {
	if cons.IsFuseBurned() {
		resp.WriteHeader(http.StatusServiceUnavailable)
		return // ### return, service is down ###
	}

	if cons.withHeaders {
		// Read the whole package
		requestBuffer := bytes.NewBuffer(nil)
		if err := req.Write(requestBuffer); err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			return // ### return, missing body or bad write ###
		}

		cons.Enqueue(requestBuffer.Bytes(), atomic.AddUint64(&cons.sequence, 1))
		resp.WriteHeader(http.StatusCreated)
	} else {
		// Read only the message body
		if req.Body == nil {
			resp.WriteHeader(http.StatusBadRequest)
			return // ### return, missing body ###
		}

		body := make([]byte, req.ContentLength)
		length, err := req.Body.Read(body)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			return // ### return, missing body or bad write ###
		}

		cons.Enqueue(body[:length], atomic.AddUint64(&cons.sequence, 1))
		resp.WriteHeader(http.StatusCreated)
	}
}

func (cons *Http) serve() {
	defer cons.WorkerDone()

	srv := http.Server{
		Addr:        cons.address,
		Handler:     http.HandlerFunc(cons.requestHandler),
		ReadTimeout: cons.readTimeoutSec,
	}

	err := srv.Serve(cons.listen)
	if _, isStopRequest := err.(shared.StopRequestError); err != nil && !isStopRequest {
		Log.Error.Print("httpd: ", err)
	}
}

// Consume opens a new http server listen on specified ip and port (address)
func (cons Http) Consume(workers *sync.WaitGroup) {
	listen, err := shared.NewStopListener(cons.address)
	if err != nil {
		Log.Error.Print("Http: ", err)
		return // ### return, could not connect ###
	}

	cons.listen = listen
	cons.AddMainWorker(workers)

	go cons.serve()
	defer cons.listen.Close()

	cons.ControlLoop()
}
