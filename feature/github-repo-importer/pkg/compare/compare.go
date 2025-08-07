package compare

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type CompareResult struct {
	OnlyInA   []string `json:"only_in_a"`
	Identical []string `json:"identical"`
	Different []string `json:"different"`
	OnlyInB   []string `json:"only_in_b"`
}

// CompareDirectories compares two directories containing YAML files.
// It returns a CompareResult struct containing the comparison results.
// The comparison is based on the normalized content of the YAML files and hashes.
func CompareDirectories(dirA, dirB string) (CompareResult, error) {
	filesA, err := collectYamlHashes(dirA)
	if err != nil {
		return CompareResult{}, err
	}
	filesB, err := collectYamlHashes(dirB)
	if err != nil {
		return CompareResult{}, err
	}

	result := CompareResult{}

	for relPath, hashA := range filesA {
		hashB, exists := filesB[relPath]
		if !exists {
			result.OnlyInA = append(result.OnlyInA, relPath)
		} else if hashA == hashB {
			result.Identical = append(result.Identical, relPath)
		} else {
			result.Different = append(result.Different, relPath)
		}
	}

	for relPath := range filesB {
		if _, exists := filesA[relPath]; !exists {
			result.OnlyInB = append(result.OnlyInB, relPath)
		}
	}

	return result, nil
}

func collectYamlHashes(root string) (map[string]string, error) {
	hashes := make(map[string]string)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		hash, err := hashNormalizedYamlFile(path)
		if err != nil {
			return fmt.Errorf("error hashing %s: %w", path, err)
		}
		hashes[relPath] = hash
		return nil
	})

	return hashes, err
}

func hashNormalizedYamlFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return "", fmt.Errorf("Can not unmarshal file to yaml: %w\n", err)
	}

	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		root := node.Content[0]
		removeKey(root, "id")
		sortMappingNode(root)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		return "", err
	}
	enc.Close()

	sum := sha256.Sum256(buf.Bytes())
	return fmt.Sprintf("%x", sum), nil
}

func removeKey(node *yaml.Node, target string) {
	if node.Kind == yaml.MappingNode {
		newContent := []*yaml.Node{}
		for i := 0; i < len(node.Content); i += 2 {
			k := node.Content[i]
			v := node.Content[i+1]
			if k.Value != target {
				removeKey(v, target)
				newContent = append(newContent, k, v)
			}
		}
		node.Content = newContent
	} else if node.Kind == yaml.SequenceNode {
		for _, elem := range node.Content {
			removeKey(elem, target)
		}
	}
}

func sortMappingNode(node *yaml.Node) {
	if node.Kind != yaml.MappingNode {
		return
	}

	type kv struct {
		Key, Value *yaml.Node
	}
	var pairs []kv
	for i := 0; i < len(node.Content); i += 2 {
		pairs = append(pairs, kv{
			Key:   node.Content[i],
			Value: node.Content[i+1],
		})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Key.Value < pairs[j].Key.Value
	})

	node.Content = node.Content[:0]
	for _, kv := range pairs {
		node.Content = append(node.Content, kv.Key, kv.Value)
		sortMappingNode(kv.Value)
		sortSequenceIfNeeded(kv.Value)
	}
}

func sortSequenceIfNeeded(node *yaml.Node) {
	if node.Kind != yaml.SequenceNode {
		return
	}
	for _, item := range node.Content {
		if item.Kind == yaml.MappingNode {
			sortMappingNode(item)
		} else if item.Kind == yaml.SequenceNode {
			sortSequenceIfNeeded(item)
		}
	}
}
