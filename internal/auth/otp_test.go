package auth

import (
	"testing"
	"time"
)

var testSecret = []byte("test-secret-0123456789-0123456789-abc")

const testUUID = "00000000-0000-0000-0000-000000000002"

// Arbitrary instant; keeps OTP tests deterministic
var fixedTime = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

func TestVerifyCode_CurrentStepAccepted(t *testing.T) {
	code := generateCode(testSecret, testUUID, "user@example.com", fixedTime)
	if !verifyCode(testSecret, testUUID, "user@example.com", code, fixedTime) {
		t.Fatal("current-step TOTP was rejected")
	}
	// Still the same 300-second step 299 seconds later.
	later := fixedTime.Truncate(totpPeriod).Add(totpPeriod - time.Second)
	if !verifyCode(testSecret, testUUID, "user@example.com", code, later) {
		t.Fatal("code rejected within its own step")
	}
}

func TestVerifyCode_PreviousStepRejected(t *testing.T) {
	previous := generateCode(testSecret, testUUID, "user@example.com", fixedTime.Add(-totpPeriod))
	current := generateCode(testSecret, testUUID, "user@example.com", fixedTime)
	if previous == current {
		t.Fatal("test setup: adjacent steps produced the same code")
	}
	// Skew is 0: the previous step's code must not verify now.
	if verifyCode(testSecret, testUUID, "user@example.com", previous, fixedTime) {
		t.Fatal("previous-step TOTP was accepted; skew must be 0")
	}
}

func TestVerifyCode_InvalidCodeRejected(t *testing.T) {
	if verifyCode(testSecret, testUUID, "user@example.com", "000000", fixedTime) &&
		verifyCode(testSecret, testUUID, "user@example.com", "999999", fixedTime) {
		t.Fatal("arbitrary codes were accepted")
	}
	if verifyCode(testSecret, testUUID, "user@example.com", "", fixedTime) {
		t.Fatal("empty code was accepted")
	}
}

func TestLocalAdminCode_ExactMatchOnly(t *testing.T) {
	// Fixed dev code belongs to admin@localhost only
	if got := generateCode(testSecret, testUUID, "admin@localhost", fixedTime); got != localAdminCode {
		t.Fatalf("admin@localhost code = %s, want %s", got, localAdminCode)
	}
	// Other @localhost addresses must go through TOTP
	if verifyCode(testSecret, testUUID, "other@localhost", localAdminCode, fixedTime) {
		t.Fatal("fixed code accepted for a non-admin @localhost account")
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := normalizeEmail("  Admin@LocalHost \n"); got != "admin@localhost" {
		t.Fatalf("normalizeEmail = %q", got)
	}
}
