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

package consumer

import (
	"bytes"
	"crypto/tls"
	"github.com/abbot/go-http-auth"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tnet"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

// HTTP consumer plugin
//
// This consumer opens up an HTTP 1.1 server and processes the contents of any
// incoming HTTP request.
//
// Configuration example
//
//  - "consumer.HTTP":
//    Address: ":80"
//    ReadTimeoutSec: 3
//    WithHeaders: true
//    Htpasswd: ""
//    BasicRealm: ""
//    Certificate: ""
//    PrivateKey: ""
//
// Address stores the host and port to bind to.
// This is allowed be any ip address/dns and port like "localhost:5880".
// By default this is set to ":80".
//
// ReadTimeoutSec specifies the maximum duration in seconds before timing out
// the HTTP read request. By default this is set to 3 seconds.
//
// WithHeaders can be set to false to only read the HTTP body instead of passing
// the whole HTTP message. By default this setting is set to true.
//
// Htpasswd can be set to the htpasswd formatted file to enable HTTP BasicAuth
//
// BasicRealm can be set for HTTP BasicAuth
//
// Certificate defines a path to a root certificate file to make this consumer
// handle HTTPS connections. Left empty by default (disabled).
// If a Certificate is given, a PrivateKey must be given, too.
//
// PrivateKey defines a path to the private key used for HTTPS connections.
// Left empty by default (disabled).
// If a Certificate is given, a PrivatKey must be given, too.
type HTTP struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	address             string        `config:"Address" default:":80"`
	readTimeoutSec      time.Duration `config:"ReadTimeoutSec" default:"3" metric:"sec"`
	withHeaders         bool          `config:"WithHeaders" default:"true"`
	htpasswd            string        `config:"Htpasswd"`
	basicRealm          string        `config:"BasicRealm"`
	secrets             auth.SecretProvider
	listen              *tnet.StopListener
	certificate         *tls.Config
}

func init() {
	core.TypeRegistry.Register(HTTP{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *HTTP) Configure(conf core.PluginConfigReader) {
	if cons.htpasswd != "" {
		if _, fileErr := os.Stat(cons.htpasswd); os.IsNotExist(fileErr) {
			conf.Errors.Pushf("htpasswd file does not exist: %s", cons.htpasswd)
			cons.htpasswd = ""
		}
		cons.secrets = auth.HtpasswdFileProvider(cons.htpasswd)
	}

	certificateFile := conf.GetString("Certificate", "")
	keyFile := conf.GetString("PrivateKey", "")

	if certificateFile != "" || keyFile != "" {
		if certificateFile == "" || keyFile == "" {
			conf.Errors.Pushf("There must always be a certificate and a private key or none of both")
		} else {

			cons.certificate = new(tls.Config)
			cons.certificate.NextProtos = []string{"http/1.1"}

			keypair, err := tls.LoadX509KeyPair(certificateFile, keyFile)
			if !conf.Errors.Push(err) {
				cons.certificate.Certificates = []tls.Certificate{keypair}
			}
		}
	}
}

func (cons *HTTP) checkAuth(r *http.Request) bool {
	a := &auth.BasicAuth{Realm: cons.basicRealm, Secrets: cons.secrets}
	if a.CheckAuth(r) == "" {
		return false
	}
	return true
}

// requestHandler will handle a single web request.
func (cons *HTTP) requestHandler(resp http.ResponseWriter, req *http.Request) {
	if cons.htpasswd != "" {
		if !cons.checkAuth(req) {
			resp.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	if cons.withHeaders {
		// Read the whole package
		requestBuffer := bytes.NewBuffer(nil)
		if err := req.Write(requestBuffer); err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			cons.Logger.Error(err)
			return // ### return, missing body or bad write ###
		}

		cons.Enqueue(requestBuffer.Bytes())
		resp.WriteHeader(http.StatusOK)
	} else {
		// Read only the message body
		if req.Body == nil {
			resp.WriteHeader(http.StatusBadRequest)
			return // ### return, missing body ###
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			cons.Logger.Error(err)
			return // ### return, missing body or bad write ###
		}
		defer req.Body.Close()

		cons.Enqueue(body)
		resp.WriteHeader(http.StatusOK)
	}
}

func (cons *HTTP) serve() {
	defer cons.WorkerDone()

	srv := http.Server{
		Addr:        cons.address,
		Handler:     http.HandlerFunc(cons.requestHandler),
		ReadTimeout: cons.readTimeoutSec,
		TLSConfig:   cons.certificate,
	}

	err := srv.Serve(cons.listen)
	if _, isStopRequest := err.(tnet.StopRequestError); err != nil && !isStopRequest {
		cons.Logger.Error(err)
	}
}

// Consume opens a new http server listen on specified ip and port (address)
func (cons HTTP) Consume(workers *sync.WaitGroup) {
	listen, err := tnet.NewStopListener(cons.address)
	if err != nil {
		cons.Logger.Error(err)
		return // ### return, could not connect ###
	}

	cons.listen = listen
	cons.AddMainWorker(workers)

	go cons.serve()
	defer cons.listen.Close()

	cons.ControlLoop()
}
