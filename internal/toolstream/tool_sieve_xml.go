package toolstream

import (
	"ds2api/internal/toolcall"
	"strings"
)

// consumeXMLToolCapture tries to extract complete XML tool call blocks from captured text.
func consumeXMLToolCapture(captured string, toolNames []string) (prefix string, calls []toolcall.ParsedToolCall, suffix string, ready bool) {
	lower := strings.ToLower(captured)
	anyOpenFound := false
	type candidate struct {
		start  int
		prefix string
		calls  []toolcall.ParsedToolCall
		suffix string
	}
	type rejectedBlock struct {
		start  int
		prefix string
		suffix string
	}
	var best *candidate
	var rejected *rejectedBlock

	// Scan every wrapper occurrence. Prose can mention a wrapper tag before the
	// actual tool block, including the same variant as the real block.
	for _, pair := range xmlToolCallTagPairs {
		searchFrom := 0
		for searchFrom < len(lower) {
			openIdx := findXMLOpenOutsideCDATA(captured, pair.open, searchFrom)
			if openIdx < 0 {
				break
			}
			// Find the matching closing tag outside CDATA. Long write-file tool
			// calls often contain XML examples in CDATA, including </tool_calls>.
			closeIdx := findMatchingXMLToolWrapperClose(captured, pair.open, pair.close, openIdx)
			if closeIdx < 0 {
				anyOpenFound = true
				searchFrom = openIdx + len(pair.open)
				continue
			}
			closeEnd := closeIdx + len(pair.close)

			xmlBlock := captured[openIdx:closeEnd]
			prefixPart := captured[:openIdx]
			suffixPart := captured[closeEnd:]
			parsed := toolcall.ParseToolCalls(xmlBlock, toolNames)
			if len(parsed) > 0 {
				prefixPart, suffixPart = trimWrappingJSONFence(prefixPart, suffixPart)
				if best == nil || openIdx < best.start {
					best = &candidate{start: openIdx, prefix: prefixPart, calls: parsed, suffix: suffixPart}
				}
				break
			}
			if rejected == nil || openIdx < rejected.start {
				rejected = &rejectedBlock{start: openIdx, prefix: prefixPart + xmlBlock, suffix: suffixPart}
			}
			searchFrom = openIdx + len(pair.open)
		}
	}
	if best != nil {
		return best.prefix, best.calls, best.suffix, true
	}
	if anyOpenFound {
		// At least one opening tag was found but none had a matching close tag.
		// Keep buffering until a closing tag arrives.
		return "", nil, "", false
	}
	if rejected != nil {
		// If this block failed to become a tool call, pass it through as text.
		return rejected.prefix, nil, rejected.suffix, true
	}
	if !containsAnyToolCallWrapper(lower) {
		invokeIdx, dsml := firstInvokeIndex(lower)
		closeTag := "</tool_calls>"
		openWrapper := "<tool_calls>"
		if dsml {
			closeTag = "</|dsml|tool_calls>"
			openWrapper = "<|DSML|tool_calls>"
		}
		closeIdx := findXMLCloseOutsideCDATA(captured, closeTag, invokeIdx)
		if invokeIdx >= 0 && closeIdx > invokeIdx {
			closeEnd := closeIdx + len(closeTag)
			xmlBlock := openWrapper + captured[invokeIdx:closeIdx] + closeTag
			prefixPart := captured[:invokeIdx]
			suffixPart := captured[closeEnd:]
			parsed := toolcall.ParseToolCalls(xmlBlock, toolNames)
			if len(parsed) > 0 {
				prefixPart, suffixPart = trimWrappingJSONFence(prefixPart, suffixPart)
				return prefixPart, parsed, suffixPart, true
			}
			return prefixPart + captured[invokeIdx:closeEnd], nil, suffixPart, true
		}
	}
	return "", nil, "", false
}

// hasOpenXMLToolTag returns true if captured text contains an XML tool opening tag
// whose SPECIFIC closing tag has not appeared yet.
func hasOpenXMLToolTag(captured string) bool {
	for _, pair := range xmlToolCallTagPairs {
		openIdx := findXMLOpenOutsideCDATA(captured, pair.open, 0)
		if openIdx >= 0 {
			if findMatchingXMLToolWrapperClose(captured, pair.open, pair.close, openIdx) < 0 {
				return true
			}
		}
	}
	return false
}

