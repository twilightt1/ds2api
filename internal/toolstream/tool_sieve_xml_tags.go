package toolstream

import "regexp"

// --- XML tool call support for the streaming sieve ---

//nolint:unused // kept as explicit tag inventory for future XML sieve refinements.
var xmlToolCallClosingTags = []string{"</tool_calls>", "</|dsml|tool_calls>", "</|dsml tool_calls>", "</dsml|tool_calls>", "</dsml tool_calls>", "</｜tool_calls>", "</|tool_calls>"}
var xmlToolCallOpeningTags = []string{
	"<tool_calls", "<invoke",
	"<|dsml|tool_calls", "<|dsml|invoke",
	"<|dsml tool_calls", "<|dsml invoke",
	"<dsml|tool_calls", "<dsml|invoke",
	"<dsml tool_calls", "<dsml invoke",
	"<｜tool_calls", "<｜invoke",
	"<|tool_calls", "<|invoke",
}

// xmlToolCallTagPairs maps each opening tag to its expected closing tag.
// Order matters: longer/wrapper tags must be checked first.
var xmlToolCallTagPairs = []struct{ open, close string }{
	{"<|dsml|tool_calls", "</|dsml|tool_calls>"},
	{"<|dsml tool_calls", "</|dsml tool_calls>"},
	{"<dsml|tool_calls", "</dsml|tool_calls>"},
	{"<dsml tool_calls", "</dsml tool_calls>"},
	{"<｜tool_calls", "</｜tool_calls>"},
	{"<|tool_calls", "</|tool_calls>"},
	{"<tool_calls", "</tool_calls>"},
}

// xmlToolCallBlockPattern matches a complete canonical XML tool call block.
//
//nolint:unused // reserved for future fast-path XML block detection.
var xmlToolCallBlockPattern = regexp.MustCompile(`(?is)((?:<tool_calls\b|<\|dsml\|tool_calls\b)[^>]*>\s*(?:.*?)\s*(?:</tool_calls>|</\|dsml\|tool_calls>))`)

// xmlToolTagsToDetect is the set of XML tag prefixes used by findToolSegmentStart.
var xmlToolTagsToDetect = []string{
	"<|dsml|tool_calls>", "<|dsml|tool_calls\n", "<|dsml|tool_calls ",
	"<|dsml|invoke ", "<|dsml|invoke\n", "<|dsml|invoke\t", "<|dsml|invoke\r",
	"<|dsml tool_calls>", "<|dsml tool_calls\n", "<|dsml tool_calls ",
	"<|dsml invoke ", "<|dsml invoke\n", "<|dsml invoke\t", "<|dsml invoke\r",
	"<dsml|tool_calls>", "<dsml|tool_calls\n", "<dsml|tool_calls ",
	"<dsml|invoke ", "<dsml|invoke\n", "<dsml|invoke\t", "<dsml|invoke\r",
	"<dsml tool_calls>", "<dsml tool_calls\n", "<dsml tool_calls ",
	"<dsml invoke ", "<dsml invoke\n", "<dsml invoke\t", "<dsml invoke\r",
	"<｜tool_calls>", "<｜tool_calls\n", "<｜tool_calls ",
	"<｜invoke ", "<｜invoke\n", "<｜invoke\t", "<｜invoke\r",
	"<|tool_calls>", "<|tool_calls\n", "<|tool_calls ",
	"<|invoke ", "<|invoke\n", "<|invoke\t", "<|invoke\r",
	"<tool_calls>", "<tool_calls\n", "<tool_calls ", "<invoke ", "<invoke\n", "<invoke\t", "<invoke\r",
}
