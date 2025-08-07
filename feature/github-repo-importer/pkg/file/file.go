package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type DumpManager struct {
	base string
}

func NewDumpManager(base string) (*DumpManager, error) {
	b := filepath.Join("./dumps", base)
	if err := os.MkdirAll(b, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create base directories: %w", err)
	}

	return &DumpManager{base: b}, nil
}

func (dm *DumpManager) WriteJSONFile(fileName string, data interface{}) error {
	filePath := filepath.Join(dm.base, fileName)
	fmt.Printf("Creating JSON file: %s\n", filePath)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal json: %v", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0o644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", filePath, err)
	}

	return nil
}
