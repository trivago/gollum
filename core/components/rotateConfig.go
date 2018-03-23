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

package components

import (
	"strconv"
	"strings"
	"time"

	"github.com/trivago/gollum/core"
)

// RotateConfig defines rotation settings
//
// Parameters
//
// - Rotation/Enable: If this value is set to "true" the logs will rotate after reaching certain thresholds.
// By default this parameter is set to "false".
//
// - Rotation/TimeoutMin: This value defines a timeout in minutes that will cause the logs to
// rotate. Can be set in parallel with RotateSizeMB.
// By default this parameter is set to "1440".
//
// - Rotation/SizeMB: This value defines the maximum file size in MB that triggers a file rotate.
// Files can get bigger than this size.
// By default this parameter is set to "1024".
//
// - Rotation/Timestamp: This value sets the timestamp added to the filename when file rotation
// is enabled. The format is based on Go's time.Format function.
// By default this parameter is to to "2006-01-02_15".
//
// - Rotation/ZeroPadding: This value sets the number of leading zeros when rotating files with
// an existing name. Setting this setting to 0 won't add zeros, every other
// number defines the number of leading zeros to be used.
// By default this parameter is set to "0".
//
// - Rotation/Compress: This value defines if a rotated logfile is to be gzip compressed or not.
// By default this parameter is set to "false".
//
// - Rotation/At: This value defines a specific time for rotation in hh:mm format.
// By default this parameter is set to "".
//
type RotateConfig struct {
	Timeout   time.Duration `config:"Rotation/TimeoutMin" default:"1440" metric:"min"`
	SizeByte  int64         `config:"Rotation/SizeMB" default:"1024" metric:"mb"`
	Timestamp string        `config:"Rotation/Timestamp" default:"2006-01-02_15"`
	ZeroPad   int           `config:"Rotation/ZeroPadding" default:"0"`
	AtHour    int           `config:"Rotation/AtHour" default:"-1"`
	AtMinute  int           `config:"Rotation/AtMin" default:"-1"`
	Compress  bool          `config:"Rotation/Compress" default:"false"`
	Enabled   bool          `config:"Rotation/Enable" default:"false"`
}

// NewRotateConfig create and returns a RotateConfig with default settings
func NewRotateConfig() RotateConfig {
	return RotateConfig{
		Enabled:   false,
		Timeout:   1440,
		SizeByte:  1024,
		Timestamp: "2006-01-02_15",
		ZeroPad:   0,
		Compress:  false,
		AtHour:    -1,
		AtMinute:  -1,
	}
}

// Configure method for interface implementation
func (rotate *RotateConfig) Configure(conf core.PluginConfigReader) {
	rotateAt := conf.GetString("Rotation/At", "")
	if rotateAt != "" {
		parts := strings.Split(rotateAt, ":")
		rotateAtHour, _ := strconv.ParseInt(parts[0], 10, 8)
		rotateAtMin, _ := strconv.ParseInt(parts[1], 10, 8)

		rotate.AtHour = int(rotateAtHour)
		rotate.AtMinute = int(rotateAtMin)
	}
}
