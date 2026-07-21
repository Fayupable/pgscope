package domain

// ClassifyOperation derives a QueryOperation from a raw SQL statement's
// leading keyword. It is intentionally simple — a best-effort classification
// for display purposes, not a SQL parser. Shared across every engine adapter
// since SQL keywords don't vary between them.
func ClassifyOperation(query string) QueryOperation {
	keyword := leadingKeyword(query)
	switch keyword {
	case "SELECT":
		return QueryOperationSelect
	case "INSERT":
		return QueryOperationInsert
	case "UPDATE":
		return QueryOperationUpdate
	case "DELETE":
		return QueryOperationDelete
	case "MERGE":
		return QueryOperationMerge
	case "CREATE", "ALTER", "DROP", "TRUNCATE":
		return QueryOperationDDL
	case "VACUUM", "ANALYZE":
		return QueryOperationVacuum
	case "":
		return QueryOperationUnknown
	default:
		return QueryOperationOther
	}
}

func leadingKeyword(query string) string {
	start := 0
	for start < len(query) && isSpace(query[start]) {
		start++
	}

	end := start
	for end < len(query) && !isSpace(query[end]) {
		end++
	}

	return toUpper(query[start:end])
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n'
}

func toUpper(word string) string {
	upper := make([]byte, len(word))
	for i := 0; i < len(word); i++ {
		c := word[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		upper[i] = c
	}
	return string(upper)
}
