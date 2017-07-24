//
package main

import (
	"fmt"
	"regexp"
	"strings"
)

// PluginDocument represents the inline documentation from a Gollum plugin's source
type PluginDocument struct {
	PackageName         string                           // Name of Go package
	PluginName          string                           // Name of Go type
	BlockHeading        string                           // Contents of the main header
	Description         string                           // Description paragraph(s)
	Parameters          map[string]Definition            // This plugin's own config parameters
	InheritedParameters map[string]map[string]Definition // Inherited config parameters
	Metadata            map[string]Definition            // This plugin's own metadata fields
	InheritedMetadata   map[string]map[string]Definition // Inherited metadata fields
	Example             string                           // Config example paragraph
}

// Definition represents a single metadata or configuration parameter definition
type Definition struct {
	desc string
	dfl  string
	unit string
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
	headingConfigurationExample string = "Example"
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

// DumpString returns a human-readable string dump of this object
func (doc *PluginDocument) DumpString() string {
	str := ""
	str += "==================== START DUMP ============================\n"
	str += "PackageName: [" + doc.PackageName + "]\n"
	str += "PluginName:  [" + doc.PluginName + "]\n"
	str += "BlockHeading: [" + doc.BlockHeading + "]\n"
	str += "Description: [" + doc.Description + "]\n\n"
	str += "Parameters: [" + "\n"
	str += dumpDefinitionMap(doc.Parameters) + "\n"
	str += "]\n\n"
	for parentName, defMap := range doc.InheritedParameters {
		str += "Parameters (from " + parentName + ")[\n"
		str += dumpDefinitionMap(defMap) + "\n"
		str += "]\n\n"
	}
	str += "Metadata: [" + "\n"
	str += dumpDefinitionMap(doc.Metadata) + "\n"
	for parentName, defMap := range doc.InheritedMetadata {
		str += "Metadata (from " + parentName + ")[\n"
		str += dumpDefinitionMap(defMap) + "\n"
		str += "]\n\n"
	}
	str += "Example: [" + "\n------------\n" + doc.Example + "\n----------]\n"
	str += "==================== END DUMP ============================\n"
	return str
}

func dumpDefinitionMap(defMap map[string]Definition) string {
	str := ""
	for name, def := range defMap {
		str += "- " + name + "\n" + def.dump() + "\n"
	}
	return str
}

func (def *Definition) dump() string {
	str := ""
	str += "    desc: " + def.desc + "\n"
	str += "    dfl:  " + def.dfl + "\n"
	str += "    unit: " + def.unit + "\n"
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

	_, doc.BlockHeading, _ = lines.next()
	if _, tmp, _ := lines.next(); tmp != "" {
		panic(fmt.Sprintf("Expected empty line after block heading, got \"%s\"", tmp))
	}

	section := sectionStart
	line, trimmedLine, lineNr := "", "", 0

	metadataText := ""
	parametersText := ""

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

	doc.Metadata = parseDefinitionList(metadataText)
	doc.Parameters = parseDefinitionList(parametersText)
}

func parseDefinitionList(text string) map[string]Definition {
	startDefRE := regexp.MustCompile("^- ([^:]+): (.*)")
	definitions := make(map[string]Definition)

	def := Definition{}
	defName := ""
	for _, line := range strings.Split(text, "\n") {
		trimmedLine := strings.Trim(line, " \t")
		if trimmedLine == "" {
			continue
		}

		if startDefRE.FindString(trimmedLine) != "" {
			// RE matches, store old one
			if defName != "" {
				definitions[defName] = def
			}

			// start new one
			defName = startDefRE.ReplaceAllString(trimmedLine, "$1")
			def = Definition{
				desc: startDefRE.ReplaceAllString(trimmedLine, "$2\n"),
			}
		} else if defName == "" {
			panic(fmt.Sprintf(
				"Definition list input looks malformed, cannot parse name of config parameter or metadata field around \"%s\"",
				trimmedLine))
		} else {
			// append to old one
			def.desc += trimmedLine + "\n"
		}
	}

	if defName != "" {
		definitions[defName] = def
	}

	return definitions
}

// InheritMetadata imports the .Metadata property of `document` into this document's
// inherited metadata list
func (doc *PluginDocument) InheritMetadata(parentDoc PluginDocument) {
	if doc.InheritedMetadata == nil {
		doc.InheritedMetadata = make(map[string]map[string]Definition)
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
		doc.InheritedParameters = make(map[string]map[string]Definition)
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

	//////////////////////////////////////////////////////////
	// Print top comment
	result += ".. Autogenerated by Gollum RST generator (docs/generator/*.go)\n\n"

	//////////////////////////////////////////////////////////
	// Print heading
	result += doc.PluginName + "\n"
	result += strings.Repeat("=", len(doc.PluginName)) + "\n"

	//////////////////////////////////////////////////////////
	// Print description
	result += "\n"
	result += docBulletsToRstBullets(doc.Description) + "\n"
	result += "\n"

	//////////////////////////////////////////////////////////
	// Print native metadata
	if len(doc.Metadata) > 0 {
		result += getRstHeading("Metadata")
		for name, def := range doc.Metadata {
			result += "**" + name + "**\n"
			result += docBulletsToRstBullets(def.desc) + "\n"
		}
	}

	//////////////////////////////////////////////////////////
	// Print inherited metadata
	for parentName, definitions := range doc.InheritedMetadata {
		if len(definitions) == 0 {
			// Skip title for empty sets
			continue
		}
		result += getRstHeading(
			"Metadata (from " + strings.TrimPrefix(parentName, "core.") + ")")

		for name, def := range definitions {
			result += "**" + name + "**\n"
			result += def.desc + "\n"
			result += "\n"
		}
	}

	//////////////////////////////////////////////////////////
	// Print native parameters
	if len(doc.Parameters) > 0 {
		result += getRstHeading("Parameters")
		for name, def := range doc.Parameters {
			result += "**" + name + "**\n"
			result += docBulletsToRstBullets(def.desc)
			// FIXME: cleaner formatting
			if def.dfl != "" {
				result += fmt.Sprintf("Default: %s", def.dfl)
			}
			if def.dfl != "" && def.unit != "" {
				result += ", "
			}
			if def.unit != "" {
				result += fmt.Sprintf("Unit: %s", def.unit)
			}
			if def.unit != "" || def.dfl != "" {
				result += "\n\n"
			}
		}
	}

	//////////////////////////////////////////////////////////
	// Print inherited parameters
	for parentName, definitions := range doc.InheritedParameters {
		if len(definitions) == 0 {
			// Skip title for empty param sets
			continue
		}
		result += getRstHeading(
			"Parameters (from " + strings.TrimPrefix(parentName, "core.") + ")")

		for name, def := range definitions {
			result += "**" + name + "**\n"
			result += def.desc + "\n"
			// FIXME: cleaner formatting
			if def.dfl != "" {
				result += fmt.Sprintf("Default: %s", def.dfl)
			}
			if def.dfl != "" && def.unit != "" {
				result += ", "
			}
			if def.unit != "" {
				result += fmt.Sprintf("Unit: %s", def.unit)
			}
			if def.unit != "" || def.dfl != "" {
				result += "\n"
			}
			result += "\n"
		}
	}

	//////////////////////////////////////////////////////////
	// Print config example
	if len(doc.Example) > 0 {
		result += getRstHeading("Examples")
		result += ".. code-block:: yaml\n\n"
		for _, line := range strings.Split(doc.Example, "\n") {
			result += "\t" + line + "\n"
		}
	}

	return result
}

func getRstHeading(str string) string {
	return str + "\n" + strings.Repeat("-", len(str)) + "\n\n"
}
