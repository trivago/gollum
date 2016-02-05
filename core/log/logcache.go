// Copyright 2015-2016 trivago GmbH
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

package Log

import (
	"io"
)

type logCache struct {
	messages []string
}

// Write caches the message as string
func (log *logCache) Write(message []byte) (int, error) {
	log.messages = append(log.messages, string(message))
	return len(message), nil
}

func (log *logCache) flush(writer io.Writer) {
	for _, message := range log.messages {
		writer.Write([]byte(message))
	}
	log.messages = []string{}
}
