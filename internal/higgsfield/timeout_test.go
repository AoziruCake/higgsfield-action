package higgsfield

import (
	"testing"
	"time"
)

func TestNewClient_statusTimeout(t *testing.T) {
	c := NewClient("id:secret", nil)
	if c.httpClient.Timeout != 60*time.Second {
		t.Fatalf("Timeout = %s, want 60s", c.httpClient.Timeout)
	}
}
