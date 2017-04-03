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

package format

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestProcessTSV(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVDirectives", []string{"1:remove"})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo\tbar\tbaz"), 0)
	result, _ := formatter.Format(msg)

	expect.Equal("foo\tbaz", string(result))
}

func TestProcessTSVDelimiter(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVDelimiter", ",")
	config.Override("ProcessTSVDirectives", []string{"1:remove"})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo,bar,baz"), 0)
	result, _ := formatter.Format(msg)

	expect.Equal("foo,baz", string(result))
}

func TestProcessTSVQuotedValues(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVQuotedValues", true)
	config.Override("ProcessTSVDirectives", []string{"1:remove"})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo\t\"\tbar\t\"\tbaz"), 0)
	result, _ := formatter.Format(msg)

	expect.Equal("foo\tbaz", string(result))
}

func TestProcessTSVDelimiterAndQuotedValues(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVQuotedValues", true)
	config.Override("ProcessTSVDelimiter", ",")
	config.Override("ProcessTSVDirectives", []string{"1:remove"})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(`foo,",bar,",baz`), 0)
	result, _ := formatter.Format(msg)

	expect.Equal("foo,baz", string(result))
}

func TestProcessTSVQuotedValuesComplex(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVQuotedValues", true)
	config.Override("ProcessTSVDirectives", []string{"0:remove", "1:remove", "2:remove"})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo\t\"\tbar\t\"\tbaz"), 0)
	result, _ := formatter.Format(msg)
	expect.Equal("\"\tbar\t\"", string(result))

	msg = core.NewMessage(nil, []byte("\"foo\t\"\tbar\tbaz"), 0)
	result, _ = formatter.Format(msg)
	expect.Equal("bar", string(result))

	msg = core.NewMessage(nil, []byte("foo\tbar\t\"\tbaz\t\""), 0)
	result, _ = formatter.Format(msg)
	expect.Equal("bar", string(result))

	msg = core.NewMessage(nil, []byte("\"\tfoo\t\"\t\"bar\"\tbip\tbap\tbop\t\"\tbaz\t\""), 0)
	result, _ = formatter.Format(msg)
	expect.Equal("\"bar\"\tbap\t\"\tbaz\t\"", string(result))
}

func TestProcessTSVDelimiterAndQuotedValuesComplex(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVQuotedValues", true)
	config.Override("ProcessTSVDelimiter", ",")
	config.Override("ProcessTSVDirectives", []string{"0:remove", "1:remove", "2:remove"})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(`foo,",bar,",baz`), 0)
	result, _ := formatter.Format(msg)
	expect.Equal(`",bar,"`, string(result))

	msg = core.NewMessage(nil, []byte(`"foo,",bar,baz`), 0)
	result, _ = formatter.Format(msg)
	expect.Equal("bar", string(result))

	msg = core.NewMessage(nil, []byte(`foo,bar,",baz,"`), 0)
	result, _ = formatter.Format(msg)
	expect.Equal("bar", string(result))

	msg = core.NewMessage(nil, []byte(`",foo,","bar",bip,bap,bop,",baz,"`), 0)
	result, _ = formatter.Format(msg)
	expect.Equal(`"bar",bap,",baz,"`, string(result))
}

func TestProcessTSVDirectives(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVDirectives", []string{
		"0:time:20060102150405:2006-01-02 15\\:04\\:05",
		"1:replace:yml:yaml",
		"1:trim:[]",
		"1:prefix:pre",
		"1:postfix:post",
		"1:quote",
		"2:remove",
		"2:agent",
		"2:agent:browser:os:version",
		"2:remove",
	})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(
		nil,
		[]byte(
			"20160819171053\t[yml]\tremoveme\tMozilla/5.0 (Windows NT 10.0; Win64; x64; rv:48.0) Gecko/20100101 Firefox/48.0",
		),
		0,
	)
	result, _ := formatter.Format(msg)

	// TODO: "rv:48.0" is currently not parsed correctly by the client library (bug)
	expect.Equal(
		"2016-08-19 17:10:53\t\"preyamlpost\"\tFirefox\tWindows 10\t48.0\t5.0\tWindows\tWindows 10\t\tGecko\t20100101\tFirefox\t48.0",
		string(result),
	)
}

func TestProcessTSVDelimiterAndDirectives(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVDelimiter", ",")
	config.Override("ProcessTSVDirectives", []string{
		"0:time:20060102150405:2006-01-02 15\\:04\\:05",
		"1:replace:yml:yaml",
		"1:trim:[]",
		"1:prefix:pre",
		"1:postfix:post",
		"1:quote",
		"2:remove",
		"2:agent",
		"2:agent:browser:os:version",
		"2:remove",
	})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(
		nil,
		[]byte(
			"20160819171053,[yml],removeme,Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:48.0) Gecko/20100101 Firefox/48.0",
		),
		0,
	)
	result, _ := formatter.Format(msg)

	expect.Equal(
		`2016-08-19 17:10:53,"preyamlpost",Firefox,Windows 10,48.0,5.0,Windows,Windows 10,,Gecko,20100101,Firefox,48.0`,
		string(result),
	)
}

func TestProcessTSVQuotedValuesAndDirectives(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVQuotedValues", true)
	config.Override("ProcessTSVDirectives", []string{"1:remove"})
	config.Override("ProcessTSVDirectives", []string{
		"0:time:20060102150405:2006-01-02 15\\:04\\:05",
		"0:quote",
		"1:replace:yml:yaml",
		"1:trim:[]",
		"1:prefix:pre",
		"1:postfix:post",
		"2:remove",
		"2:agent",
		"2:agent:browser:os:version",
		"2:remove",
	})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(
		nil,
		[]byte(
			"20160819171053\t\"[\tyml\t]\"\tremoveme\tMozilla/5.0 (Windows NT 10.0; Win64; x64; rv:48.0) Gecko/20100101 Firefox/48.0",
		),
		0,
	)
	result, _ := formatter.Format(msg)

	expect.Equal(
		"\"2016-08-19 17:10:53\"\t\"pre\tyaml\tpost\"\tFirefox\tWindows 10\t48.0\t5.0\tWindows\tWindows 10\t\tGecko\t20100101\tFirefox\t48.0",
		string(result),
	)
}

func TestProcessTSVDelimiterAndQuotedValuesAndDirectives(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("ProcessTSVQuotedValues", true)
	config.Override("ProcessTSVDelimiter", " ")
	config.Override("ProcessTSVDirectives", []string{
		`0:time:20060102150405:2006-01-02 15\:04\:05`,
		"0:quote",
		"1:replace:yml:yaml",
		"1:trim:[]",
		"1:prefix:pre ",
		"1:postfix: post",
		"1:quote",
		"2:remove",
		"2:agent",
		"2:agent:browser:os:version",
		"2:remove",
		"3:quote",
		"7:quote",
	})

	plugin, err := core.NewPluginWithType("format.ProcessTSV", config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(
		nil,
		[]byte(
			`20160819171053 [yml] removeme "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:48.0) Gecko/20100101 Firefox/48.0"`,
		),
		0,
	)
	result, _ := formatter.Format(msg)

	expect.Equal(
		`"2016-08-19 17:10:53" "pre yaml post" Firefox "Windows 10" 48.0 5.0 Windows "Windows 10"  Gecko 20100101 Firefox 48.0`,
		string(result),
	)
}
