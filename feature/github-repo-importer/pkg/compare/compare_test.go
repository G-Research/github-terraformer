package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashingYamlFiles(t *testing.T) {
	tests := []struct {
		name           string
		pathOfImported string
		pathOfFresh    string
		wantEqual      bool
	}{
		{
			name:           "two identical yaml files - literal copies",
			pathOfImported: "testdata/imported/imported1.yaml",
			pathOfFresh:    "testdata/existing/existing1.yaml",
			wantEqual:      true,
		},
		{
			name:           "two identical yaml files - different order of attributes",
			pathOfImported: "testdata/imported/imported2.yaml",
			pathOfFresh:    "testdata/existing/existing2.yaml",
			wantEqual:      true,
		},
		{
			name:           "two identical yaml files - one with ruleset id other without",
			pathOfImported: "testdata/imported/imported3.yaml",
			pathOfFresh:    "testdata/existing/existing3.yaml",
			wantEqual:      true,
		},
		{
			name:           "two identical yaml files - one with ruleset id other without - different order of attributes",
			pathOfImported: "testdata/imported/imported4.yaml",
			pathOfFresh:    "testdata/existing/existing4.yaml",
			wantEqual:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashOfImported, err := hashNormalizedYamlFile(tt.pathOfImported)
			hashOfFresh, err := hashNormalizedYamlFile(tt.pathOfFresh)

			assert.NoError(t, err)
			if tt.wantEqual {
				assert.Equal(t, hashOfImported, hashOfFresh)
			} else {
				assert.NotEqual(t, hashOfImported, hashOfFresh)
			}
		})
	}
}
