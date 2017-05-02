// Copyright 2015-2017 trivago GmbH
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
	"github.com/trivago/tgo/tnet"
	"net/http"
	"sync"
	"github.com/trivago/tgo/thealthcheck"
	"io/ioutil"
	"errors"
)

// HTTPRequest producer plugin
// The HTTPRequest producers sends messages as HTTP packet to a given webserver.
// This producer uses a fuse breaker when a request fails with an error
// code > 400 or the connection is down.
// Configuration example
//
//  - "producer.HTTPRequest":
//    RawData: true
//    Encoding: "text/plain; charset=utf-8"
//    Address: "localhost:80"
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
	core.BufferedProducer
	host       string
	port       string
	protocol   string
	address    string
	encoding   string
	rawPackets bool
	listen     *tnet.StopListener
	lastError  error
}

func init() {
	core.TypeRegistry.Register(HTTPRequest{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *HTTPRequest) Configure(conf core.PluginConfigReader) error {
	var err error
	prod.BufferedProducer.Configure(conf)
	prod.SetStopCallback(prod.close)
	prod.SetCheckFuseCallback(prod.isHostUp)

	address := conf.GetString("Address", "localhost:80")
	prod.protocol, prod.host, prod.port, err = tnet.SplitAddress(address, "http")
	conf.Errors.Push(err)

	if prod.host == "" {
		prod.host = "localhost"
	}

	prod.address = fmt.Sprintf("%s://%s:%s", prod.protocol, prod.host, prod.port)
	prod.encoding = conf.GetString("Encoding", "text/plain; charset=utf-8")
	prod.rawPackets = conf.GetBool("RawData", true)

	//prod.Log.Debug.Printf(
	//"protocol=%s, host=%s, port=%s, address=%s, encoding=%s, rawPackets=%d",
	//	prod.protocol, prod.host, prod.port, prod.address, prod.encoding, prod.rawPackets)

	// Health check to ping the backend with an HTTP GET
	thealthcheck.AddEndpointPathArray(
		[]string{conf.GetTypename(), prod.GetID(), "pingBackend"},
		func()(int, string){
			return prod.healthcheckPingBackend()
		},
	)

	// Health check to check the last result
	// TBD: This may be meaningless in a high-traffic environment; a statistics
	// based check could make more sense.
	thealthcheck.AddEndpointPathArray(
		[]string{conf.GetTypename(), prod.GetID(), "lastError"},
		func()(int, string){
			if prod.lastError == nil {
				return thealthcheck.StatusOK, "OK"
			}
			return thealthcheck.StatusServiceUnavailable, fmt.Sprintf("ERROR: %s", prod.lastError)
		},
	)

	return conf.Errors.OrNil()
}

func (prod *HTTPRequest) healthcheckPingBackend() (int, string) {
	code, body, err := httpRequestWrapper(http.Get(prod.address))
	if err != nil {
		return code, fmt.Sprintf("%s", err)
	}
	return code, body
}

// Wrapper around the (*http.Response, error) values returned by HTTP clients.
//
// Reads the response body and code, returns (code int, body string err error).
// If the query succeeded with HTTP 200, err == nil
// If the query failed in some way, err contains a description of the error,
// code and body are populated whenever possible.
func httpRequestWrapper(resp *http.Response, err error) (int, string, error) {
	if err != nil {
		// Fail
		return thealthcheck.StatusServiceUnavailable, "", err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// Fail
		return resp.StatusCode, "", err
	}

	respBodyString := fmt.Sprintf("%s", respBody)

	err = nil
	if resp.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("%d %s", resp.StatusCode, respBodyString))
	}
	return resp.StatusCode, respBodyString, err
}

func (prod *HTTPRequest) isHostUp() bool {
	resp, err := http.Get(prod.address)
	return err != nil && resp != nil && resp.StatusCode < 400
}

func (prod *HTTPRequest) sendReq(msg *core.Message) {
	var (
		req *http.Request
		err error
	)

	originalMsg := msg.Clone()
	requestData := bytes.NewBuffer(msg.Data())

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
		req, err = http.NewRequest("POST", prod.address, requestData)
		if req != nil {
			req.Header.Add("content-type", prod.encoding)
		}
	}

	if err != nil {
		prod.Log.Error.Print("Invalid request", err)
		prod.Drop(originalMsg)
		prod.lastError = err
		return // ### return, malformed request ###
	}

	go func() {
		_, _, err := httpRequestWrapper(http.DefaultClient.Do(req))
		prod.lastError = err
		if err != nil {
			// Fail
			prod.Log.Error.Print("Send failed: ", err)
			if !prod.isHostUp() {
				prod.Control() <- core.PluginControlFuseBurn
			}
			prod.Drop(originalMsg)
			return
		}
		// Success
		prod.Control() <- core.PluginControlFuseActive
	}()
}

func (prod *HTTPRequest) close() {
	defer prod.WorkerDone()
	prod.DefaultClose()
}

// Produce writes to stdout or stderr.
func (prod *HTTPRequest) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.sendReq)
}
