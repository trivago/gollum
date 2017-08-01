// Documentation generator for Gollum
//
// Reads in Go source files from a given file or directory.
// Generates end-user documentation in RST format for Gollum plugins found therein.
//
// The data is processed as follows:
//
//   *.go sources
//     -> ast.Node/ast.Package/etc objects
//       -> GollumPlugin objects
//          -> PluginDocument objects [recursively process embedded types' sources]
//             -> RST strings
//
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"strings"
)

const embedTag = "`gollumdoc:\"embed_type\"`"

// Map of imported package names => package paths
var globalImportMap map[string]string

type typeEmbed struct {
	pkg  string
	name string
}

// Main
func main() {
	// Args
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <source.go> <destination.rst>\n", os.Args[0])
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
	sourcePackage := parseSourcePath(sourcePath)
	globalImportMap = getPackageImportMap(sourcePackage)

	// Search for "type FooPlugin struct { ... }" declarations
	pluginList := findGollumPlugins(sourcePackage)

	// Parse struct declarations into PluginDocument objects
	docList := []PluginDocument{}
	for _, plugin := range pluginList {
		docList = append(docList, NewPluginDocument(plugin))
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
func parseSourcePath(sourcePath string) *ast.Package {
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
		return !strings.HasSuffix(filterStat.Name(), "_test.go")
	}

	if !sourceFileStat.IsDir() {
		parseDir = strings.TrimSuffix(sourcePath, "/"+sourceFileStat.Name())
		parseFilter = func(filterStat os.FileInfo) bool {
			return filterStat.Name() == sourceFileStat.Name() &&
				!strings.HasSuffix(filterStat.Name(), "_test.go")
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
		return packageAst
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
	if !found {
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
		if !stat.IsDir() {
			continue
		}

		return testPath
	}

	return ""
}

// Returns a map of package names and their paths imported by astPackage:
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
			packagePath := importSpec.Path.Value[1 : len(importSpec.Path.Value)-1]

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

		// Add this file's local path to import list
		// - there may not be other references to this package
		tmp, exists := result[file.Name.Name]
		if exists && tmp != path.Dir(fileName) {
			fmt.Printf("WARNING: Possible package naming conflict in file %q: package %q "+
				"references both importdir %q and relative dir %q\n",
				fileName, file.Name.Name, tmp, path.Dir(fileName))
		}
		result[file.Name.Name] = path.Dir(fileName)
	}

	return result
}
