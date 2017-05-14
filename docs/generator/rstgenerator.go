package main

import (
	"strings"
)

// Returns an RST representation of this PluginDocument
func PluginDocumentToRST(doc PluginDocument) string {
	// Heading
	result := doc.PackageName + "." + doc.PluginName
	if doc.Comment != "" {
		result += " " + doc.Comment
	}
	result += "\n" + strings.Repeat("=", len(result))

	//
	result += "\n\n"
	result += rstAddLineBreaks(doc.Description, "")
	result += "\n\n"

	//
	if len(doc.Parameters) > 0 {
		result += "Parameters\n----------\n\n"

		for _, p := range doc.Parameters {
			result += "**" + p.name + "**\n"
			result += rstAddLineBreaks(p.desc, "  ")
			result += "\n"
		}
	}

	//
	for paramSetName, paramSet := range doc.ParameterSets {
		head := "Parameters (from " + paramSetName + ")"
		result += head + "\n"
		result += strings.Repeat("-", len(head)) + "\n\n"

		for _, p := range paramSet {
			result += "**" + p.name + "**\n"
			result += rstAddLineBreaks(p.desc, "  ")
			result += "\n"
		}
	}

	//
	result += "Example\n-------\n\n"
	result += ".. code-block:: yaml\n\n"
	result += "\t" + doc.ExampleFirstLine + "\n"
	for _, line := range doc.Example {
		result += "\t" + line + "\n"
	}

	return result
}

func rstAddLineBreaks(text string, prefix string) string {
	sentences := strings.Split(text, ". ")
	result := ""
	skipPrefix := false
	isEnum := false

	for _, s := range sentences {
		sLower := strings.ToLower(s)
		sTrim := strings.Trim(s, " ")
		switch {
		case len(sTrim) <= 1:
			// ignore

		case isEnum && sTrim[0] == enumChar:
			postfix := ". "
			if sTrim[len(sTrim)-1] == '.' {
				postfix = " "
			}
			result += "\n" + prefix + s + postfix

		case isEnum:
			result += sTrim + ". "
			if sLower[0] != ' ' && sLower[1] == ' ' {
				isEnum = false
			}

		case sTrim[0] == enumChar:
			result += prefix + s + ". "
			isEnum = true

		case strings.Contains(sLower, "e.g") || strings.Contains(sLower, "i.e"):
			result += prefix + s + ". "
			skipPrefix = true

		default:
			postfix := ".\n"
			if sTrim[len(sTrim)-1] == '.' {
				postfix = "\n"
			}

			if skipPrefix {
				result += s + postfix
				skipPrefix = false
			} else {
				result += prefix + s + postfix
			}
		}
	}

	if len(result) > 1 && result[len(result)-1] != '\n' {
		result += "\n"
	}

	return result
}

