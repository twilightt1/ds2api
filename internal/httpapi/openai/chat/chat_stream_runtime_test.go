package chat

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestChatStreamKeepAliveEmitsEmptyChoiceDataFrame(t *testing.T) {
	rec := httptest.NewRecorder()
	runtime := newChatStreamRuntime(
		rec,
		http.NewResponseController(rec),
		true,
		"chatcmpl-test",
		time.Now().Unix(),
		"deepseek-v4-flash",
		"prompt",
		false,
		false,
		true,
		nil,
		nil,
		false,
		false,
	)

	runtime.sendKeepAlive()

	body := rec.Body.String()
	if !strings.Contains(body, ": keep-alive\n\n") {
		t.Fatalf("expected keep-alive comment, got %q", body)
	}
	frames, done := parseSSEDataFrames(t, body)
	if done {
		t.Fatalf("keep-alive must not emit [DONE], body=%q", body)
	}
	if len(frames) != 1 {
		t.Fatalf("expected one data frame, got %d body=%q", len(frames), body)
	}
	if got := asString(frames[0]["id"]); got != "chatcmpl-test" {
		t.Fatalf("expected completion id to be preserved, got %q", got)
	}
	if got := asString(frames[0]["object"]); got != "chat.completion.chunk" {
		t.Fatalf("expected chat chunk object, got %q", got)
	}
	choices, _ := frames[0]["choices"].([]any)
	if len(choices) != 0 {
		t.Fatalf("expected empty choices heartbeat, got %#v", choices)
	}
}
