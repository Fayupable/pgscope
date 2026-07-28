package history

// redactQueryLiterals replaces string and numeric literals in a raw SQL
// query with "***" before it's persisted to the history store. pg_stat_
// activity reports queries with their actual bound values, which can
// include sensitive data (emails, tokens, etc.) — this is a best-effort
// defense-in-depth measure, not a guarantee of complete redaction.
//
// Handles: plain '...' strings (including doubled ” escapes), E'...'
// escape strings, X'...' hex strings, B'...' bit strings, and $$...$$ /
// $tag$...$tag$ dollar-quoted strings (common in function bodies and DO
// blocks). Parameter placeholders ($1, $2, ...) are left untouched, since
// they never carry a literal value themselves.
func redactQueryLiterals(query string) string {
	runes := []rune(query)
	var b []rune
	i := 0

	for i < len(runes) {
		c := runes[i]

		if c == '$' {
			if end, isPlaceholder := skipPlaceholder(runes, i); isPlaceholder {
				b = append(b, runes[i:end]...)
				i = end
				continue
			}
			if end, matched := skipDollarQuote(runes, i); matched {
				b = append(b, []rune("***")...)
				i = end
				continue
			}
			b = append(b, c)
			i++
			continue
		}

		if isStringPrefix(c) && (i == 0 || !isIdentChar(runes[i-1])) && i+1 < len(runes) && runes[i+1] == '\'' {
			b = append(b, []rune("***")...)
			i = skipSingleQuoteLiteral(runes, i+1)
			continue
		}

		if c == '\'' {
			b = append(b, []rune("***")...)
			i = skipSingleQuoteLiteral(runes, i)
			continue
		}

		if isDigit(c) && (i == 0 || !isIdentChar(runes[i-1])) {
			j := scanNumericLiteral(runes, i)
			b = append(b, []rune("***")...)
			i = j
			continue
		}

		b = append(b, c)
		i++
	}

	return string(b)
}

// skipSingleQuoteLiteral takes the index of an opening ' and returns the
// index just past its matching closing ', treating ” as an escaped quote
// rather than a terminator.
func skipSingleQuoteLiteral(runes []rune, quoteIndex int) int {
	i := quoteIndex + 1
	for i < len(runes) {
		if runes[i] == '\'' {
			if i+1 < len(runes) && runes[i+1] == '\'' {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return i
}

// skipPlaceholder recognizes a parameter placeholder like $1 or $42 — these
// never carry a literal value, so they're passed through unredacted.
func skipPlaceholder(runes []rune, dollarIndex int) (end int, matched bool) {
	j := dollarIndex + 1
	if j >= len(runes) || !isDigit(runes[j]) {
		return dollarIndex, false
	}
	for j < len(runes) && isDigit(runes[j]) {
		j++
	}
	return j, true
}

// skipDollarQuote recognizes a dollar-quoted string starting at dollarIndex
// ($$...$$ or $tag$...$tag$, as used in function bodies and DO blocks) and
// returns the index just past its closing delimiter. If the opening
// delimiter isn't well-formed, matched is false. If no closing delimiter is
// found, the whole remainder is treated as the literal — safer to
// over-redact than under-redact.
func skipDollarQuote(runes []rune, dollarIndex int) (end int, matched bool) {
	j := dollarIndex + 1
	for j < len(runes) && runes[j] != '$' && isIdentChar(runes[j]) {
		j++
	}
	if j >= len(runes) || runes[j] != '$' {
		return dollarIndex, false
	}

	delimiter := runes[dollarIndex : j+1]
	contentStart := j + 1

	closeIdx := indexOfRuneSeq(runes, contentStart, delimiter)
	if closeIdx == -1 {
		return len(runes), true
	}
	return closeIdx + len(delimiter), true
}

func indexOfRuneSeq(runes []rune, from int, seq []rune) int {
	for k := from; k+len(seq) <= len(runes); k++ {
		match := true
		for m := 0; m < len(seq); m++ {
			if runes[k+m] != seq[m] {
				match = false
				break
			}
		}
		if match {
			return k
		}
	}
	return -1
}

func isStringPrefix(r rune) bool {
	switch r {
	case 'e', 'E', 'x', 'X', 'b', 'B':
		return true
	}
	return false
}

// scanNumericLiteral returns the index just past a numeric literal starting
// at start (which must be a digit). Handles PostgreSQL 16's non-decimal
// integer literals (0x1F, 0o17, 0b1010 — hex/octal/binary), underscore
// digit separators (1_000_000), and scientific notation (1e10, 1.5e-10) —
// not just plain decimal digits, so none of those leak past the initial
// masked character.
func scanNumericLiteral(runes []rune, start int) int {
	j := start

	if runes[j] == '0' && j+1 < len(runes) && isRadixPrefix(runes[j+1]) {
		j += 2
		for j < len(runes) && (isHexDigit(runes[j]) || runes[j] == '_') {
			j++
		}
		return j
	}

	for j < len(runes) && (isDigit(runes[j]) || runes[j] == '.' || runes[j] == '_') {
		j++
	}

	if j < len(runes) && (runes[j] == 'e' || runes[j] == 'E') {
		k := j + 1
		if k < len(runes) && (runes[k] == '+' || runes[k] == '-') {
			k++
		}
		if k < len(runes) && isDigit(runes[k]) {
			for k < len(runes) && isDigit(runes[k]) {
				k++
			}
			j = k
		}
	}

	return j
}

func isRadixPrefix(r rune) bool {
	switch r {
	case 'x', 'X', 'o', 'O', 'b', 'B':
		return true
	}
	return false
}

func isHexDigit(r rune) bool {
	return isDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isIdentChar(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || isDigit(r)
}
