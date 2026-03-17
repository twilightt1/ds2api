package util

import (
	"encoding/json"
	"strings"
)

func parseToolCallsPayload(payload string) []ParsedToolCall {
	var decoded any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		repaired := repairInvalidJSONBackslashesWithPathContext(payload)
		repaired = RepairLooseJSON(repaired)
		if err := json.Unmarshal([]byte(repaired), &decoded); err != nil {
			return nil
		}
	}

	switch v := decoded.(type) {
	case map[string]any:
		if tc, ok := v["tool_calls"]; ok {
			return parseToolCallList(tc)
		}
		if parsed, ok := parseToolCallItem(v); ok {
			return []ParsedToolCall{parsed}
		}
	case []any:
		return parseToolCallList(v)
	}
	return nil
}

func parseToolCallList(v any) []ParsedToolCall {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]ParsedToolCall, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if tc, ok := parseToolCallItem(m); ok {
			out = append(out, tc)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseToolCallItem(m map[string]any) (ParsedToolCall, bool) {
	name, _ := m["name"].(string)
	inputRaw, hasInput := m["input"]

	if fn, ok := m["function"].(map[string]any); ok {
		if name == "" {
			name, _ = fn["name"].(string)
		}
		if !hasInput {
			if v, ok := fn["arguments"]; ok {
				inputRaw = v
				hasInput = true
			}
		}
	}
	if !hasInput {
		for _, key := range []string{"arguments", "args", "parameters", "params"} {
			if v, ok := m[key]; ok {
				inputRaw = v
				hasInput = true
				break
			}
		}
	}
	if strings.TrimSpace(name) == "" {
		return ParsedToolCall{}, false
	}
	return ParsedToolCall{
		Name:  strings.TrimSpace(name),
		Input: parseToolCallInput(inputRaw),
	}, true
}

func parseToolCallInput(v any) map[string]any {
	switch x := v.(type) {
	case nil:
		return map[string]any{}
	case map[string]any:
		return x
	case string:
		raw := strings.TrimSpace(x)
		if raw == "" {
			return map[string]any{}
		}

		if parsed := decodeJSONObject(raw); parsed != nil {
			if hasSuspiciousPathControlChars(parsed) {
				repaired := repairInvalidJSONBackslashesWithPathContext(raw)
				if repaired != raw {
					if reparsed := decodeJSONObject(repaired); reparsed != nil {
						return reparsed
					}
				}
			}
			return parsed
		}

		repaired := repairInvalidJSONBackslashesWithPathContext(raw)
		if repaired != raw {
			if reparsed := decodeJSONObject(repaired); reparsed != nil {
				return reparsed
			}
		}

		repairedLoose := RepairLooseJSON(raw)
		if repairedLoose != raw {
			if reparsed := decodeJSONObject(repairedLoose); reparsed != nil {
				return reparsed
			}
		}
		return map[string]any{"_raw": raw}
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return map[string]any{}
		}
		var parsed map[string]any
		if err := json.Unmarshal(b, &parsed); err == nil && parsed != nil {
			return parsed
		}
		return map[string]any{}
	}
}

func decodeJSONObject(raw string) map[string]any {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil && parsed != nil {
		return parsed
	}
	return nil
}

func hasSuspiciousPathControlChars(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for key, value := range x {
			if isPathLikeKey(key) && hasControlCharsInString(value) {
				return true
			}
			if hasSuspiciousPathControlChars(value) {
				return true
			}
		}
	case []any:
		for _, item := range x {
			if hasSuspiciousPathControlChars(item) {
				return true
			}
		}
	}
	return false
}

func isPathLikeKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" {
		return false
	}
	for _, candidate := range []string{"path", "file", "filepath", "filename", "cwd", "dir", "directory"} {
		if lower == candidate || strings.HasSuffix(lower, "_"+candidate) || strings.HasSuffix(lower, candidate+"_path") {
			return true
		}
	}
	return false
}

func hasControlCharsInString(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.ContainsAny(s, "\n\r\t")
}
