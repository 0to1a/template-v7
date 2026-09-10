package template

import (
	"strings"
	"testing"
)

func TestRenderOTP(t *testing.T) {
	html, err := RenderOTP(OTPData{Code: "654321"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "654321") {
		t.Fatalf("rendered HTML missing the code: %q", html)
	}
}
