package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Definition represents a single metadata or configuration parameter definition
// with optional nested definitions
type Definition struct {
	desc     string
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
type DefinitionList map[string]*Definition

// newDefinitionListFromString parses the input string and creates a (nested) DefinitionList object
func newDefinitionListFromString(text string) DefinitionList {
	emptyLineRE := regexp.MustCompile("^[[:space:]]*$")
	depthRE := regexp.MustCompile("^( *)(.*)")
	startDefRE := regexp.MustCompile("^( *)- ([^:]+):[[:space:]]*(.*)")

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
			newName := startDefRE.ReplaceAllString(line, "$2")
			newItem := &Definition{
				desc: startDefRE.ReplaceAllString(line, "$3\n"),
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
				currentList = &currentItem.parent.parent.children
			}

			if _, found := (*currentList)[newName]; found {
				panic(fmt.Sprintf("Comment line %d: Duplicate key \"%s\" near \"%s\"\n",
					lineNo, newName, line))
			}
			(*currentList)[newName] = newItem

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

func (list *DefinitionList) set(name string, def *Definition) {
	(*list)[name] = def
}

func (list *DefinitionList) dumpString() string {
	str := ""
	for name, def := range *list {
		str += "- " + name + "\n" + def.dumpString() + "\n"
	}
	return str
}

// getRST formats the DefinitionList as ReStructuredText
func (list DefinitionList) getRST(paramFields bool, depth int) string {
	// List items in alphabetical order
	var sorted []string
	for name := range list {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	result := ""
	for _, name := range sorted {
		// Heading
		result +=  indentLines("**" + name + "**", 2 * depth)

		// Optional default value and unit
		if paramFields && (list[name].unit != "" || list[name].dfl != "") {
			// TODO: cleaner formatting
			result += " ("
			if list[name].dfl != "" {
				result += fmt.Sprintf("default: %s", list[name].dfl)
			}
			if list[name].dfl != "" && list[name].unit != "" {
				result += ", "
			}
			if list[name].unit != "" {
				result += fmt.Sprintf("unit: %s", list[name].unit)
			}
			result += ")"
		}
		result += "\n\n"

		// Body
		result += indentLines(docBulletsToRstBullets(list[name].desc), 2 * (depth + 1))
		//result += list[name].desc
		result += "\n\n"

		// Children
		result += list[name].children.getRST(paramFields, depth+1)
	}

	//return indentLines(result, 2 * (depth + 1))
	return result
}

// Inserts two spaces at the beginning of each line, making the string a blockquote in RST
func indentLines(source string, level int) string {
	return regexp.MustCompile("(?m:(^))").ReplaceAllString(source, strings.Repeat(" ", level))
}
