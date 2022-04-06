package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/invopop/jsonschema"
	ortfodb "github.com/ortfo/db"
)

func main() {
	writeSchema(&ortfodb.Configuration{}, "configuration")
	writeSchema(&[]ortfodb.Work{}, "database")
}

func writeSchema(typeInstance interface{}, schemaName string) {
	// Don't make $defs, it's unecessary cruft at this small scale
	reflector := &jsonschema.Reflector{DoNotReference: true}
	// Infer the schema
	schema := reflector.Reflect(typeInstance)
	// Write the JSON back to ../[name of schema].schema.json
	ioutil.WriteFile("../"+schemaName+".schema.json", toJSON(schema), 0o644)
}

func toJSON(schema *jsonschema.Schema) []byte {
	// Marshal the schema
	schemaJSON, err := schema.MarshalJSON()
	// Errors shouldn't happen here
	if err != nil {
		panic(err)
	}
	// Ident it
	var schemaIndented bytes.Buffer
	json.Indent(&schemaIndented, schemaJSON, "", "  ")
	schemaStr := schemaIndented.String()
	// Make all lowercase (most keys are PascalCase because of Go conventions)
	schemaStr = strings.ToLower(schemaStr)
	// Fix $schema not working in vscode because of missing TLS
	schemaStr = strings.Replace(schemaStr, "http://json-schema.org/draft/2020-12/schema", "https://json-schema.org/draft/2020-12/schema", 1)
	// Turn back into bytes
	return []byte(schemaStr)
}
