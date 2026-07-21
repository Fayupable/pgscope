package postgres

import (
	"reflect"
	"testing"
)

func TestFindComparedColumns(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "single equality comparison",
			text: "SELECT * FROM orders WHERE customer_id = $1",
			want: []string{"customer_id"},
		},
		{
			name: "multiple comparisons across operators",
			text: "SELECT * FROM orders WHERE customer_id = $1 AND total > $2",
			want: []string{"customer_id", "total"},
		},
		{
			name: "column name is lowercased",
			text: "SELECT * FROM orders WHERE CustomerId = $1",
			want: []string{"customerid"},
		},
		{
			name: "literal value instead of placeholder is ignored",
			text: "SELECT * FROM orders WHERE status = 'active'",
			want: []string{},
		},
		{
			name: "no comparisons at all",
			text: "SELECT * FROM orders",
			want: []string{},
		},
		{
			name: "comparison against another column, not a placeholder, is ignored",
			text: "SELECT * FROM orders WHERE customer_id = user_id",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findComparedColumns(tt.text)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findComparedColumns(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestExtractSuspectedColumns(t *testing.T) {
	tests := []struct {
		name       string
		queryTexts []string
		want       []string
	}{
		{
			name: "most frequent column ranks first",
			queryTexts: []string{
				"SELECT * FROM orders WHERE customer_id = $1",
				"SELECT * FROM orders WHERE customer_id = $1",
				"SELECT * FROM orders WHERE status = $1",
			},
			want: []string{"customer_id", "status"},
		},
		{
			name: "tie is broken alphabetically for determinism",
			queryTexts: []string{
				"SELECT * FROM orders WHERE zeta = $1",
				"SELECT * FROM orders WHERE alpha = $1",
			},
			want: []string{"alpha", "zeta"},
		},
		{
			name:       "no matching text yields empty result",
			queryTexts: []string{"SELECT * FROM orders"},
			want:       []string{},
		},
		{
			name: "result is capped at suspectedColumnLimit",
			queryTexts: []string{
				"SELECT * FROM orders WHERE a = $1 AND b = $2 AND c = $3",
			},
			want: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSuspectedColumns(tt.queryTexts)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractSuspectedColumns(%v) = %v, want %v", tt.queryTexts, got, tt.want)
			}
		})
	}
}