func shouldKeepBareInvokeCapture(captured string) bool {
	lower := strings.ToLower(captured)
	invokeIdx, dsml := firstInvokeIndex(lower)
	if invokeIdx < 0 || containsAnyToolCallWrapper(lower) {
		return false
	}
	invokeOpenLen := len("<invoke")
	parameterOpen := "<parameter"
	if dsml {
		invokeOpenLen = len("<|dsml|invoke")
		parameterOpen = "<|dsml|parameter"
	}
	if dsml && strings.HasPrefix(lower[invokeIdx:], "<|dsml invoke") {
		invokeOpenLen = len("<|dsml invoke")
		parameterOpen = "<|dsml parameter"
	}
	if dsml && strings.HasPrefix(lower[invokeIdx:], "<dsml|invoke") {
		invokeOpenLen = len("<dsml|invoke")
		parameterOpen = "<dsml|parameter"
	}
	if dsml && strings.HasPrefix(lower[invokeIdx:], "<dsml invoke") {
		invokeOpenLen = len("<dsml invoke")
		parameterOpen = "<dsml parameter"
	}
	if findAnyXMLCloseOutsideCDATA(captured, possibleWrapperCloseTags(dsml), invokeIdx) > invokeIdx {
		return true
	}

	startEnd := findXMLTagEnd(captured, invokeIdx+invokeOpenLen)
	if startEnd < 0 {
		return true
	}
	body := captured[startEnd+1:]
	trimmedBody := strings.TrimLeft(body, " \t\r\n")
	if trimmedBody == "" {
		return true
	}

	invokeCloseIdx := findAnyXMLCloseOutsideCDATA(captured, possibleInvokeCloseTags(dsml), startEnd+1)
	if invokeCloseIdx >= 0 {
		afterClose := captured[invokeCloseIdx:]
		for _, closeTag := range possibleInvokeCloseTags(dsml) {
			if strings.HasPrefix(strings.ToLower(afterClose), closeTag) {
				afterClose = afterClose[len(closeTag):]
				break
			}
		}
		return strings.TrimSpace(afterClose) == ""
	}

	trimmedLower := strings.ToLower(trimmedBody)
	return strings.HasPrefix(trimmedLower, parameterOpen) ||
		strings.HasPrefix(trimmedLower, "{") ||
		strings.HasPrefix(trimmedLower, "[")
}

func containsAnyToolCallWrapper(lower string) bool {
	return strings.Contains(lower, "<tool_calls") ||
		strings.Contains(lower, "<|dsml|tool_calls") ||
		strings.Contains(lower, "<|dsml tool_calls") ||
		strings.Contains(lower, "<dsml|tool_calls") ||
		strings.Contains(lower, "<dsml tool_calls") ||
		strings.Contains(lower, "<｜tool_calls") ||
		strings.Contains(lower, "<|tool_calls")
}

func possibleWrapperCloseTags(dsml bool) []string {
	if !dsml {
		return []string{"</tool_calls>"}
	}
	return []string{"</|dsml|tool_calls>", "</|dsml tool_calls>", "</dsml|tool_calls>", "</dsml tool_calls>", "</｜tool_calls>", "</|tool_calls>"}
}

func possibleInvokeCloseTags(dsml bool) []string {
	if !dsml {
		return []string{"</invoke>"}
	}
	return []string{"</|dsml|invoke>", "</|dsml invoke>", "</dsml|invoke>", "</dsml invoke>", "</｜invoke>", "</|invoke>"}
}

func findAnyXMLCloseOutsideCDATA(s string, closeTags []string, start int) int {
	best := -1
	for _, closeTag := range closeTags {
		idx := findXMLCloseOutsideCDATA(s, closeTag, start)
		if idx >= 0 && (best < 0 || idx < best) {
			best = idx
		}
	}
	return best
}

func firstInvokeIndex(lower string) (int, bool) {
	xmlIdx := strings.Index(lower, "<invoke")
	// Check all DSML-like invoke prefixes.
	dsmlPrefixes := []string{"<|dsml|invoke", "<|dsml invoke", "<dsml|invoke", "<dsml invoke", "<｜invoke", "<|invoke"}
	dsmlIdx := -1
	for _, prefix := range dsmlPrefixes {
		idx := strings.Index(lower, prefix)
		if idx >= 0 && (dsmlIdx < 0 || idx < dsmlIdx) {
			dsmlIdx = idx
		}
	}
	switch {
	case xmlIdx < 0:
		return dsmlIdx, dsmlIdx >= 0
	case dsmlIdx < 0:
		return xmlIdx, false
	case dsmlIdx < xmlIdx:
		return dsmlIdx, true
	default:
		return xmlIdx, false
	}
}

// findPartialXMLToolTagStart checks if the string ends with a partial canonical
// XML wrapper tag (e.g., "<too") and returns the position of the '<'.
func findPartialXMLToolTagStart(s string) int {
	lastLT := strings.LastIndex(s, "<")
	if lastLT < 0 {
		return -1
	}
	tail := s[lastLT:]
	// If there's a '>' in the tail, the tag is closed — not partial.
	if strings.Contains(tail, ">") {
		return -1
	}
	lowerTail := strings.ToLower(tail)
	// Check if the tail is a prefix of any known XML tool tag.
	for _, tag := range xmlToolCallOpeningTags {
		tagWithLT := tag
		if !strings.HasPrefix(tagWithLT, "<") {
			tagWithLT = "<" + tagWithLT
		}
		if strings.HasPrefix(tagWithLT, lowerTail) {
			return lastLT
		}
	}
	return -1
}
