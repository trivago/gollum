//
package main

import (
	"fmt"
	"strings"
)

// PluginDocument represents the inline documentation from a Gollum plugin's source
type PluginDocument struct {
	PackageName         string                    // Name of Go package
	PluginName          string                    // Name of Go type
	BlockHeading        string                    // Contents of the main header
	Description         string                    // Description paragraph(s)
	Parameters          DefinitionList            // This plugin's own config parameters
	InheritedParameters map[string]DefinitionList // Inherited config parameters
	Metadata            DefinitionList            // This plugin's own metadata fields
	InheritedMetadata   map[string]DefinitionList // Inherited metadata fields
	Example             string                    // Config example paragraph
}

// sliceIterator lets us iterate traverse through a bunch of strings conveniently
type sliceIterator struct {
	slice    []string
	position int
}

// next returns the current string and advances the iterator
func (iter *sliceIterator) next() (string, string, int) {
	if iter.position > len(iter.slice)-1 {
		return "", "", -1
	}
	position := iter.position
	iter.position++
	return iter.slice[position], strings.Trim(iter.slice[position], " \t"), position
}

// peek returns the current string without moving the iterator position
func (iter *sliceIterator) peek() (string, string, int) {
	if iter.position > len(iter.slice)-1 {
		return "", "", -1
	}

	return iter.slice[iter.position],
		strings.Trim(iter.slice[iter.position], " \t"),
		iter.position
}

// Parser state
type section uint8

const (
	sectionStart section = iota
	sectionMetadata
	sectionParameters
	sectionConfigurationExample
)

// Magic constants
const (
	headingMetadata             string = "Metadata"
	headingParameters           string = "Parameters"
	headingConfigurationExample string = "Examples"
)

// NewPluginDocument creates a new PluginDocument object for the named plugin
func NewPluginDocument(packageName string, pluginName string) PluginDocument {
	pluginDocument := PluginDocument{
		PackageName: packageName,
		PluginName:  pluginName,
		//ParameterSets: make(map[string][]Definition),
	}
	return pluginDocument
}

// DumpString returns a human-readable string dumpString of this object
func (doc *PluginDocument) DumpString() string {
	str := ""
	str += "==================== START DUMP ============================\n"
	str += "PackageName: [" + doc.PackageName + "]\n"
	str += "PluginName:  [" + doc.PluginName + "]\n"
	str += "BlockHeading: [" + doc.BlockHeading + "]\n"
	str += "Description: [" + doc.Description + "]\n\n"
	str += "Parameters: [" + "\n"
	str += doc.Parameters.dumpString() + "\n"
	str += "]\n\n"
	for parentName, defMap := range doc.InheritedParameters {
		str += "Parameters (from " + parentName + ")[\n"
		str += defMap.dumpString() + "\n"
		str += "]\n\n"
	}
	str += "Metadata: [" + "\n"
	str += doc.Metadata.dumpString() + "\n"
	for parentName, defMap := range doc.InheritedMetadata {
		str += "Metadata (from " + parentName + ")[\n"
		str += defMap.dumpString() + "\n"
		str += "]\n\n"
	}
	str += "Example: [" + "\n------------\n" + doc.Example + "\n----------]\n"
	str += "==================== END DUMP ============================\n"
	return str
}

