package main

import (
	"strings"
	"testing"
)

func TestSanitizeWorkflowLog(t *testing.T) {
	got := sanitizeWorkflowLog("::add-mask::secret\nnext\r%3A%3A")
	if strings.Contains(got, "::") || strings.Contains(got, "\n") || strings.Contains(got, "\r") {
		t.Fatalf("command characters survived: %q", got)
	}
	if strings.Contains(got, "%3A%3A") {
		t.Fatalf("encoded colons were not escaped: %q", got)
	}
	if !strings.Contains(got, "%253A%253A") {
		t.Fatalf("got %q", got)
	}
}
