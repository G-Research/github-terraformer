package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v67/github"
	"github.com/gr-oss-devops/github-repo-importer/pkg/file"
)

func FetchCustomProperties(client *github.Client, owner, repo string, dumpManager *file.DumpManager) (map[string]string, error) {
	customPropertyValues, r, err := client.Repositories.GetAllCustomPropertyValues(context.Background(), owner, repo)
	if err != nil {
		if r != nil && r.StatusCode == http.StatusForbidden {
			fmt.Printf("skipping custom properties due to insufficient permissions: %v\n", err)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get custom properties: %w", err)
	}

	if err := dumpManager.WriteJSONFile("custom_properties.json", customPropertyValues); err != nil {
		fmt.Printf("failed to write custom_properties.json: %v\n", err)
	}

	result := make(map[string]string)
	for _, prop := range customPropertyValues {
		if prop.Value == nil {
			continue
		}
		if v, ok := prop.Value.(string); ok {
			result[prop.PropertyName] = v
		}
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result, nil
}
