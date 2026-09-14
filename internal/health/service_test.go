package health

import (
	"context"
	"testing"
)

// TC-001-1: /health answers with zero dependencies — nil pool proves it never
// touches the database the check-database cron uses
func TestGetHealth_TC001_1(t *testing.T) {
	resp, err := NewService(nil).GetHealth(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, want %q", resp.Status, "ok")
	}
}
