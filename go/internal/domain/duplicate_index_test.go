package domain

import "testing"

func TestDetectDuplicateIndexes(t *testing.T) {
	tests := []struct {
		name    string
		indexes []IndexInfo
		want    []struct{ redundant, covering string }
	}{
		{
			name: "identical column lists flags the alphabetically later index as redundant",
			indexes: []IndexInfo{
				{Table: "orders", Name: "idx_a", AccessMethod: "btree", Columns: []string{"user_id"}},
				{Table: "orders", Name: "idx_b", AccessMethod: "btree", Columns: []string{"user_id"}},
			},
			want: []struct{ redundant, covering string }{
				{redundant: "idx_b", covering: "idx_a"},
			},
		},
		{
			name: "a leading-column subset is redundant against a superset index",
			indexes: []IndexInfo{
				{Table: "orders", Name: "idx_user", AccessMethod: "btree", Columns: []string{"user_id"}},
				{Table: "orders", Name: "idx_user_status", AccessMethod: "btree", Columns: []string{"user_id", "status"}},
			},
			want: []struct{ redundant, covering string }{
				{redundant: "idx_user", covering: "idx_user_status"},
			},
		},
		{
			name: "same columns in a different order are not considered redundant",
			indexes: []IndexInfo{
				{Table: "orders", Name: "idx_a", AccessMethod: "btree", Columns: []string{"user_id", "status"}},
				{Table: "orders", Name: "idx_b", AccessMethod: "btree", Columns: []string{"status", "user_id"}},
			},
			want: nil,
		},
		{
			name: "a column subset that isn't a leading prefix is not redundant",
			indexes: []IndexInfo{
				{Table: "orders", Name: "idx_status_only", AccessMethod: "btree", Columns: []string{"status"}},
				{Table: "orders", Name: "idx_user_status", AccessMethod: "btree", Columns: []string{"user_id", "status"}},
			},
			want: nil,
		},
		{
			name: "primary key index is never flagged as redundant, even when another index covers it",
			indexes: []IndexInfo{
				{Table: "orders", Name: "orders_pkey", AccessMethod: "btree", Columns: []string{"id"}, Primary: true},
				{Table: "orders", Name: "idx_id_created", AccessMethod: "btree", Columns: []string{"id", "created_at"}},
			},
			want: nil,
		},
		{
			name: "unique index is never flagged as redundant, even when another index covers it",
			indexes: []IndexInfo{
				{Table: "users", Name: "users_email_key", AccessMethod: "btree", Columns: []string{"email"}, Unique: true},
				{Table: "users", Name: "idx_email_created", AccessMethod: "btree", Columns: []string{"email", "created_at"}},
			},
			want: nil,
		},
		{
			name: "a plain index covered by a primary key is flagged, and the primary key is the keeper",
			indexes: []IndexInfo{
				{Table: "orders", Name: "idx_id_dup", AccessMethod: "btree", Columns: []string{"id"}},
				{Table: "orders", Name: "orders_pkey", AccessMethod: "btree", Columns: []string{"id"}, Primary: true},
			},
			want: []struct{ redundant, covering string }{
				{redundant: "idx_id_dup", covering: "orders_pkey"},
			},
		},
		{
			name: "different access methods are never considered duplicates",
			indexes: []IndexInfo{
				{Table: "orders", Name: "idx_btree", AccessMethod: "btree", Columns: []string{"user_id"}},
				{Table: "orders", Name: "idx_hash", AccessMethod: "hash", Columns: []string{"user_id"}},
			},
			want: nil,
		},
		{
			name: "same columns but different partial predicates are never considered duplicates",
			indexes: []IndexInfo{
				{Table: "orders", Name: "idx_active", AccessMethod: "btree", Columns: []string{"user_id"}, Partial: true, Predicate: "status = 'active'"},
				{Table: "orders", Name: "idx_cancelled", AccessMethod: "btree", Columns: []string{"user_id"}, Partial: true, Predicate: "status = 'cancelled'"},
			},
			want: nil,
		},
		{
			name: "identical indexes on different tables are evaluated independently",
			indexes: []IndexInfo{
				{Table: "orders", Name: "idx_a", AccessMethod: "btree", Columns: []string{"user_id"}},
				{Table: "orders", Name: "idx_b", AccessMethod: "btree", Columns: []string{"user_id"}},
				{Table: "payments", Name: "idx_c", AccessMethod: "btree", Columns: []string{"user_id"}},
			},
			want: []struct{ redundant, covering string }{
				{redundant: "idx_b", covering: "idx_a"},
			},
		},
		{
			name: "a lone index with no siblings is never flagged",
			indexes: []IndexInfo{
				{Table: "orders", Name: "idx_only", AccessMethod: "btree", Columns: []string{"user_id"}},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectDuplicateIndexes(tt.indexes)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectDuplicateIndexes() returned %d duplicates, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, w := range tt.want {
				if got[i].RedundantIndex != w.redundant || got[i].CoveringIndex != w.covering {
					t.Errorf("duplicate[%d] = (redundant=%q, covering=%q), want (redundant=%q, covering=%q)",
						i, got[i].RedundantIndex, got[i].CoveringIndex, w.redundant, w.covering)
				}
			}
		})
	}
}

func TestDetectDuplicateIndexes_NeverReturnsNilSlice(t *testing.T) {
	got := DetectDuplicateIndexes(nil)
	if got == nil {
		t.Fatal("DetectDuplicateIndexes(nil) returned a nil slice, want an empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("DetectDuplicateIndexes(nil) = %v, want empty", got)
	}
}

func TestDetectDuplicateIndexes_Explanation(t *testing.T) {
	t.Run("mentions the primary key when the covering index backs it", func(t *testing.T) {
		got := DetectDuplicateIndexes([]IndexInfo{
			{Table: "orders", Name: "idx_id_dup", AccessMethod: "btree", Columns: []string{"id"}},
			{Table: "orders", Name: "orders_pkey", AccessMethod: "btree", Columns: []string{"id"}, Primary: true},
		})

		if len(got) != 1 {
			t.Fatalf("expected exactly 1 duplicate, got %d", len(got))
		}
		if want := "backs the primary key"; !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	})

	t.Run("mentions the unique constraint when the covering index enforces one", func(t *testing.T) {
		got := DetectDuplicateIndexes([]IndexInfo{
			{Table: "users", Name: "idx_email_dup", AccessMethod: "btree", Columns: []string{"email"}},
			{Table: "users", Name: "users_email_key", AccessMethod: "btree", Columns: []string{"email"}, Unique: true},
		})

		if len(got) != 1 {
			t.Fatalf("expected exactly 1 duplicate, got %d", len(got))
		}
		if want := "enforces a unique constraint"; !contains(got[0].Explanation, want) {
			t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
		}
	})

	t.Run("names both indexes and their columns when neither is primary or unique", func(t *testing.T) {
		got := DetectDuplicateIndexes([]IndexInfo{
			{Table: "orders", Name: "idx_user", AccessMethod: "btree", Columns: []string{"user_id"}},
			{Table: "orders", Name: "idx_user_status", AccessMethod: "btree", Columns: []string{"user_id", "status"}},
		})

		if len(got) != 1 {
			t.Fatalf("expected exactly 1 duplicate, got %d", len(got))
		}
		for _, want := range []string{"idx_user", "idx_user_status", "user_id", "status"} {
			if !contains(got[0].Explanation, want) {
				t.Errorf("Explanation = %q, want it to contain %q", got[0].Explanation, want)
			}
		}
	})
}
