package github

import "github.com/invopop/jsonschema"

func (Repository) JSONSchemaExtend(schema *jsonschema.Schema) {
	topics, ok := schema.Properties.Get("topics")
	if !ok {
		return
	}
	topics.Items = &jsonschema.Schema{
		Type:    "string",
		Pattern: `^[a-z0-9][a-z0-9-]{0,49}$`,
	}
}
