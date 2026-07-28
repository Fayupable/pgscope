package history

import "testing"

func TestRedactQueryLiterals(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "no literals passes through unchanged",
			query: "SELECT * FROM orders WHERE status = status",
			want:  "SELECT * FROM orders WHERE status = status",
		},
		{
			name:  "a simple string literal is masked",
			query: "SELECT * FROM users WHERE email = 'ali@example.com'",
			want:  "SELECT * FROM users WHERE email = ***",
		},
		{
			name:  "a standalone numeric literal is masked",
			query: "SELECT * FROM orders WHERE id = 12345",
			want:  "SELECT * FROM orders WHERE id = ***",
		},
		{
			name:  "a decimal numeric literal is masked as one token",
			query: "SELECT * FROM payments WHERE amount = 199.99",
			want:  "SELECT * FROM payments WHERE amount = ***",
		},
		{
			name:  "multiple literals in the same query are all masked",
			query: "SELECT * FROM orders WHERE user_id = 42 AND status = 'pending'",
			want:  "SELECT * FROM orders WHERE user_id = *** AND status = ***",
		},
		{
			name:  "an escaped single quote inside a string literal does not truncate the mask",
			query: "SELECT * FROM users WHERE name = 'O''Brien'",
			want:  "SELECT * FROM users WHERE name = ***",
		},
		{
			name:  "digits that are part of an identifier are left alone",
			query: "SELECT column1, table2 FROM schema3.orders",
			want:  "SELECT column1, table2 FROM schema3.orders",
		},
		{
			name:  "an empty string literal is still masked",
			query: "SELECT * FROM users WHERE middle_name = ''",
			want:  "SELECT * FROM users WHERE middle_name = ***",
		},
		{
			name:  "empty query stays empty",
			query: "",
			want:  "",
		},
		{
			name:  "a query with only a numeric literal is fully masked",
			query: "123",
			want:  "***",
		},
		{
			name:  "a hexadecimal integer literal is masked in full",
			query: "SELECT * FROM flags WHERE mask = 0x1F",
			want:  "SELECT * FROM flags WHERE mask = ***",
		},
		{
			name:  "a binary integer literal is masked in full",
			query: "SELECT * FROM flags WHERE mask = 0b101010",
			want:  "SELECT * FROM flags WHERE mask = ***",
		},
		{
			name:  "an octal integer literal is masked in full",
			query: "SELECT * FROM flags WHERE mask = 0o17",
			want:  "SELECT * FROM flags WHERE mask = ***",
		},
		{
			name:  "an underscore-separated numeric literal is masked in full",
			query: "SELECT * FROM accounts WHERE balance = 1_000_000",
			want:  "SELECT * FROM accounts WHERE balance = ***",
		},
		{
			name:  "scientific notation with a positive implicit exponent is masked in full",
			query: "SELECT * FROM measurements WHERE value = 1e10",
			want:  "SELECT * FROM measurements WHERE value = ***",
		},
		{
			name:  "scientific notation with a negative exponent is masked in full",
			query: "SELECT * FROM measurements WHERE value = 1.5e-10",
			want:  "SELECT * FROM measurements WHERE value = ***",
		},
		{
			name:  "scientific notation with an explicit positive exponent sign is masked in full",
			query: "SELECT * FROM measurements WHERE value = 2E+5",
			want:  "SELECT * FROM measurements WHERE value = ***",
		},
		{
			name:  "a lone 'e' immediately after a number without digits is not treated as an exponent",
			query: "SELECT * FROM t WHERE x = 5e",
			want:  "SELECT * FROM t WHERE x = ***e",
		},
		{
			name:  "an E-string is masked",
			query: "SELECT * FROM logs WHERE message = E'line1\\nline2'",
			want:  "SELECT * FROM logs WHERE message = ***",
		},
		{
			name:  "a hex literal is masked",
			query: "SELECT * FROM files WHERE checksum = X'DEADBEEF'",
			want:  "SELECT * FROM files WHERE checksum = ***",
		},
		{
			name:  "a bit literal is masked",
			query: "SELECT * FROM flags WHERE mask = B'1010'",
			want:  "SELECT * FROM flags WHERE mask = ***",
		},
		{
			name:  "an empty dollar-quoted string is masked",
			query: "DO $$ SELECT 1 $$",
			want:  "DO ***",
		},
		{
			name:  "a tagged dollar-quoted string is masked",
			query: "CREATE FUNCTION f() RETURNS void AS $body$ SELECT 'secret@example.com'; $body$ LANGUAGE sql",
			want:  "CREATE FUNCTION f() RETURNS void AS *** LANGUAGE sql",
		},
		{
			name:  "an unterminated dollar-quote redacts to the end of the string rather than leaking content",
			query: "DO $$ SELECT 'oops, no closing tag",
			want:  "DO ***",
		},
		{
			name:  "a single-digit parameter placeholder is left untouched",
			query: "SELECT * FROM users WHERE id = $1",
			want:  "SELECT * FROM users WHERE id = $1",
		},
		{
			name:  "a multi-digit parameter placeholder is left untouched",
			query: "SELECT * FROM users WHERE id = $1 AND org_id = $42",
			want:  "SELECT * FROM users WHERE id = $1 AND org_id = $42",
		},
		{
			name:  "an ordinary identifier prefixed with e/x/b is not mistaken for a string prefix",
			query: "SELECT extra, x_value, b_flag FROM widgets",
			want:  "SELECT extra, x_value, b_flag FROM widgets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactQueryLiterals(tt.query); got != tt.want {
				t.Errorf("redactQueryLiterals(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}
