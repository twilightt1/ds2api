package util

import (
	"regexp"
	"strings"
)

var unquotedKeyPattern = regexp.MustCompile(`([{,]\s*)([a-zA-Z_][a-zA-Z0-9_]*)\s*:`)

// fallback pattern for shallow objects; scanner-based repair runs first.
var missingArrayBracketsPattern = regexp.MustCompile(`(:\s*)(\{(?:[^{}]|\{[^{}]*\})*\}(?:\s*,\s*\{(?:[^{}]|\{[^{}]*\})*\})+)`)

func repairInvalidJSONBackslashes(s string) string {
	return repairInvalidJSONBackslashesWithPathContext(s)
}

func repairInvalidJSONBackslashesWithPathContext(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var out strings.Builder
	out.Grow(len(s) + 10)

	runes := []rune(s)
	pathKeyContext := buildPathKeyStringMask(runes)
	inString := false
	escaped := false
	stringStart := -1

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '"' && !escaped {
			inString = !inString
			if inString {
				stringStart = i
			} else {
				stringStart = -1
			}
			out.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && inString {
			if i+1 < len(runes) {
				next := runes[i+1]
				if next == 'u' {
					if i+5 < len(runes) && isHex4(runes[i+2:i+6]) {
						out.WriteRune('\\')
						out.WriteRune('u')
						for _, hx := range runes[i+2 : i+6] {
							out.WriteRune(hx)
						}
						i += 5
						escaped = false
						continue
					}
				} else if shouldKeepEscape(next, pathKeyContext[stringStart]) {
					out.WriteRune('\\')
					out.WriteRune(next)
					i++
					escaped = false
					continue
				}
			}
			out.WriteString("\\\\")
			escaped = false
			continue
		}
		out.WriteRune(r)
		escaped = r == '\\' && !escaped
		if r != '\\' {
			escaped = false
		}
	}
	return out.String()
}

func shouldKeepEscape(next rune, inPathContext bool) bool {
	switch next {
	case '"', '\\', '/', 'b', 'f':
		return true
	case 'n', 'r', 't':
		return !inPathContext
	case 'u':
		return true
	default:
		return false
	}
}

func buildPathKeyStringMask(runes []rune) map[int]bool {
	mask := map[int]bool{}
	inString := false
	escaped := false
	stringStart := -1
	var lastKey string

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if !inString {
			if r == '"' {
				inString = true
				stringStart = i
			}
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r != '"' {
			continue
		}

		value := string(runes[stringStart+1 : i])
		j := i + 1
		for j < len(runes) && (runes[j] == ' ' || runes[j] == '\n' || runes[j] == '\r' || runes[j] == '\t') {
			j++
		}
		if j < len(runes) && runes[j] == ':' {
			lastKey = strings.ToLower(strings.TrimSpace(value))
		} else if isPathLikeKey(lastKey) {
			mask[stringStart] = true
		}

		inString = false
		stringStart = -1
	}
	return mask
}

func RepairLooseJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = unquotedKeyPattern.ReplaceAllString(s, `$1"$2":`)
	s = repairMissingArrayBracketsByScanner(s)
	return missingArrayBracketsPattern.ReplaceAllString(s, `$1[$2]`)
}

func repairMissingArrayBracketsByScanner(s string) string {
	const maxScanLen = 200_000
	if len(s) == 0 || len(s) > maxScanLen {
		return s
	}

	var out strings.Builder
	out.Grow(len(s) + 8)
	i := 0
	for i < len(s) {
		if s[i] != ':' {
			out.WriteByte(s[i])
			i++
			continue
		}
		out.WriteByte(':')
		i++
		for i < len(s) && isJSONWhitespace(s[i]) {
			out.WriteByte(s[i])
			i++
		}
		if i >= len(s) || s[i] != '{' {
			continue
		}

		start := i
		end := scanJSONObjectEnd(s, start)
		if end < 0 {
			out.WriteString(s[start:])
			break
		}
		cursor := end
		next := skipJSONWhitespace(s, cursor)
		if next >= len(s) || s[next] != ',' {
			out.WriteString(s[start:end])
			i = end
			continue
		}

		seqEnd := end
		hasMultiple := false
		for {
			comma := skipJSONWhitespace(s, seqEnd)
			if comma >= len(s) || s[comma] != ',' {
				break
			}
			objStart := skipJSONWhitespace(s, comma+1)
			if objStart >= len(s) || s[objStart] != '{' {
				break
			}
			objEnd := scanJSONObjectEnd(s, objStart)
			if objEnd < 0 {
				break
			}
			hasMultiple = true
			seqEnd = objEnd
		}
		if !hasMultiple {
			out.WriteString(s[start:end])
			i = end
			continue
		}

		out.WriteByte('[')
		out.WriteString(s[start:seqEnd])
		out.WriteByte(']')
		i = seqEnd
	}
	return out.String()
}

func scanJSONObjectEnd(s string, start int) int {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			continue
		}
		if c == '{' {
			depth++
			continue
		}
		if c == '}' {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

func skipJSONWhitespace(s string, i int) int {
	for i < len(s) && isJSONWhitespace(s[i]) {
		i++
	}
	return i
}

func isJSONWhitespace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r' || b == '\t'
}

func isHex4(seq []rune) bool {
	if len(seq) != 4 {
		return false
	}
	for _, r := range seq {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
