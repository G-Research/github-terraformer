package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"
)

// printer renders the localized validation messages. A non-nil printer is
// required: ErrorKind.LocalizedString panics on a nil printer.
var printer = message.NewPrinter(language.English)

// orgSchemaRelPath is the location, relative to the config directory, where an
// organization can drop a schema that overrides the built-in one.
const orgSchemaRelPath = ".schemas/repository-config.schema.json"

var (
	validateConfigDir    string
	validateSchemaPath   string
	validateFallbackPath string
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate repository config YAML files against the JSON schema",
	Long: `Validate validates every repos/*.yaml and repos/*.yml file in the config
directory against the repository configuration JSON schema, failing fast with the
offending file and JSON path before Terraform runs.

Schema resolution order:
  1. --schema flag (explicit override)
  2. <config-dir>/` + orgSchemaRelPath + ` (org-level override, if present)
  3. --fallback-schema flag (e.g. github-terraformer's own .schemas/ file)
  4. built-in schema generated from the importer's Go structs`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(cmd, validateConfigDir, validateSchemaPath, validateFallbackPath)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringVar(&validateConfigDir, "config-dir", "", "Path to the config repository containing repos/*.yaml")
	validateCmd.Flags().StringVar(&validateSchemaPath, "schema", "", "Explicit schema path (takes precedence over all other sources)")
	validateCmd.Flags().StringVar(&validateFallbackPath, "fallback-schema", "", "Fallback schema path used when no org override exists (e.g. github-terraformer/.schemas/repository-config.schema.json)")
	_ = validateCmd.MarkFlagRequired("config-dir")
}

func runValidate(cmd *cobra.Command, configDir, schemaOverride, fallbackSchema string) error {
	schema, err := loadValidationSchema(cmd, configDir, schemaOverride, fallbackSchema)
	if err != nil {
		return err
	}

	files, err := globRepoConfigs(configDir)
	if err != nil {
		return fmt.Errorf("list repo config files: %w", err)
	}
	sort.Strings(files)

	if len(files) == 0 {
		cmd.Println("No repos/*.yaml or repos/*.yml files found, skipping validation")
		return nil
	}

	var failures []string
	failedFiles := 0
	for _, file := range files {
		fileErrs := validateFile(file, schema)
		if len(fileErrs) > 0 {
			failures = append(failures, fileErrs...)
			failedFiles++
			continue
		}
		cmd.Printf("ok -- %s\n", file)
	}

	if len(failures) > 0 {
		cmd.PrintErrln("\nSchema validation errors were encountered:")
		for _, f := range failures {
			cmd.PrintErrf("  %s\n", f)
		}
		return fmt.Errorf("schema validation failed for %d file(s)", failedFiles)
	}

	return nil
}

