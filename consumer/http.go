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

package consumer

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/abbot/go-http-auth"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tnet"
)

// HTTP consumer plugin
//
// This consumer opens up an HTTP 1.1 server and processes the contents of any
// incoming HTTP request.
//
// Parameters
//
// - Address: Defines the TCP port and optional IP address to listen on.
// Sets http.Server.Addr; for defails, see its Go documentation.
//
// Syntax: [hostname|address]:<port>
//
// - ReadTimeoutSec: Defines the maximum duration in seconds before timing out
// the HTTP read request. Sets http.Server.ReadTimeout; for details, see its
// Go documentation.
//
// - WithHeaders: If true, relays the complete HTTP request to the generated
// Gollum message. If false, relays only the HTTP request body and ignores
// headers.
//
// - Htpasswd: Path to an htpasswd-formatted password file. If defined, turns
// on HTTP Basic Authentication in the server.
//
// - BasicRealm: Defines the Authentication Realm for HTTP Basic Authentication.
// Meaningful only in conjunction with Htpasswd.
//
// - Certificate: Path to an X509 formatted certificate file. If defined, turns on
// SSL/TLS  support in the HTTP server. Requires PrivateKey to be set.
//
// - PrivateKey: Path to an X509 formatted private key file. Meaningful only in
// conjunction with Certificate.
//
// Examples
//
// This example listens on port 9090 and writes to the stream "http_in_00".
//
//   "HttpIn00":
//     Type: "consumer.HTTP"
//     Streams: "http_in_00"
//     Address: "localhost:9090"
//     WithHeaders: false
//
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
	return a.CheckAuth(r) != ""
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
