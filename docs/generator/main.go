// Documentation generator for Gollum
//
// Reads in Go source files from a given file or directory.
// Generates end-user documentation in RST format for Gollum plugins found therein.
//
// The data is processed as follows:
//
//   *.go sources
//     -> ast.Node/ast.Package/etc objects
//       -> pluginStructType objects
//          -> PluginDocument objects [recursively process embedded types' sources]
//             -> RST strings
//
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"io"
	"strings"
	"path"
)

const EMBED_TAG = "`gollumdoc:\"embed_type\"`"

// Represents a "type FooPlugin struct { .... }" declaration parsed from the
// source and its immediately preceding comment block.
type pluginStructType struct {
	Pkg         string
	Name        string
	Comment     string
	Embeds      []typeEmbed
}
type typeEmbed struct {
	pkg string
	name string
}

// Map of imported package names => package paths
var globalImportMap map[string]string

// Main
func main() {
	// Args
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <source.gp> <destination.rst>\n", os.Args[0])
		os.Exit(1)
	}
	sourcePath, destFilePath := os.Args[1], os.Args[2]

	// Open output file
	outFile, err := os.Create(destFilePath)
	defer outFile.Close()
	if err != nil {
		panic(err)
	}

	// Read .go sources from file|directory sourcePath
	sourcePackage, _ := parseSourcePath(sourcePath)
	globalImportMap = getPackageImportMap(sourcePackage)

	// Search for "type FooPlugin struct { ... }" declarations
	pstList := getPluginStructTypes(sourcePackage)

	// Parse struct declarations into PluginDocument objects
	docList := []PluginDocument{}
	for _, pst := range pstList {
		docList = append(docList, pst.createPluginDocument())
	}

	// Generate an RST file from the PluginDocuments
	for _, doc := range docList {
		io.WriteString(outFile, doc.GetRST())
		io.WriteString(outFile, "\n\n")
	}
}

