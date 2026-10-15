package higgsfieldaction_test

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// GitHub Actions mounts GITHUB_WORKSPACE as the runner user (uid 1001, mode 0755).
// The image must run as that uid. Distroless :nonroot is 65532 and cannot write there.
func TestDockerfileRunsAsWorkspaceWritableUser(t *testing.T) {
	t.Parallel()

	file, err := os.Open("Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var runtimeFrom, user string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "FROM ") {
			runtimeFrom = line
			user = ""
		}
		if strings.HasPrefix(line, "USER ") {
			user = strings.TrimSpace(strings.TrimPrefix(line, "USER "))
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
	if user != "1001" {
		t.Fatalf("USER = %q, want 1001", user)
	}
}
