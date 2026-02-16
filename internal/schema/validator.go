package schema

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

// Validator handles JSON schema validation for tool arguments.
// It caches compiled schemas for performance.
type Validator struct {
	cache sync.Map // map[string]*gojsonschema.Schema
}

// NewValidator creates a new validator instance.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks if the JSON arguments string matches the provided schema.
// The schema can be a map[string]any, a string (JSON), or a struct.
func (v *Validator) Validate(schemaData any, argsJSON string) error {
	// 1. Get or compile the schema
	schemaLoader, err := v.getSchemaLoader(schemaData)
	if err != nil {
		return fmt.Errorf("invalid schema definition: %w", err)
	}

	// 2. Prepare the document loader
	documentLoader := gojsonschema.NewStringLoader(argsJSON)

	// 3. Validate
	result, err := schemaLoader.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation execution failed: %w", err)
	}

	if result.Valid() {
		return nil
	}

	// 4. Format errors
	var errs []string
	for _, desc := range result.Errors() {
		errs = append(errs, desc.String())
	}
	return fmt.Errorf("schema validation failed:\n- %s", dumpErrors(errs))
}

func (v *Validator) getSchemaLoader(schemaData any) (*gojsonschema.Schema, error) {
	// Create a stable key for caching.
	// For map/structs, we marshal to JSON.
	jsonBytes, err := json.Marshal(schemaData)
	if err != nil {
		return nil, err
	}
	key := string(jsonBytes)

	// Check cache
	if val, ok := v.cache.Load(key); ok {
		return val.(*gojsonschema.Schema), nil
	}

	// Compile
	loader := gojsonschema.NewBytesLoader(jsonBytes)
	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return nil, err
	}

	// Cache it
	v.cache.Store(key, schema)
	return schema, nil
}

func dumpErrors(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	if len(errs) == 1 {
		return errs[0]
	}
	// return first 3 errors to avoid massive output
	truncated := ""
	if len(errs) > 3 {
		truncated = fmt.Sprintf("... and %d more", len(errs)-3)
		errs = errs[:3]
	}

	// Join with newlines
	result := ""
	for i, e := range errs {
		if i > 0 {
			result += "\n- "
		}
		result += e
	}
	return result + truncated
}
