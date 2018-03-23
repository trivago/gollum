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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tio"
)

// influxDBWriter10 implements the io.Writer interface for InfluxDB 0.9 connections
type influxDBWriter10 struct {
	client           http.Client
	writeURL         string
	queryURL         string
	pingURL          string
	databaseTemplate string
	host             string
	username         string
	password         string
	separator        rune
	connectionUp     bool
	timeBasedDBName  bool
	Control          func() chan<- core.PluginControl
	buffer           tio.ByteStream
	logger           logrus.FieldLogger
}

// Configure sets the database connection values
func (writer *influxDBWriter10) configure(conf core.PluginConfigReader, prod *InfluxDB) error {
	writer.host = conf.GetString("Host", "localhost:8086")
	writer.username = conf.GetString("User", "")
	writer.password = conf.GetString("Password", "")
	writer.databaseTemplate = conf.GetString("Database", "default")
	writer.buffer = tio.NewByteStream(4096)
	writer.connectionUp = false
	writer.timeBasedDBName = conf.GetBool("TimeBasedName", true)
	writer.Control = prod.Control
	writer.logger = prod.Logger

	writer.writeURL = fmt.Sprintf("http://%s/write", writer.host)
	writer.queryURL = fmt.Sprintf("http://%s/query", writer.host)
	writer.pingURL = fmt.Sprintf("http://%s/ping", writer.host)
	writer.separator = '?'

	if writer.username != "" {
		credentials := fmt.Sprintf("?u=%s&p=%s", url.QueryEscape(writer.username), url.QueryEscape(writer.password))
		writer.writeURL += credentials
		writer.queryURL += credentials
		writer.separator = '&'
	}

	writer.writeURL = fmt.Sprintf("%s%cprecision=ms", writer.writeURL, writer.separator)
	return conf.Errors.OrNil()
}

func (writer *influxDBWriter10) isConnectionUp() bool {
	if writer.connectionUp {
		return true // ### return, connection not reported to be down ###
	}

	if response, err := http.Get(writer.pingURL); err == nil && response != nil {
		defer response.Body.Close()
		switch response.Status[:3] {
		case "200", "204":
			if _, hasInfluxHeader := response.Header["X-Influxdb-Version"]; hasInfluxHeader {
				writer.connectionUp = true
				writer.logger.Debug("Connected to " + writer.host)
			}
		}
	}

	return writer.connectionUp
}

func (writer *influxDBWriter10) createDatabase(database string) error {
	url := fmt.Sprintf("%s%cq=%s", writer.queryURL, writer.separator, url.QueryEscape("CREATE DATABASE \""+database+"\""))

	response, err := writer.client.Get(url)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	switch response.Status[:3] {
	case "200", "201":
		writer.logger.Info("Created database ", database)
		return nil

	default:
		body, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf("Could not create database %s with status \"%s\" and error \"%s\"", database, response.Status, body)
	}
}

func (writer *influxDBWriter10) post() (int, error) {
	databaseName := writer.databaseTemplate
	if writer.timeBasedDBName {
		databaseName = time.Now().Format(databaseName)
	}
	writeURL := fmt.Sprintf("%s&db=%s", writer.writeURL, url.QueryEscape(databaseName)) // Allow timestamping the database with the current time

	response, err := writer.client.Post(writeURL, "text/plain; charset=utf-8", &writer.buffer)
	if err != nil {
		writer.connectionUp = false
		// TBD: health check? (ex-fuse breaker)
		return 0, err // ### return, failed to connect ###
	}

	defer response.Body.Close()

	// Check status codes
	switch response.Status[:3] {
	case "200", "204":
		return writer.buffer.Len(), nil // ### return, OK ###

	case "404":
		body, _ := ioutil.ReadAll(response.Body)
		// 404 Not found: Database not found: "foobar"
		if strings.Contains(string(body), databaseName) {
			err := writer.createDatabase(databaseName)
			if err != nil {
				return 0, err // ### return, failed to create database ###
			}

			writer.buffer.ResetRead()
			return writer.post() // ### return, retry ###
		}
		fallthrough

	default:
		body, _ := ioutil.ReadAll(response.Body)
		writer.connectionUp = false
		return 0, fmt.Errorf("%s returned %s: %s", writer.writeURL, response.Status, string(body))
	}
}

func (writer *influxDBWriter10) Write(data []byte) (int, error) {
	writer.buffer.Reset()
	writer.buffer.Write(data)
	return writer.post()
}
