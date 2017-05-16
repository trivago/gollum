// Copyright 2015-2017 trivago GmbH
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
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestProcessTSV(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("Directives", []string{"1:remove"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo\tbar\tbaz"), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)

	expect.NoError(err)
	expect.Equal("foo\tbaz", msg.String())
}

func TestProcessTSVDelimiter(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("Delimiter", ",")
	config.Override("Directives", []string{"1:remove"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo,bar,baz"), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)

	expect.NoError(err)
	expect.Equal("foo,baz", msg.String())
}

func TestProcessTSVQuotedValues(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("QuotedValues", true)
	config.Override("Directives", []string{"1:remove"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo\t\"\tbar\t\"\tbaz"), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)

	expect.NoError(err)
	expect.Equal("foo\tbaz", msg.String())
}

func TestProcessTSVDelimiterAndQuotedValues(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("QuotedValues", true)
	config.Override("Delimiter", ",")
	config.Override("Directives", []string{"1:remove"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(`foo,",bar,",baz`), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)

	expect.NoError(err)
	expect.Equal("foo,baz", msg.String())
}

func TestProcessTSVQuotedValuesComplex(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("QuotedValues", true)
	config.Override("Directives", []string{"0:remove", "1:remove", "2:remove"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo\t\"\tbar\t\"\tbaz"), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal("\"\tbar\t\"", msg.String())

	msg = core.NewMessage(nil, []byte("\"foo\t\"\tbar\tbaz"), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal("bar", msg.String())

	msg = core.NewMessage(nil, []byte("foo\tbar\t\"\tbaz\t\""), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal("bar", msg.String())

	msg = core.NewMessage(nil, []byte("\"\tfoo\t\"\t\"bar\"\tbip\tbap\tbop\t\"\tbaz\t\""), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal("\"bar\"\tbap\t\"\tbaz\t\"", msg.String())
}

func TestProcessTSVDelimiterAndQuotedValuesComplex(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("QuotedValues", true)
	config.Override("Delimiter", ",")
	config.Override("Directives", []string{"0:remove", "1:remove", "2:remove"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(`foo,",bar,",baz`), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal(`",bar,"`, msg.String())

	msg = core.NewMessage(nil, []byte(`"foo,",bar,baz`), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal("bar", msg.String())

	msg = core.NewMessage(nil, []byte(`foo,bar,",baz,"`), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal("bar", msg.String())

	msg = core.NewMessage(nil, []byte(`",foo,","bar",bip,bap,bop,",baz,"`), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal(`"bar",bap,",baz,"`, msg.String())
}

func TestProcessTSVDirectives(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("Directives", []string{
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

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(
		nil,
		[]byte(
			"20160819171053\t[yml]\tremoveme\tMozilla/5.0 (Windows NT 10.0; Win64; x64; rv:48.0) Gecko/20100101 Firefox/48.0",
		),
		0,
		core.InvalidStreamID,
	)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	// TODO: "rv:48.0" is currently not parsed correctly by the client library (bug)
	expect.Equal(
		"2016-08-19 17:10:53\t\"preyamlpost\"\tFirefox\tWindows 10\t48.0\t5.0\tWindows\tWindows 10\t\tGecko\t20100101\tFirefox\t48.0",
		msg.String(),
	)
}

func TestProcessTSVDelimiterAndDirectives(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("Delimiter", ",")
	config.Override("Directives", []string{
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

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(
		nil,
		[]byte(
			"20160819171053,[yml],removeme,Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:48.0) Gecko/20100101 Firefox/48.0",
		),
		0,
		core.InvalidStreamID,
	)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(
		`2016-08-19 17:10:53,"preyamlpost",Firefox,Windows 10,48.0,5.0,Windows,Windows 10,,Gecko,20100101,Firefox,48.0`,
		msg.String(),
	)
}

func TestProcessTSVQuotedValuesAndDirectives(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("QuotedValues", true)
	config.Override("Directives", []string{"1:remove"})
	config.Override("Directives", []string{
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

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(
		nil,
		[]byte(
			"20160819171053\t\"[\tyml\t]\"\tremoveme\tMozilla/5.0 (Windows NT 10.0; Win64; x64; rv:48.0) Gecko/20100101 Firefox/48.0",
		),
		0,
		core.InvalidStreamID,
	)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(
		"\"2016-08-19 17:10:53\"\t\"pre\tyaml\tpost\"\tFirefox\tWindows 10\t48.0\t5.0\tWindows\tWindows 10\t\tGecko\t20100101\tFirefox\t48.0",
		msg.String(),
	)
}

func TestProcessTSVDelimiterAndQuotedValuesAndDirectives(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("QuotedValues", true)
	config.Override("Delimiter", " ")
	config.Override("Directives", []string{
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

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(
		nil,
		[]byte(
			`20160819171053 [yml] removeme "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:48.0) Gecko/20100101 Firefox/48.0"`,
		),
		0,
		core.InvalidStreamID,
	)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(
		`"2016-08-19 17:10:53" "pre yaml post" Firefox "Windows 10" 48.0 5.0 Windows "Windows 10"  Gecko 20100101 Firefox 48.0`,
		msg.String(),
	)
}

func TestProcessTSVApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessTSV")
	config.Override("Directives", []string{"1:remove"})
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ProcessTSV)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("PAYLOAD"), 0, core.InvalidStreamID)
	msg.MetaData().SetValue("foo", []byte("foo\tbar\tbaz"))
	err = formatter.ApplyFormatter(msg)

	expect.NoError(err)
	expect.Equal("PAYLOAD", msg.String())
	expect.Equal("foo\tbaz", msg.MetaData().GetValueString("foo"))
}