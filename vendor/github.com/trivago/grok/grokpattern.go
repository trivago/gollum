package grok

import (
	"fmt"
	"regexp"
	"strings"
)

type grokPattern struct {
	expression string
	typeHints  typeHintByKey
}

var (
	namedReference = regexp.MustCompile(`%{(\w+(?::\w+(?::\w+)?)?)}`)
)

func newPattern(pattern string, knownPatterns patternMap, namedOnly bool) (*grokPattern, error) {
	typeHints := typeHintByKey{}

	for _, keys := range namedReference.FindAllStringSubmatch(pattern, -1) {

		names := strings.Split(keys[1], ":")
		refKey, refAlias := names[0], names[0]
		if len(names) > 1 {
			refAlias = names[1]
		}

		// Add type cast information only if type set, and not string
		if len(names) == 3 {
			if names[2] != "string" {
				typeHints[refAlias] = names[2]
			}
		}

		refPattern, patternExists := knownPatterns[refKey]
		if !patternExists {
			return nil, fmt.Errorf("no pattern found for %%{%s}", refKey)
		}

		var refExpression string
		if !namedOnly || (namedOnly && len(names) > 1) {
			refExpression = fmt.Sprintf("(?P<%s>%s)", refAlias, refPattern.expression)
		} else {
			refExpression = fmt.Sprintf("(%s)", refPattern.expression)
		}

		// Add new type Informations
		for key, typeName := range refPattern.typeHints {
			if _, hasTypeHint := typeHints[key]; !hasTypeHint {
				typeHints[key] = strings.ToLower(typeName)
			}
		}

		pattern = strings.Replace(pattern, keys[0], refExpression, -1)
	}

	return &grokPattern{
		expression: pattern,
		typeHints:  typeHints,
	}, nil
}
