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

	// Config parameters parsed from struct tags (`config:"ParamName" ....`)
	StructTagParams DefinitionList
}

// Generates a PluginDocument from the pluginStructType
func (pst pluginStructType) createPluginDocument() PluginDocument {
	// Create PluginDocument
	pluginDocument := NewPluginDocument(pst.Pkg, pst.Name)

	// Parse the comment string
	pluginDocument.ParseString(pst.Comment)

	// Set Param values from struct tags
	for _, stParamDef := range pst.StructTagParams.slice {

		if docParamDef, found := pluginDocument.Parameters.findByName(stParamDef.name); found {
			// Parameter is already documented, add values from struct tags
			docParamDef.unit = stParamDef.unit
			docParamDef.dfl = stParamDef.dfl

		} else {
			// Undocumented parameter
			pluginDocument.Parameters.add(stParamDef)
		}
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
