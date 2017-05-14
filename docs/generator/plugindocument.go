//
package main

import (
	"strings"
	"fmt"
)

const (
	enumChar = '*'
)


// Represents the inline documentation from a Gollum plugins's source
type PluginDocument struct {
	PluginName       string
	PackageName      string
	Comment          string
	Description      string
	Parameters       []PluginParameter
	ParameterSets    map[string][]PluginParameter
	ExampleFirstLine string
	Example          []string
}

type PluginParameter struct {
	name string
	desc string
}

// Creates a new PluginDocument by parsing a string ( == comments without leading "//"s)
func NewPluginDocument(packageName string, pluginName string) PluginDocument {
	pluginDocument := PluginDocument{
		PackageName: packageName,
		PluginName: pluginName,
		ParameterSets: make(map[string][]PluginParameter),
	}
	return pluginDocument
}

func (doc *PluginDocument) ParseString(comment string) error {
	remains, err := getNextLine(comment)
	if err != nil {
		return err
	}

	exampleIdx := strings.Index(strings.ToLower(remains), "configuration example")
	if exampleIdx == -1 {
		doc.Description = strings.Replace(remains, "\n", " ", -1)
		return nil // ### return, simple plugin ###
	}

	doc.Description = strings.Replace(remains[:exampleIdx], "\n", " ", -1)
	remains = remains[exampleIdx:]

	exampleStartIdx := getNextBlockIdx(remains)
	if exampleStartIdx == -1 {
		return fmt.Errorf("Example start not found") // ### return, no example ###
	}
	remains = remains[exampleStartIdx+2:]

	exampleEndIdx := getNextBlockIdx(remains)
	if exampleEndIdx == -1 {
		return fmt.Errorf("Example end not found") // ### return, no example ###
	}

	doc.parseExample(remains[:exampleEndIdx])

	remains = remains[exampleEndIdx+2:]
	for len(remains) > 0 {
		name := getFirstWord(remains)
		descEndIdx := getNextBlockIdx(remains)
		if descEndIdx > 0 {
			p := PluginParameter{
				name: name,
				desc: strings.Replace(remains[:descEndIdx], "\n", " ", -1),
			}
			doc.Parameters = append(doc.Parameters, p)
			remains = remains[descEndIdx+2:]
		} else {
			p := PluginParameter{
				name: name,
				desc: strings.Replace(remains, "\n", " ", -1),
			}
			doc.Parameters = append(doc.Parameters, p)
			break // ### break, last PluginParameter ###
		}
	}

	return nil
}

func (doc *PluginDocument) IncludeParameters(document PluginDocument) {
	doc.ParameterSets[document.PackageName + "." + document.PluginName] = document.Parameters
	for name, paramSet := range document.ParameterSets {
		doc.ParameterSets[name] = paramSet
	}
}

func (doc *PluginDocument) parseExample(comment string) {
	var line string
	remains := comment

	for i := 0; ; i++ {
		eolIdx := strings.Index(remains, "\n")
		if eolIdx == -1 {
			line = normalize(remains)
		} else {
			line = normalize(remains[:eolIdx])
		}

		if i == 0 {
			doc.ExampleFirstLine = line
		} else {
			doc.Example = append(doc.Example, line)
		}

		if eolIdx == -1 {
			return // ### return, done ###
		}
		remains = remains[eolIdx+1:]
	}
}

func getFirstWord(sentence string) string {
	trimmed := strings.Trim(sentence, "\n\t ")
	if firstSpaceIdx := strings.Index(trimmed, " "); firstSpaceIdx > 0 {
		return trimmed[:firstSpaceIdx]
	}
	return trimmed
}

func getNextLine(sentence string) (string, error) {
	eolIdx := strings.IndexRune(sentence, '\n')
	if eolIdx == -1 {
		return "", fmt.Errorf("Could not find next line")
	}
	return sentence[eolIdx+1:], nil
}

func getNextBlockIdx(sentence string) int {
	return strings.Index(sentence, "\n\n")
}

func normalize(sentence string) string {
	indent := 0
	for i := 0; i < len(sentence); i++ {
		switch sentence[i] {
		case '\t':
			indent += 2
		case ' ':
			indent++
		default:
			indent /= 2
			return strings.Repeat("    ", indent) + sentence[i:]
		}
	}

	return sentence
}
