package grok

import (
	"regexp"
)

// Config is used to pass a set of configuration values to the grok.New function.
type Config struct {
	NamedCapturesOnly   bool
	SkipDefaultPatterns bool
	RemoveEmptyValues   bool
	Patterns            map[string]string
}

// Grok holds a cache of known pattern substitions and acts as a builder for
// compiled grok patterns. All pattern substitutions must be passed at creation
// time and cannot be changed during runtime.
type Grok struct {
	patterns    patternMap
	removeEmpty bool
	namedOnly   bool
}

// New returns a Grok object that caches a given set of patterns and creates
// compiled grok patterns based on the passed configuration settings.
// You can use multiple grok objects that act independently.
func New(config Config) (*Grok, error) {
	patterns := patternMap{}

	if !config.SkipDefaultPatterns {
		// Add default patterns first so they can be referenced later
		if err := patterns.addList(DefaultPatterns, config.NamedCapturesOnly); err != nil {
			return nil, err
		}
	}

	// Add passed patterns
	if err := patterns.addList(config.Patterns, config.NamedCapturesOnly); err != nil {
		return nil, err
	}

	return &Grok{
		patterns:    patterns,
		namedOnly:   config.NamedCapturesOnly,
		removeEmpty: config.RemoveEmptyValues,
	}, nil
}

// Compile precompiles a given grok expression. This function should be used
// when a grok expression is used more than once.
func (grok Grok) Compile(pattern string) (*CompiledGrok, error) {
	grokPattern, err := newPattern(pattern, grok.patterns, grok.namedOnly)
	if err != nil {
		return nil, err
	}

	compiled, err := regexp.Compile(grokPattern.expression)
	if err != nil {
		return nil, err
	}

	return &CompiledGrok{
		regexp:      compiled,
		typeHints:   grokPattern.typeHints,
		removeEmpty: grok.removeEmpty,
	}, nil
}

// Match returns true if the given data matches the pattern.
// The given pattern is compiled on every call to this function.
// If you want to call this function more than once consider using Compile.
func (grok Grok) Match(pattern string, data []byte) (bool, error) {
	complied, err := grok.Compile(pattern)
	if err != nil {
		return false, err
	}

	return complied.Match(data), nil
}

// MatchString returns true if the given text matches the pattern.
// The given pattern is compiled on every call to this function.
// If you want to call this function more than once consider using Compile.
func (grok Grok) MatchString(pattern, text string) (bool, error) {
	complied, err := grok.Compile(pattern)
	if err != nil {
		return false, err
	}

	return complied.MatchString(text), nil
}

// Parse processes the given data and returns a map containing the values of
// all named fields as byte arrays. If a field is parsed more than once, the
// last match is return.
// The given pattern is compiled on every call to this function.
// If you want to call this function more than once consider using Compile.
func (grok Grok) Parse(pattern string, data []byte) (map[string][]byte, error) {
	complied, err := grok.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return complied.Parse(data), nil
}

// ParseString processes the given text and returns a map containing the values of
// all named fields as strings. If a field is parsed more than once, the
// last match is return.
// The given pattern is compiled on every call to this function.
// If you want to call this function more than once consider using Compile.
func (grok Grok) ParseString(pattern, text string) (map[string]string, error) {
	complied, err := grok.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return complied.ParseString(text), nil
}

// ParseTyped processes the given data and returns a map containing the values
// of all named fields converted to their corresponding types. If no typehint is
// given, the value will be converted to string.
// The given pattern is compiled on every call to this function.
// If you want to call this function more than once consider using Compile.
func (grok Grok) ParseTyped(pattern string, data []byte) (map[string]interface{}, error) {
	complied, err := grok.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return complied.ParseTyped(data)
}

// ParseStringTyped processes the given data and returns a map containing the
// values of all named fields converted to their corresponding types. If no
// typehint is given, the value will be converted to string.
// The given pattern is compiled on every call to this function.
// If you want to call this function more than once consider using Compile.
func (grok Grok) ParseStringTyped(pattern, text string) (map[string]interface{}, error) {
	complied, err := grok.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return complied.ParseStringTyped(text)
}

// ParseToMultiMap acts like Parse but allows multiple matches per field.
// The given pattern is compiled on every call to this function.
// If you want to call this function more than once consider using Compile.
func (grok Grok) ParseToMultiMap(pattern string, data []byte) (map[string][][]byte, error) {
	complied, err := grok.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return complied.ParseToMultiMap(data), nil
}

// ParseStringToMultiMap acts like ParseString but allows multiple matches per
// field.
// The given pattern is compiled on every call to this function.
// If you want to call this function more than once consider using Compile.
func (grok Grok) ParseStringToMultiMap(pattern, text string) (map[string][]string, error) {
	complied, err := grok.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return complied.ParseStringToMultiMap(text), nil
}
