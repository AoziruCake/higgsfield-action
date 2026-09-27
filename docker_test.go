package higgsfieldaction_test

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// GitHub Actions mounts GITHUB_WORKSPACE as the runner user (uid 1001, mode 0755).
// Distroless :nonroot is uid 65532 and cannot create output directories there.
func TestDockerfileRunsAsWorkspaceWritableUser(t *testing.T) {
	t.Parallel()

	file, err := os.Open("Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var runtimeFrom string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "FROM ") {
			runtimeFrom = line
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if runtimeFrom == "" {
		t.Fatal("Dockerfile has no FROM line")
	}
	if strings.Contains(runtimeFrom, "nonroot") {
		t.Fatalf("runtime %s cannot mkdir in GITHUB_WORKSPACE (permission denied)", runtimeFrom)
	}
}
