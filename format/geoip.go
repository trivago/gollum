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

package format

import (
	"net"
	"strings"

	"github.com/mmcloughlin/geohash"
	"github.com/trivago/gollum/core"
	geoip2 "gopkg.in/oschwald/geoip2-golang.v1"
)

// GeoIP formatter
//
// This formatter parses an IP and outputs it's geo information as metadata
// fields to the set target.
//
// Parameters
//
// - GeoIPFile: Defines a GeoIP file to load this setting is mandatory. Files
// can be found e.g. at http://dev.maxmind.com/geoip/geoip2/geolite2/.
// By default this parameter is set to "".
//
// - Fields: An array of the fields to extract from the GeoIP.
// Available fields are: "city", "country-code", "country", "continent-code",
// "continent", "timezone", "proxy", "satellite", "location", "location-hash"
// By default this is set to ["city","country","continent","location-hash"].
//
// - Prefix: Defines a prefix for each of the keys generated.
// By default this is set to "".
//
// Examples
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: stdin
//    Modulators:
//      - format.GeoIP
//        Source: client-ip
type GeoIP struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	prefix               string `config:"Prefix"`
	fields               []string
	db                   *geoip2.Reader
}

func init() {
	core.TypeRegistry.Register(GeoIP{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *GeoIP) Configure(conf core.PluginConfigReader) {
	format.fields = conf.GetStringArray("Fields", []string{"city", "country", "continent", "location-hash"})

	for i, field := range format.fields {
		format.fields[i] = strings.ToLower(field)
	}

	geoIPFile := conf.GetString("GeoIPFile", "")
	if geoIPFile == "" {
		format.Logger.Error("setting a GeoIPFile is mandatory")
		return

	}

	var err error
	format.db, err = geoip2.Open(geoIPFile)
	conf.Errors.Push(err)
}

// ApplyFormatter update message payload
func (format *GeoIP) ApplyFormatter(msg *core.Message) error {
	if format.db == nil {
		format.Logger.Warn("no GeoIPFile loaded")
		return nil
	}

	metadata := format.ForceTargetAsMetadata(msg)
	ipString := format.GetSourceDataAsString(msg)

	ip := net.ParseIP(ipString)
	record, err := format.db.City(ip)
	if err != nil {
		format.Logger.WithError(err).Warningf("ip \"%s\" could not be resolved", ipString)
		return err
	}

	for _, field := range format.fields {
		key := format.prefix + field
		switch field {
		case "city":
			if name, exists := record.City.Names["en"]; exists {
				metadata.Set(key, name)
			}

		case "country-code":
			metadata.Set(key, record.Country.IsoCode)

		case "country":
			if name, exists := record.Country.Names["en"]; exists {
				metadata.Set(key, name)
			}

		case "continent-code":
			metadata.Set(key, record.Continent.Code)

		case "continent":
			if name, exists := record.Continent.Names["en"]; exists {
				metadata.Set(key, name)
			}

		case "timezone":
			metadata.Set(key, record.Location.TimeZone)

		case "proxy":
			metadata.Set(key, record.Traits.IsAnonymousProxy)

		case "satellite":
			metadata.Set(key, record.Traits.IsSatelliteProvider)

		case "location":
			metadata.Set(key, []float64{record.Location.Latitude, record.Location.Longitude})

		case "location-hash":
			metadata.Set(key, geohash.Encode(record.Location.Latitude, record.Location.Longitude))
		}
	}
	return nil
}
