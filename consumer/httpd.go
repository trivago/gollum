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
//     ReadTimeout: 10
//
// Address stores the identifier to bind to.
// This must be a ip address and port like "127.0.0.1:80", ":80", "0.0.0.0:80", whatever
// By default this is set to ":80".
//
// ReadTimeout specifies the maximum duration before timing out read of the request.
// The number is entered in seconds.
// By default this is set to 10 seconds.
type Httpd struct {
	core.ConsumerBase
	address     string
	readTimeout int
	sequence    uint64
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
	cons.readTimeout = conf.GetInt("ReadTimeout", 10)

	return err
}

// requestHandler will handle a single web request.
func (cons *Httpd) requestHandler(w http.ResponseWriter, r *http.Request) {
	// We need a content length and a body.
	// Without this information we won`t get out of the warm and cozy bed.
	if r.ContentLength <= 0 {
		w.WriteHeader(http.StatusLengthRequired)
		return
	}

	buffer := bytes.NewBuffer(make([]byte, 0, int(r.ContentLength)))
	n, err := buffer.ReadFrom(r.Body)
	if err != nil || n <= 0 {
		// Uuuuh, we can`t parse the request body. Thats bad.
		// This is not our fault, right?
		// The user must be the mistake, right master?
		// So we are bad to the user.
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cons.PostData(buffer.Bytes(), atomic.AddUint64(&cons.sequence, 1))

	// Send message that everything is fine :)
	w.WriteHeader(http.StatusCreated)
}

// Consume opens a new http server listen on specified ip and port (address)
func (cons Httpd) Consume(workers *sync.WaitGroup) {
	s := &http.Server{
		Addr:        cons.address,
		Handler:     http.HandlerFunc(cons.requestHandler),
		ReadTimeout: time.Duration(cons.readTimeout) * time.Second,
	}

	// Sometimes we can`t bind on a specified port
	// e.g. another process is using it or it is not allowed by OS
	// In this case an error will be thrown
	err := s.ListenAndServe()
	if err != nil {
		Log.Error.Print("HTTPd error: ", err)
		return
	}

	cons.DefaultControlLoop(nil)
}
