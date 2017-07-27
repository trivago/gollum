package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Definition represents a single metadata or configuration parameter definition
// with optional nested definitions
type Definition struct {
	desc     string
	name     string
	dfl      string
	unit     string
	parent   *Definition
	children DefinitionList
}

func (def *Definition) dumpString() string {
	str := ""
	str += "    desc: " + def.desc + "\n"
	str += "    dfl:  " + def.dfl + "\n"
	str += "    unit: " + def.unit + "\n"
	return str
}

// DefinitionList contains a map of definitions
type DefinitionList struct {
	slice []*Definition
}

// newDefinitionListFromString parses the input string and creates a (nested) DefinitionList object
func newDefinitionListFromString(text string) DefinitionList {
	emptyLineRE := regexp.MustCompile("^[[:space:]]*$")
	depthRE := regexp.MustCompile("^( *)(.*)")

	startDefRE := regexp.MustCompile("^ *- (.*)")
	keyedDefRE := regexp.MustCompile("^([^:]+):[[:space:]]*(.*)")

	var (
		currentList  *DefinitionList
		currentItem  *Definition
		currentDepth int
	)

	topList := DefinitionList{}
	currentList = &topList

	for lineNo, line := range strings.Split(text, "\n") {
		// nesting depth == nr. of indentation spaces
		if emptyLineRE.MatchString(line) {
			continue
		}
		newDepth := len(depthRE.ReplaceAllString(line, "$1"))

		if startDefRE.MatchString(line) {
			// Start new definition
			start := startDefRE.ReplaceAllString(line, "$1")

			newItem := &Definition{}
			if keyedDefRE.MatchString(start) {
				newItem.name = keyedDefRE.ReplaceAllString(start, "$1")
				newItem.desc = keyedDefRE.ReplaceAllString(start, "$2") + "\n"

			} else {
				newItem.name = ""
				newItem.desc = start + "\n"
			}

			if newDepth == currentDepth {
				// Keep current nesting level
				if currentItem != nil {
					newItem.parent = currentItem.parent
				}

			} else if newDepth > currentDepth {
				// Dive deeper
				newItem.parent = currentItem
				currentItem.children = DefinitionList{}
				currentList = &currentItem.children

			} else if newDepth < currentDepth {
				// One up
				if currentItem.parent.parent != nil {
					currentList = &currentItem.parent.parent.children
				} else {
					currentList = &topList
				}
			}

			currentList.add(newItem)
			currentItem = newItem
			currentDepth = newDepth

		} else if currentItem == nil {
			// Parse error
			panic(fmt.Sprintf("Comment line %d: Parse error: currentItem == nil "+
				"near \"%s\" (plain paragraphs mixed with definition list?)\n", lineNo, line))

		} else if newDepth != currentDepth {
			// Parse error
			panic(fmt.Sprintf("Comment line %d: Indentation error: expected indentation "+
				"to remain at %d, found %d near \"%s\"\n", lineNo, currentDepth, newDepth, line))

		} else {
			// Append to current definition (without indentation)
			currentItem.desc += depthRE.ReplaceAllString(line, "$2") + "\n"
		}
	}

	return topList
}

func (list *DefinitionList) add(def *Definition) {
	list.slice = append(list.slice, def)
}

func (list *DefinitionList) findByName(name string) (*Definition, bool) {
	for _, def := range list.slice {
		if def.name == name {
			return def, true
		}
	}
	return nil, false
}

func (list *DefinitionList) dumpString() string {
	str := ""
	for _, def := range list.slice {
		str += "- " + def.name + "\n" + def.dumpString() + "\n"
	}
	return str
}

// getRST formats the DefinitionList as ReStructuredText
func (list DefinitionList) getRST(paramFields bool, depth int) string {
	/***
	// List items in alphabetical order
	var sorted []string
	for _, def := range list.slice {
		sorted = append(sorted, def.name)
	}
	sort.Strings(sorted)
	***/

	result := ""
	for _, def := range list.slice {
		// Heading
		if strings.Trim(def.name, " \t") != "" {
			result += indentLines("**"+def.name+"**", 2*depth)
		} else {
			// FIXME: bullet lists or something
			result += "** ... **"
		}

		// Optional default value and unit
		if paramFields && (def.unit != "" || def.dfl != "") {
			// TODO: cleaner formatting
			result += " ("
			if def.dfl != "" {
				result += fmt.Sprintf("default: %s", def.dfl)
			}
			if def.dfl != "" && def.unit != "" {
				result += ", "
			}
			if def.unit != "" {
				result += fmt.Sprintf("unit: %s", def.unit)
			}
			result += ")"
		}
		result += "\n\n"

		// Body
		result += indentLines(docBulletsToRstBullets(def.desc), 2*(depth+1))
		//result += def.desc
		result += "\n\n"

		// Children
		result += def.children.getRST(paramFields, depth+1)
	}

	//return indentLines(result, 2 * (depth + 1))
	return result
}

// Inserts two spaces at the beginning of each line, making the string a blockquote in RST
func indentLines(source string, level int) string {
	return regexp.MustCompile("(?m:(^))").ReplaceAllString(source, strings.Repeat(" ", level))
}
