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

package logger

import (
	"github.com/x-cray/logrus-prefixed-formatter"
)

// NewConsoleFormatter returns a a ConsoleFormatter reference
func NewConsoleFormatter() *prefixed.TextFormatter {
	f := prefixed.TextFormatter{}

	f.ForceColors = true
	f.FullTimestamp = true
	f.ForceFormatting = true
	f.TimestampFormat = "2006-01-02 15:04:05 MST"

	f.SetColorScheme(&prefixed.ColorScheme{
		PrefixStyle:     "blue+h",
		InfoLevelStyle:  "white+h",
		DebugLevelStyle: "cyan",
	})

	return &f
}
