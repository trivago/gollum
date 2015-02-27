// Copyright 2015 trivago GmbH
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

package shared

import (
	"fmt"
	"strings"
	"testing"
)

type bufferedReaderTestData struct {
	expect Expect
	tokens []string
	parsed int
}

func (br *bufferedReaderTestData) write(data []byte, seq uint64) {
	br.expect.StringEq(br.tokens[seq], string(data))
	br.parsed++
}

func TestBufferedReaderSimple(t *testing.T) {
	data := bufferedReaderTestData{
		expect: NewExpect(t),
		tokens: []string{"test1", "test 2", "test\r3"},
		parsed: 0,
	}

	parseData := strings.Join(data.tokens, "\n")
	parseReader := strings.NewReader(parseData)
	reader := NewBufferedReader(1024, 0, "\n", data.write)

	reader.Read(parseReader)
	data.expect.IntEq(2, data.parsed)
}

func TestBufferedReaderRLE(t *testing.T) {
	data := bufferedReaderTestData{
		expect: NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	parseData := ""
	for _, s := range data.tokens {
		parseData += fmt.Sprintf("%d:%s", len(s), s)
	}

	parseReader := strings.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagRLE, "", data.write)

	reader.Read(parseReader)
	data.expect.IntEq(3, data.parsed)
}

func TestBufferedReaderSeq(t *testing.T) {
	data := bufferedReaderTestData{
		expect: NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	parseData := ""
	for i, s := range data.tokens {
		parseData += fmt.Sprintf("%d:%s\n", i, s)
	}

	parseReader := strings.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagSequence, "\n", data.write)

	reader.Read(parseReader)
	data.expect.IntEq(3, data.parsed)
}

func TestBufferedReaderRLESeq(t *testing.T) {
	data := bufferedReaderTestData{
		expect: NewExpect(t),
		tokens: []string{"test1", "test 2", "test\t3"},
		parsed: 0,
	}

	parseData := ""
	for i, s := range data.tokens {
		parseData += fmt.Sprintf("%d:%d:%s", len(s)+2, i, s)
	}

	parseReader := strings.NewReader(parseData)
	reader := NewBufferedReader(1024, BufferedReaderFlagRLE|BufferedReaderFlagSequence, "", data.write)

	reader.Read(parseReader)
	data.expect.IntEq(3, data.parsed)
}
