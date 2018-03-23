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
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/thealthcheck"
)

// HTTPRequest producer
//
// The HTTPRequest producer sends messages as HTTP requests to a given webserver.
//
// In RawData mode, incoming messages are expected to contain complete
// HTTP requests in "wire format", such as:
// ::
//
//   POST /foo/bar HTTP/1.0\n
//   Content-type: text/plain\n
//   Content-length: 24
//   \n
//   Dummy test\n
//   Request data\n
//
// In this mode, the message's contents is parsed as an HTTP request and
// sent to the destination server (virtually) unchanged. If the message
// cannot be parsed as an HTTP request, an error is logged. Only the scheme,
// host and port components of the "Address" URL are used; any path and query
// parameters are ignored. The "Encoding" parameter is ignored.
//
// If RawData mode is off, a POST request is made to the destination server
// for each incoming message, using the complete URL in "Address". The
// incoming message's contents are delivered in the POST request's body
// and Content-type is set to the value of "Encoding"
//
// Parameters
//
// - Address: defines the URL to send http requests to. If the value doesn't
// contain "://",  it is prepended with "http://", so short forms like
// "localhost:8088" are accepted. The default value is "http://localhost:80".
//
// - RawData: Turns "RawData" mode on. See the description above.
//
// - Encoding: Defines the payload encoding when RawData is set to false.
//
// Examples
//
//  HttpOut01:
//    Type: producer.HTTPRequest
//    Streams: http_01
//    Address: "http://localhost:8099/test"
//    RawData: true
//
type HTTPRequest struct {
	core.BufferedProducer `gollumdoc:"embed_type"`

	destinationURL *url.URL
	encoding       string `config:"Encoding" default:"text/plain; charset=utf-8"`
	rawPackets     bool   `config:"RawData" default:"true"`
	lastError      error
}

func init() {
	core.TypeRegistry.Register(HTTPRequest{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *HTTPRequest) Configure(conf core.PluginConfigReader) {
	var err error
	prod.SetStopCallback(prod.close)

	address := conf.GetString("Address", "http://localhost:80")

	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	prod.destinationURL, err = url.Parse(address)
	conf.Errors.Push(err)

	// Default health check to ping the backend with an HTTP GET
	prod.AddHealthCheck(prod.healthcheckPingBackend)

	// Additional health check to check the last result
	// TBD: This may be meaningless in a high-traffic environment; a statistics
	// based check could make more sense.
	prod.AddHealthCheckAt("/lastError", func() (int, string) {
		if prod.lastError == nil {
			return thealthcheck.StatusOK, "OK"
		}
		return thealthcheck.StatusServiceUnavailable, fmt.Sprintf("ERROR: %s", prod.lastError)
	})
}

func (prod *HTTPRequest) healthcheckPingBackend() (int, string) {
	code, body, err := httpRequestWrapper(http.Get(prod.destinationURL.String()))
	if err != nil {
		return code, strconv.Quote(err.Error())
	}
	return code, strconv.Quote(body)
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
		err = fmt.Errorf("%d %s", resp.StatusCode, respBodyString)
	}
	return resp.StatusCode, respBodyString, err
}

func (prod *HTTPRequest) isHostUp() bool {
	resp, err := http.Get(prod.destinationURL.String())
	return err != nil && resp != nil && resp.StatusCode < 400
}

// The onMessage callback
func (prod *HTTPRequest) sendReq(msg *core.Message) {
	var (
		req *http.Request
		err error
	)

	requestData := bytes.NewBuffer(msg.GetPayload())

	if prod.rawPackets {
		// Assume the message already contains an HTTP request in wire format.
		// Create a Request object, override host, port and scheme, and send it out.
		req, err = http.ReadRequest(bufio.NewReader(requestData))
		if req != nil {
			req.URL.Host = prod.destinationURL.Host
			req.URL.Scheme = prod.destinationURL.Scheme
			req.RequestURI = ""
		}
	} else {
		// Encapsulate the message in a POST request
		req, err = http.NewRequest("POST", prod.destinationURL.String(), requestData)
		if req != nil {
			req.Header.Add("Content-type", prod.encoding)
		}
	}

	if err != nil {
		prod.Logger.Error("Invalid request: ", err)
		prod.TryFallback(msg)
		prod.lastError = err
		return // ### return, malformed request ###
	}

	go func() {
		_, _, err := httpRequestWrapper(http.DefaultClient.Do(req))
		prod.lastError = err
		if err != nil {
			// Fail
			prod.Logger.WithError(err).Error("Send failed")
			if !prod.isHostUp() {
				prod.Logger.Error("Host is down")
			}
			prod.TryFallback(msg)
			return
		}
		// Success
		// TBD: health check? (ex-fuse breaker)
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
