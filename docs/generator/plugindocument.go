//
package main

import (
	"fmt"
	"regexp"
	"strings"
)

// PluginDocument represents the inline documentation from a Gollum plugin's source
type PluginDocument struct {
	PackageName   string                       // Name of Go package
	PluginName    string                       // Name of Go type
	Description   string                       // Description paragraph(s)
	Example       string                       // Config example paragraph
	Parameters    []PluginParameter            // This plugin's own config parameters
	ParameterSets map[string][]PluginParameter // Inherited config parameters
}

// PluginParameter represents a single configuration parameter in a Gollum plugin
type PluginParameter struct {
	name string
	desc string
}

// Parser state
type parserState uint8

const (
	parserStateTitle parserState = iota
	parserStateDescription
	parserStateExample
	parserStateParameterBegin
	parserStateParameterCont
)

// Magics
const (
	parserStanzaExampleRe string = "^configuration example(\\:|)$"
)

// NewPluginDocument creates a new PluginDocument object for the named plugin
func NewPluginDocument(packageName string, pluginName string) PluginDocument {
	pluginDocument := PluginDocument{
		PackageName:   packageName,
		PluginName:    pluginName,
		ParameterSets: make(map[string][]PluginParameter),
	}
	return pluginDocument
}

// ParseString parses and imports a string into this PluginDocument. The string
// should be the text of the comment block preceding the the plugin's
// `type FooBar struct { ... }` declaration, without the preceding `// `s. The
// comment block is assumed to have the following syntax:
//
// Structure of comment block:
//
// // <TITLE (ignored)>
// //
// // <DESCRIPTION...>
// // <DESCRIPTION...>
// // * <DESC_BULLET1...>
// //   <DESC_BULLET1...>
// // * <DESC_BULLET2...>
// //
// // Configuration example
// // <EXAMPLE...>
// // <EXAMPLE...>
// // <EXAMPLE...>
// //
// // <PARAM1_NAME> <PARAM1_TEXT....>t
// // <PARAM1_TEXT ....>
// // <PARAM1_TEXT ....>
// // * <PARAM1_BULLET1 ....>
// //   <PARAM1_BULLET1 ....>
// // * <PARAM1_BULLET2 ....>
// //
// // <PARAM2_NAME> <PARAM2_TEXT....>
// // <PARAM2_TEXT ....>
// // <PARAM2_TEXT ....>
//
func (doc *PluginDocument) ParseString(comment string) {

	lines := strings.Split(comment, "\n")

	state := parserStateTitle
	for _, line := range lines {
		// Trim whitespace
		trimmedLine := strings.Trim(line, " \t")

		// Assign rows in their places
		switch state {
		case parserStateTitle:
			// first line, ignored
			state = parserStateDescription

		case parserStateDescription:
			matched, err := regexp.MatchString(parserStanzaExampleRe, strings.ToLower(trimmedLine))
			if err != nil {
				panic(err)
			}
			if matched {
				// magic string => start example section
				state = parserStateExample
				continue
			}
			doc.Description += line + "\n"

		case parserStateExample:
			if trimmedLine == "" {
				if doc.Example == "" {
					// Allow empty line between "Configuration example" and the example
					continue
				}
				// \n => start parameters section
				state = parserStateParameterBegin
				continue
			}
			doc.Example += line + "\n"

		case parserStateParameterBegin:
			if trimmedLine == "" {
				// Avoid creating an empty parameter when comment has trailing lines
				continue
			}
			tmp := strings.SplitN(trimmedLine, " ", 2)
			doc.Parameters = append(doc.Parameters, PluginParameter{
				name: tmp[0],
				desc: tmp[1] + "\n",
			})
			state = parserStateParameterCont

		case parserStateParameterCont:
			if trimmedLine == "" {
				// \n => start next parameter
				state = parserStateParameterBegin
				continue
			}
			doc.Parameters[len(doc.Parameters)-1].desc += line + "\n"

		default:
			panic(fmt.Sprintf("Unknown state %d\n", state))
		}
	}

}

// IncludeParameters imports the .Parameters property of `document` into this document's
// inherited param list at .ParameterSets[<document.package>.<document.name>]
func (doc *PluginDocument) IncludeParameters(document PluginDocument) {
	doc.ParameterSets[document.PackageName+"."+document.PluginName] = document.Parameters
	for name, paramSet := range document.ParameterSets {
		doc.ParameterSets[name] = paramSet
	}
}

// This function prepends and appends "\n" to all "*" bullet list items.
//
// RST requires preceding and following "\n"s before bullet list items, but
// Gollum's plugindoc format relies on "\n" to separate sections only, so this
// transforms the latter to former.
func docBulletsToRstBullets(text string) string {
	result := ""
	inBullet := false
	for _, line := range strings.Split(text, "\n") {
		if len(line) == 0 {
			result += "\n"
			continue
		}
		chr := line[0]
		if inBullet {
			if chr == " "[0] {
				result += line + "\n"
				continue
			}
			result += "\n" + line + "\n"
			inBullet = false

		} else {
			if chr == "*"[0] {
				result += "\n" + line + "\n"
				inBullet = true
				continue
			}
			result += line + "\n"
		}
	}
	return result
}

// GetRST returns an RST representation of this PluginDocument.
func (doc PluginDocument) GetRST() string {
	result := ""

	// Comment
	result += ".. Autogenerated by Gollum RST generator (docs/generator/*.go)\n\n"

	// Heading
	result += doc.PluginName + "\n"

	result += strings.Repeat("=", len(doc.PluginName)) + "\n"

	//
	result += "\n"
	result += docBulletsToRstBullets(doc.Description) + "\n"
	result += "\n"

	//
	if len(doc.Parameters) > 0 {
		result += "Parameters\n----------\n\n"
		for _, p := range doc.Parameters {
			result += "**" + p.name + "**\n"
			result += docBulletsToRstBullets(p.desc) + "\n"
		}
	}

	//
	for paramSetName, paramSet := range doc.ParameterSets {
		if len(paramSet) == 0 {
			// Skip title for empty param sets
			continue
		}
		head := "Parameters (from " + strings.TrimPrefix(paramSetName, "core.") + ")"
		result += head + "\n"
		result += strings.Repeat("-", len(head)) + "\n\n"

		for _, p := range paramSet {
			result += "**" + p.name + "**\n"
			result += p.desc + "\n"
			result += "\n"
		}
	}

	//
	if len(doc.Example) > 0 {
		result += "Example\n-------\n\n"
		result += ".. code-block:: yaml\n\n"
		for _, line := range strings.Split(doc.Example, "\n") {
			result += "\t" + line + "\n"
		}
	}

	return result
}
