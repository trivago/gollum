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
	"bytes"
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

const (
	influxDB09Footer = "]}"
)

// influxDBWriter09 implements the io.Writer interface for InfluxDB 0.9 connections
type influxDBWriter09 struct {
	client           http.Client
	writeURL         string
	queryURL         string
	pingURL          string
	messageHeader    string
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
func (writer *influxDBWriter09) configure(conf core.PluginConfigReader, prod *InfluxDB) error {
	writer.host = conf.GetString("Host", "localhost:8086")
	writer.username = conf.GetString("User", "")
	writer.password = conf.GetString("Password", "")
	writer.databaseTemplate = conf.GetString("Database", "default")
	writer.buffer = tio.NewByteStream(4096)
	writer.connectionUp = false
	writer.Control = prod.Control
	writer.logger = prod.Logger

	writer.writeURL = fmt.Sprintf("http://%s/write", writer.host)
	writer.queryURL = fmt.Sprintf("http://%s/query", writer.host)
	writer.pingURL = fmt.Sprintf("http://%s/ping", writer.host)
	writer.separator = '?'
	writer.timeBasedDBName = conf.GetBool("TimeBasedName", true)

	if writer.username != "" {
		credentials := fmt.Sprintf("?u=%s&p=%s", url.QueryEscape(writer.username), url.QueryEscape(writer.password))
		writer.writeURL += credentials
		writer.queryURL += credentials
		writer.separator = '&'
	}

	if retentionPolicy := conf.GetString("RetentionPolicy", ""); retentionPolicy != "" {
		writer.messageHeader = fmt.Sprintf("{\"database\":\"%%s\",\"retentionPolicy\":\"%s\",\"points\":[", retentionPolicy)
	} else {
		writer.messageHeader = "{\"database\":\"%s\",\"points\":["
	}

	return conf.Errors.OrNil()
}

func (writer *influxDBWriter09) isConnectionUp() bool {
	if writer.connectionUp {
		return true // ### return, connection not reported to be down ###
	}

	if response, err := http.Get(writer.pingURL); err == nil && response != nil {
		defer response.Body.Close()
		switch response.Status[:3] {
		case "200", "204":
			if _, hasInfluxHeader := response.Header["X-Influxdb-Version"]; hasInfluxHeader {
				writer.connectionUp = true
				writer.logger.Info("Connected to " + writer.host)
			}
		}
	}

	return writer.connectionUp
}

func (writer *influxDBWriter09) createDatabase(database string) error {
	url := fmt.Sprintf("%s%cq=%s", writer.queryURL, writer.separator, url.QueryEscape("CREATE DATABASE \""+database+"\""))

	response, err := writer.client.Get(url)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	switch response.Status[:3] {
	case "200", "201":
		writer.logger.Infof("Created database %s", database)
		return nil

	default:
		body, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf("Could not create database %s with status \"%s\" and error \"%s\"", database, response.Status, body)
	}
}

func (writer *influxDBWriter09) post(databaseName string) (int, error) {
	response, err := writer.client.Post(writer.writeURL, "application/json", &writer.buffer)
	if err != nil {
		writer.connectionUp = false
		// TBD: health check? (ex-fuse breaker)
		return 0, err // ### return, failed to connect ###
	}

	defer response.Body.Close()
	switch response.Status[:3] {
	case "200", "204":
		return writer.buffer.Len(), nil // ### return, OK ###

	case "400", "404":
		body, _ := ioutil.ReadAll(response.Body)
		if strings.Contains(string(body), databaseName) {
			err := writer.createDatabase(databaseName)
			if err != nil {
				return 0, err // ### return, failed to create database ###
			}
			writer.buffer.ResetRead()
			return writer.post(databaseName) // ### return, retry ###
		}
		fallthrough

	default:
		body, _ := ioutil.ReadAll(response.Body)
		writer.connectionUp = false
		return 0, fmt.Errorf("%s returned %s: %s", writer.writeURL, response.Status, string(body))
	}
}

func (writer *influxDBWriter09) Write(data []byte) (int, error) {
	writer.buffer.Reset()

	database := writer.databaseTemplate
	if writer.timeBasedDBName {
		database = time.Now().Format(database)
	}

	fmt.Fprintf(&writer.buffer, writer.messageHeader, database)

	data = bytes.TrimRight(data, " \t,") // remove the last comma
	writer.buffer.Write(data)
	writer.buffer.WriteString(influxDB09Footer)

	return writer.post(database)
}
