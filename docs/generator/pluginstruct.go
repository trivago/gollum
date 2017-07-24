package main

// Represents a "type FooPlugin struct { .... }" declaration parsed from the
// source and its immediately preceding comment block.
type pluginStructType struct {
	Pkg  string
	Name string

	// Comment block immediately preceding this struct
	Comment string

	// Struct types embedded (inherited) by this struct
	Embeds []typeEmbed

	// Config parameters marked with struct tags (`config:"ParamName" ....`)
	Params map[string]Definition
}

// Generates a PluginDocument from the pluginStructType
func (pst pluginStructType) createPluginDocument() PluginDocument {
	// Create PluginDocument
	pluginDocument := NewPluginDocument(pst.Pkg, pst.Name)

	// Parse the comment string
	pluginDocument.ParseString(pst.Comment)

	// Set Param values from struct tags
	for paramName, paramDef := range pst.Params {

		if docParamDef, found := pluginDocument.Parameters[paramName]; found {
			paramDef.desc = docParamDef.desc
		}

		pluginDocument.Parameters[paramName] = paramDef
	}

	// - Recursively generate PluginDocuments for all embedded types (SimpleProducer etc.)
	//   with embedTag in this plugin's embed declaration
	// - Include their parameter lists in this doc
	// - Include their metadata lists in this doc
	for _, embed := range pst.Embeds {
		importDir := getImportDir(embed.pkg)
		astPackage := parseSourcePath(importDir)
		for _, embedPst := range getPluginStructTypes(astPackage) {
			if embedPst.Pkg == embed.pkg && embedPst.Name == embed.name {
				// Recursion
				doc := embedPst.createPluginDocument()
				pluginDocument.InheritParameters(doc)
				pluginDocument.InheritMetadata(doc)
			}
		}
	}

	return pluginDocument
}