// globRepoConfigs returns the repos/*.yaml and repos/*.yml files under
// configDir, matching the extensions the importer's expand command accepts.
func globRepoConfigs(configDir string) ([]string, error) {
	var files []string
	for _, ext := range []string{"*.yaml", "*.yml"} {
		matches, err := filepath.Glob(filepath.Join(configDir, "repos", ext))
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	return files, nil
}

// loadValidationSchema resolves and compiles the schema to validate against.
// Resolution order:
//  1. explicit schemaOverride (--schema flag)
//  2. org-level override at <configDir>/.schemas/repository-config.schema.json
//  3. fallbackSchema (--fallback-schema flag, e.g. github-terraformer's own .schemas/ file)
//  4. built-in schema generated from the importer's Go structs
func loadValidationSchema(cmd *cobra.Command, configDir, schemaOverride, fallbackSchema string) (*jsonschema.Schema, error) {
	if schemaOverride == "" {
		if candidate, ok := resolveOrgSchema(cmd, configDir); ok {
			schemaOverride = candidate
		}
	}

	schemaPath := ""
	switch {
	case schemaOverride != "":
		cmd.Printf("Using org schema override: %s\n", schemaOverride)
		schemaPath = schemaOverride
	case fallbackSchema != "":
		cmd.Printf("Using base schema: %s\n", fallbackSchema)
		schemaPath = fallbackSchema
	default:
		cmd.Println("Using built-in schema")
	}

	return compileSchema("mem://repository-config.schema.json", schemaPath, MarshalRepositoryConfigSchema)
}

// compileSchema compiles a JSON schema from schemaPath, or from the built-in
// marshaller when schemaPath is empty. memURL is the in-memory resource name and
// only needs to be unique per schema.
func compileSchema(memURL, schemaPath string, builtin func() ([]byte, error)) (*jsonschema.Schema, error) {
	var doc any
	var err error
	if schemaPath != "" {
		f, openErr := os.Open(schemaPath)
		if openErr != nil {
			return nil, fmt.Errorf("open schema %s: %w", schemaPath, openErr)
		}
		defer func() { _ = f.Close() }()
		if doc, err = jsonschema.UnmarshalJSON(f); err != nil {
			return nil, fmt.Errorf("parse schema %s: %w", schemaPath, err)
		}
	} else {
		raw, marshalErr := builtin()
		if marshalErr != nil {
			return nil, marshalErr
		}
		if doc, err = jsonschema.UnmarshalJSON(bytes.NewReader(raw)); err != nil {
			return nil, fmt.Errorf("parse built-in schema: %w", err)
		}
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(memURL, doc); err != nil {
		return nil, fmt.Errorf("load schema: %w", err)
	}
	schema, err := compiler.Compile(memURL)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	return schema, nil
}

// resolveOrgSchema returns the override schema path if it exists and resolves to
// a location inside configDir, guarding against symlink escapes.
func resolveOrgSchema(cmd *cobra.Command, configDir string) (string, bool) {
	candidate := filepath.Join(configDir, orgSchemaRelPath)
	if _, err := os.Stat(candidate); err != nil {
		return "", false
	}

	root, err := filepath.Abs(configDir)
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}

	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", false
	}

	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		cmd.PrintErrf("WARNING: ignoring schema override outside config directory: %s\n", candidate)
		return "", false
	}
	return candidate, true
}

// loadYAMLDocument reads and decodes path into a JSON-compatible value ready for
// schema validation. The raw bytes are returned too, so a caller that also needs a
// typed struct can decode from them instead of reading and parsing the file twice.
func loadYAMLDocument(path string) ([]byte, any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("invalid YAML: %w", err)
	}

	instance, err := toJSONValue(raw)
	if err != nil {
		return nil, nil, err
	}
	return data, instance, nil
}

// validateFile parses a single YAML file and validates it against the schema,
// returning one message per violation, each citing the file and JSON path.
func validateFile(path string, schema *jsonschema.Schema) []string {
	_, instance, err := loadYAMLDocument(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", path, err)}
	}
	return validateInstance(path, instance, schema)
}

// validateInstance validates an already-decoded document against the schema.
func validateInstance(path string, instance any, schema *jsonschema.Schema) []string {
	err := schema.Validate(instance)
	if err == nil {
		return nil
	}

	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return []string{fmt.Sprintf("%s: %v", path, err)}
	}

	var messages []string
	collectLeafErrors(path, ve, &messages)
	if len(messages) == 0 {
		// Should not happen, but never swallow a validation failure silently.
		messages = append(messages, fmt.Sprintf("%s: %v", path, err))
	}
	return messages
}

// collectLeafErrors walks the validation error tree and emits one message per
// leaf failure, citing the file and the JSON path of the offending value.
func collectLeafErrors(path string, ve *jsonschema.ValidationError, out *[]string) {
	if len(ve.Causes) == 0 {
		loc := "/" + strings.Join(ve.InstanceLocation, "/")
		if loc == "/" {
			loc = "(root)"
		}
		*out = append(*out, fmt.Sprintf("%s: %s at %s", path, ve.ErrorKind.LocalizedString(printer), loc))
		return
	}
	for _, cause := range ve.Causes {
		collectLeafErrors(path, cause, out)
	}
}

// toJSONValue normalizes a YAML-decoded value into JSON-compatible types so that
// schema validation follows JSON semantics (e.g. numbers, timestamps).
func toJSONValue(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("normalize YAML to JSON: %w", err)
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(b))
}
