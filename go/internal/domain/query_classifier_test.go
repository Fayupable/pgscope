package domain

import "testing"

func TestClassifyOperation(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  QueryOperation
	}{
		{name: "select", query: "SELECT * FROM orders", want: QueryOperationSelect},
		{name: "insert", query: "INSERT INTO orders (id) VALUES ($1)", want: QueryOperationInsert},
		{name: "update", query: "UPDATE orders SET status = $1", want: QueryOperationUpdate},
		{name: "delete", query: "DELETE FROM orders WHERE id = $1", want: QueryOperationDelete},
		{name: "merge", query: "MERGE INTO orders USING staging ON true", want: QueryOperationMerge},
		{name: "create is DDL", query: "CREATE TABLE orders (id int)", want: QueryOperationDDL},
		{name: "alter is DDL", query: "ALTER TABLE orders ADD COLUMN x int", want: QueryOperationDDL},
		{name: "drop is DDL", query: "DROP TABLE orders", want: QueryOperationDDL},
		{name: "truncate is DDL", query: "TRUNCATE orders", want: QueryOperationDDL},
		{name: "vacuum", query: "VACUUM orders", want: QueryOperationVacuum},
		{name: "analyze", query: "ANALYZE orders", want: QueryOperationVacuum},
		{name: "lowercase keyword still matches", query: "select * from orders", want: QueryOperationSelect},
		{name: "leading whitespace is skipped", query: "   \n\tSELECT 1", want: QueryOperationSelect},
		{name: "unrecognized keyword falls back to other", query: "EXPLAIN SELECT 1", want: QueryOperationOther},
		{name: "empty query is unknown", query: "", want: QueryOperationUnknown},
		{name: "whitespace-only query is unknown", query: "   ", want: QueryOperationUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyOperation(tt.query); got != tt.want {
				t.Errorf("ClassifyOperation(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}
