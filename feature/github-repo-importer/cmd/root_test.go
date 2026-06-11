package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeAnnotation(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no special characters",
			in:   "repository not found",
			want: "repository not found",
		},
		{
			name: "percent is escaped",
			in:   "50% off",
			want: "50%25 off",
		},
		{
			name: "newline is escaped",
			in:   "line1\nline2",
			want: "line1%0Aline2",
		},
		{
			name: "carriage return is escaped",
			in:   "line1\rline2",
			want: "line1%0Dline2",
		},
		{
			name: "combined special characters",
			in:   "50% off\r\nnext line",
			want: "50%25 off%0D%0Anext line",
		},
		{
			name: "percent is escaped before its replacement is rescanned",
			in:   "%0A",
			want: "%250A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escapeAnnotation(tt.in))
		})
	}
}
