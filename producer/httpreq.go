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
	"net/http"
	"sync"
	"time"
)

// HttpReq producer plugin
// Configuration example
//
//   - "producer.HttpReq":
//     Enable:  true
//     Host:    "staging.trv:8080"
//	   ReadTimeoutSec: 5
//
// ReadTimeoutSec specifies the maximum duration in seconds before timing out
// read of the request. By default this is set to 3 seconds.
type HttpReq struct {
	core.ProducerBase
	host           string
	listen         *shared.StopListener
	readTimeoutSec time.Duration
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

	if !conf.HasValue("Host") {
		return core.NewProducerError("No Host configured for producer.HttpReq")
	}
	prod.host = conf.GetString("Host", "")
	prod.readTimeoutSec = time.Duration(conf.GetInt("ReadTimeoutSec", 3)) * time.Second

	return nil
}

func (prod *HttpReq) sendReq(msg core.Message) {
	go func() {
		b := bytes.NewBuffer(prod.ProducerBase.Format(msg))
		r, err := http.ReadRequest(bufio.NewReader(b))
		if err != nil {
			Log.Error.Print("HttpReq request error:", err)
			return
		}
		r.URL.Host = prod.host
		r.RequestURI = ""
		r.URL.Scheme = "http"
		_, e := http.DefaultClient.Do(r)
		if e != nil {
			Log.Error.Print("HttpReq error sending request: ", e)
		} else {
			// Request Sent
		}
	}()
}

// Produce writes to stdout or stderr.
func (prod HttpReq) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	defer prod.WorkerDone()

	prod.DefaultControlLoop(prod.sendReq, nil)
}
