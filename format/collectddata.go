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
	"bytes"
	"encoding/json"
)

type collectdPacket struct {
	Values         []float64
	Dstypes        []string
	Dsnames        []string
	Time           float64
	Interval       float32
	Host           string
	Plugin         string
	PluginInstance string `json:"plugin_instance"`
	PluginType     string `json:"type"`
	TypeInstance   string `json:"type_instance"`
}

func parseCollectdPacket(data []byte) (collectdPacket, error) {
	collectdData := collectdPacket{}
	err := json.Unmarshal(bytes.Trim(data, " \t,"), &collectdData)
	return collectdData, err
}