// Parses the .go sourcers in the file|directory `sourcePath ` into an ast.Package object.
// This lets us blindly accept files and directories as input. Expects to find exactly one
// package under sourcePath. Returns (*ast.Package, *token.FileSet). Panics on errors.
func parseSourcePath(sourcePath string) (*ast.Package, *token.FileSet) {
	// stat() the sourcePath
	sourceFileHandle, err := os.Open(sourcePath)
	if err != nil {
		panic(err)
	}
	sourceFileStat, err := sourceFileHandle.Stat()
	if err != nil {
		panic(err)
	}
	defer sourceFileHandle.Close()

	// If sourcePath is a file, only parse that file.
	// Always ignore *_test.go files to avoid unexpected "foo_test" packages.
	parseDir := sourcePath
	parseFilter := func(filterStat os.FileInfo) bool {
		return ! strings.HasSuffix(filterStat.Name(), "_test.go")
	}

	if ! sourceFileStat.IsDir() {
		parseDir = strings.TrimSuffix(sourcePath, "/" + sourceFileStat.Name())
		parseFilter = func(filterStat os.FileInfo) bool {
			return filterStat.Name() == sourceFileStat.Name() &&
				! strings.HasSuffix(filterStat.Name(), "_test.go")
		}
	}

	// Parse sources in the file or directory
	fileSet := token.NewFileSet()
	packageMap, err := parser.ParseDir(fileSet, parseDir, parseFilter, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	if len(packageMap) != 1 {
		panic(fmt.Sprintf("ERROR: Path %s contains %d packages instead of one\n",
			sourcePath, len(packageMap)))
	}

	for _, packageAst := range packageMap {
		return packageAst, fileSet
	}
	panic("Not reached")
}

// Returns the filesystem path for the given package name
// Searches "./", "./vendor/" and "${GOPATH}/src/"
// Returns "" if the import was not found
//
// https://groups.google.com/forum/#!topic/golang-nuts/iv63CKEG2do
func getImportDir(packageName string) string {
	packagePath, found := globalImportMap[packageName]
	if ! found {
		return ""
	}

	// ./<packagePath> is needed to resolve references to plugin's own package
	searchList := []string{
		"./" + packageName,
		"./" + packagePath,
		"./vendor/" + packagePath,
		os.Getenv("GOPATH") + "/src/" + packagePath,
	}

	for _, testPath := range searchList {
		fp, err := os.Open(testPath)
		if err != nil {
			continue
		}
		stat, err := fp.Stat()
		if err != nil {
			continue
		}
		if ! stat.IsDir() {
			continue
		}

		return testPath
	}

	return ""
}

// Returns a map package names and their paths imported by astPackage:
//   packagename => package path
//    "foo"      => "github.com/bar/foo"
func getPackageImportMap(astPackage *ast.Package) map[string]string {

	result := make(map[string]string)

	// Iterate files in this package
	// (There's also an *ast.Package.Imports property but it seems to be empty)
	for fileName, file := range astPackage.Files {
		// Iterate imports in this file
		for _, importSpec := range file.Imports {
			// Remove quotes
			packagePath := importSpec.Path.Value[1:len(importSpec.Path.Value)-1]

			// Determine package name
			var packageName string
			if importSpec.Name != nil {
				// Package is aliased to another name
				packageName = importSpec.Name.Name

			} else {
				// Use the default name, the path's last element
				tmp := strings.Split(packagePath, "/")
				packageName = tmp[len(tmp)-1]
			}

			// Check for naming conflicts
			tmp, exists := result[packageName]
			if exists && tmp != packagePath {
				panic(fmt.Sprintf("Package naming conflict in %s: %s => %s, was already %s\n",
					fileName, packageName, tmp, packagePath))
			}

			result[packageName] = packagePath
		}

		// Add this file's local path to import list - there may not be other refernces to this package
		tmp, exists := result[file.Name.Name]
		if exists && tmp != path.Dir(fileName) {
			fmt.Printf("WARNING: Possible package naming conflict in file %q: package %q references both importdir %q and relative dir %q\n",
				fileName, file.Name.Name, tmp, path.Dir(fileName))
		}
		result[file.Name.Name] = path.Dir(fileName)
	}

	return result
}

// Searches the AST rooted at `pkgRoot` for struct types representing Gollum plugins
func getPluginStructTypes(pkgRoot *ast.Package) []pluginStructType {
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
	pattern := PatternNode {
		Comparison: "*ast.GenDecl",
        Children: []PatternNode {
			{
				Comparison: "*ast.CommentGroup",
				// Note: there are N pcs of "*ast.Comment" children
				// here but we don't need to match them
			},
			{
				Comparison: "*ast.TypeSpec",
				Children: []PatternNode {
					{
						Comparison: "*ast.Ident",
						Callback:   func(astNode ast.Node) bool {
							// Checks that the name starts with a capital letter
							return ast.IsExported(astNode.(*ast.Ident).Name)
						},
					},
					{
						Comparison: "*ast.StructType",
						Children: []PatternNode {
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
	results := []pluginStructType{}
	for _, genDecl := range tree.Search(pattern) {
		pst := pluginStructType{
			Pkg:     pkgRoot.Name,
			// Indexes assumed based on the search pattern above
			Name:    genDecl.Children[1].Children[0].AstNode.(*ast.Ident).Name,
			Comment: genDecl.Children[0].AstNode.(*ast.CommentGroup).Text(),
			Embeds:  getStructTypeEmbedList(pkgRoot.Name,
				genDecl.Children[1].Children[1].Children[0].AstNode.(*ast.FieldList),
			),
		}

		results = append(results, pst)
	}

	return results
}

// Returns a list of type embeds in fieldList marked with EMBED_TAG for getPluginStructTypes()
func getStructTypeEmbedList(packageName string, fieldList *ast.FieldList) []typeEmbed {
	results := []typeEmbed{}
	for _, field := range fieldList.List {
		if field.Tag == nil ||
			field.Tag.Kind != token.STRING ||
			field.Tag.Value != EMBED_TAG {
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
			fmt.Printf("WARNING: struct tag %s in unexpected location\n", EMBED_TAG)
			continue
		}
	}

	return results
}

// Generates a PluginDocument from the pluginStructType
func (pst pluginStructType) createPluginDocument() PluginDocument {
	// Create PluginDocument and parse the comment string
	pluginDocument := NewPluginDocument(pst.Pkg, pst.Name)
	pluginDocument.ParseString(pst.Comment)

	// Recursively generate PluginDocuments for all embedded types (SimpleProducer etc.)
	// with EMBED_TAG in this plugin's embed declaration and include their parameter lists
	// in this plugin's document.
	for _, embed := range pst.Embeds {
		importDir := getImportDir(embed.pkg)
		fmt.Printf("importdir(%s): %s\n", embed.pkg, importDir)
		astPackage, _ := parseSourcePath(importDir)
		for _, embedPst := range getPluginStructTypes(astPackage) {
			if embedPst.Pkg == embed.pkg && embedPst.Name == embed.name {
				// Recursion
				doc := embedPst.createPluginDocument()
				pluginDocument.IncludeParameters(doc)
			}
		}
	}

	return pluginDocument
}