// ParseString parses and imports a string into this PluginDocument.
//
// The string  should be the text of the comment block preceding the the plugin's
// `type FooBar struct { ... }` declaration, without the preceding `// `s.
func (doc *PluginDocument) ParseString(comment string) {

	lines := sliceIterator{
		slice:    strings.Split(comment, "\n"),
		position: 0,
	}

	// This parses the comment block into 5 strings:
	// doc.BlockHeading, doc.Description, doc.Example,
	// metadataText, parametersText

	_, doc.BlockHeading, _ = lines.next()
	if _, tmp, _ := lines.next(); tmp != "" {
		panic(fmt.Sprintf("Expected empty line after block heading, got \"%s\"", tmp))
	}

	section := sectionStart

	var (
		metadataText   string
		parametersText string
		line           string
		trimmedLine    string
		lineNr         int
	)

	for {
		prevTrimmedLine := trimmedLine
		line, trimmedLine, lineNr = lines.next()

		if lineNr < 0 {
			break
		}

		// Look for section delimiter
		_, nextTrimmedLine, _ := lines.peek()
		if prevTrimmedLine == "" && nextTrimmedLine == "" {
			switch trimmedLine {
			case headingMetadata:
				section = sectionMetadata
				lines.next()
				continue

			case headingParameters:
				section = sectionParameters
				lines.next()
				continue

			case headingConfigurationExample:
				section = sectionConfigurationExample
				lines.next()
				continue
			}

			// Support for arbitrary section headers could be added here
		}

		// Assign lines to their own section
		switch section {
		case sectionStart:
			doc.Description += line + "\n"
			continue

		case sectionMetadata:
			metadataText += line + "\n"
			continue

		case sectionParameters:
			parametersText += line + "\n"
			continue

		case sectionConfigurationExample:
			doc.Example += line + "\n"
			continue
		}
	}

	// Metadata and Parameters sections are assumed to contain (recursive) definition lists
	// Description and Examples are taken as-is
	doc.Metadata = newDefinitionListFromString(metadataText)
	doc.Parameters = newDefinitionListFromString(parametersText)
}

// InheritMetadata imports the .Metadata property of `document` into this document's
// inherited metadata list
func (doc *PluginDocument) InheritMetadata(parentDoc PluginDocument) {
	if doc.InheritedMetadata == nil {
		doc.InheritedMetadata = make(map[string]DefinitionList)
	}

	doc.InheritedMetadata[parentDoc.PackageName+"."+parentDoc.PluginName] =
		parentDoc.Metadata

	for parentName, set := range parentDoc.InheritedMetadata {
		doc.InheritedMetadata[parentName] = set
	}
}

// InheritParameters imports the .Parameters property of `document` into this document's
// inherited param list
func (doc *PluginDocument) InheritParameters(parentDoc PluginDocument) {
	if doc.InheritedParameters == nil {
		doc.InheritedParameters = make(map[string]DefinitionList)
	}

	doc.InheritedParameters[parentDoc.PackageName+"."+parentDoc.PluginName] =
		parentDoc.Parameters

	for parentName, paramSet := range parentDoc.InheritedParameters {
		doc.InheritedParameters[parentName] = paramSet
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

	// Print top comment
	result += ".. Autogenerated by Gollum RST generator (docs/generator/*.go)\n\n"

	// Print heading
	result += doc.PluginName + "\n"
	result += strings.Repeat("=", len(doc.PluginName)) + "\n"

	// Print description
	result += "\n"
	result += docBulletsToRstBullets(doc.Description) + "\n"
	result += "\n"

	// Print native metadata
	if len(doc.Metadata) > 0 {
		result += formatRstHeading("Metadata")
		result += doc.Metadata.getRST(false, 0)
	}

	// Print inherited metadata
	for parentName, definitions := range doc.InheritedMetadata {
		if len(definitions) == 0 {
			// Skip title for empty sets
			continue
		}
		result += formatRstHeading(
			"Metadata (from " + strings.TrimPrefix(parentName, "core.") + ")")
		result += definitions.getRST(false, 0)
	}

	// Print native parameters
	if len(doc.Parameters) > 0 {
		result += formatRstHeading("Parameters")
		result += doc.Parameters.getRST(true, 0)
	}

	// Print inherited parameters
	for parentName, definitions := range doc.InheritedParameters {
		if len(definitions) == 0 {
			// Skip title for empty param sets
			continue
		}
		result += formatRstHeading(
			"Parameters (from " + strings.TrimPrefix(parentName, "core.") + ")")
		result += definitions.getRST(true, 0)
	}

	// Print config example
	if len(doc.Example) > 0 {
		result += formatRstHeading("Examples")
		result += ".. code-block:: yaml\n\n"
		for _, line := range strings.Split(doc.Example, "\n") {
			result += "\t" + line + "\n"
		}
	}

	return result
}

// Returns str as an RST heading
func formatRstHeading(str string) string {
	return str + "\n" + strings.Repeat("-", len(str)) + "\n\n"
}
