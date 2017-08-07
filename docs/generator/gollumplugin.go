package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
)

// GollumPlugin represents a "type FooPlugin struct { .... }" declaration parsed
// from the source and its immediately preceding comment block.
type GollumPlugin struct {
	Pkg  string
	Name string

	// Comment block immediately preceding this struct
	Comment string

	// Struct types embedded (inherited) by this struct
	Embeds []typeEmbed

	// Config parameters parsed from struct tags (`config:"ParamName" ....`)
	StructTagParams DefinitionList
}

// Searches the AST rooted at `pkgRoot` for struct types representing Gollum plugins
func findGollumPlugins(pkgRoot *ast.Package) []GollumPlugin {
	// Generate a tree structure from the parse results returned by AST
	tree := NewTree(pkgRoot)

	// The tree looks like this:
	/**
		*ast.Package
		    *ast.File
		        *ast.Ident
		        *ast.GenDecl
		            *ast.ImportSpec
		                *ast.BasicLit
		        [....]
		        *ast.GenDecl
		            *ast.CommentGroup
		                *ast.Comment
		            *ast.TypeSpec
		                *ast.Ident
		                *ast.StructType
		                    *ast.FieldList
		                        *ast.Field
		                            *ast.Ident
		                            *ast.Ident
		        [....[
	**/

	// Search pattern
	pattern := PatternNode{
		Comparison: "*ast.GenDecl",
		Children: []PatternNode{
			{
				Comparison: "*ast.CommentGroup",
				// Note: there are N pcs of "*ast.Comment" children
				// here but we don't need to match them
			},
			{
				Comparison: "*ast.TypeSpec",
				Children: []PatternNode{
					{
						Comparison: "*ast.Ident",
						Callback: func(astNode ast.Node) bool {
							// Checks that the name starts with a capital letter
							return ast.IsExported(astNode.(*ast.Ident).Name)
						},
					},
					{
						Comparison: "*ast.StructType",
						Children: []PatternNode{
							{
								Comparison: "*ast.FieldList",
								// There are N pcs of "*ast.Field" children here
							},
						},
					},
				},
			},
		},
	}

	// Search the tree
	results := []GollumPlugin{}
	for _, genDecl := range tree.Search(pattern) {
		pst := GollumPlugin{
			Pkg: pkgRoot.Name,
			// Indexes assumed based on the search pattern above
			Name:    genDecl.Children[1].Children[0].AstNode.(*ast.Ident).Name,
			Comment: genDecl.Children[0].AstNode.(*ast.CommentGroup).Text(),
			Embeds: getGollumPluginEmbedList(pkgRoot.Name,
				genDecl.Children[1].Children[1].Children[0].AstNode.(*ast.FieldList),
			),
			StructTagParams: getGollumPluginConfigParams(
				genDecl.Children[1].Children[1].Children[0].AstNode.(*ast.FieldList),
			),
		}

		results = append(results, pst)
	}

	return results
}

// Returns a list of type embeds in fieldList marked with embedTag for findGollumPlugins()
func getGollumPluginEmbedList(packageName string, fieldList *ast.FieldList) []typeEmbed {
	results := []typeEmbed{}
	for _, field := range fieldList.List {
		if field.Tag == nil ||
			field.Tag.Kind != token.STRING ||
			field.Tag.Value != embedTag {
			continue
		}

		switch t := field.Type.(type) {
		case *ast.Ident:
			// Relative reference within the same package ("SimpleConsumer")
			results = append(results, typeEmbed{
				pkg:  packageName,
				name: field.Type.(*ast.Ident).Name,
			})

		case *ast.SelectorExpr:
			// Reference to another package ("core.SimpleConsumer")
			sel := field.Type.(*ast.SelectorExpr)

			results = append(results, typeEmbed{
				pkg:  sel.X.(*ast.Ident).Name,
				name: sel.Sel.Name,
			})

		default:
			_ = t
			fmt.Printf("WARNING: struct tag %s in unexpected location\n", embedTag)
			continue
		}
	}

	return results
}

// getGollumPluginConfigParams scans the struct for properties that map directly
// to config params (identified by tags like `config:"TimeoutMs" default:"0" metric:"ms"`)
// and returns a map of Definitions with the value of "config" as key.
func getGollumPluginConfigParams(fieldList *ast.FieldList) DefinitionList {
	results := DefinitionList{}

	for _, field := range fieldList.List {
		if field.Tag == nil || field.Tag.Kind != token.STRING {
			continue
		}
		tags := parseStructTag(field.Tag.Value)
		if paramName, found := tags["config"]; found {
			results.add(&Definition{
				name: paramName,
				// Default description if not overridden in the plugin's comment block
				desc: "(no documentation available)",
				dfl:  tags["default"],
				unit: tags["metric"],
			})
		}
	}

	return results
}

// parseStructTag parses a string like
//
//   `config:"ReadTimeoutSec" default:"3" metric:"sec"`
//
// into a map of tag names and values
func parseStructTag(tag string) map[string]string {
	re := regexp.MustCompile("^([^:\"]+):\"([^\"]*)\" *(.*)")
	result := map[string]string{}

	if tag[0] != '`' || tag[len(tag)-1] != '`' {
		panic(fmt.Sprintf("Expected tag to begin and end with `, got: \"%s\"", tag))
	}

	parseStructTagInner(re, result, tag[1:len(tag)-1])
	return result
}

// Recursive component of parseStructTag()
func parseStructTagInner(re *regexp.Regexp, result map[string]string, tag string) {
	if tag == "" || re.FindString(tag) == "" {
		return
	}

	tagName := re.ReplaceAllString(tag, "$1")
	tagValue := re.ReplaceAllString(tag, "$2")
	rest := re.ReplaceAllString(tag, "$3")

	result[tagName] = tagValue
	parseStructTagInner(re, result, rest)
}
