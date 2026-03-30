package sse

import "strings"

func filterLeakedContentFilterParts(parts []ContentPart) []ContentPart {
	if len(parts) == 0 {
		return parts
	}
	out := make([]ContentPart, 0, len(parts))
	for _, p := range parts {
		cleaned := stripLeakedContentFilterSuffix(p.Text)
		if strings.TrimSpace(cleaned) == "" {
			continue
		}
		p.Text = cleaned
		out = append(out, p)
	}
	return out
}

func stripLeakedContentFilterSuffix(text string) string {
	if text == "" {
		return text
	}
	upperText := strings.ToUpper(text)
	idx := strings.Index(upperText, "CONTENT_FILTER")
	if idx < 0 {
		return text
	}
	return strings.TrimRight(text[:idx], " \t\r\n")
}
