package main

import "strings"

// sanitizeWorkflowLog makes an error safe to print on a GitHub Actions runner.
// The runner treats "::" and percent-encoded newlines in stdout and stderr as
// workflow commands. "%" is escaped first so "%3A%3A" cannot decode into "::".
func sanitizeWorkflowLog(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "::", ": :")
	return s
}
