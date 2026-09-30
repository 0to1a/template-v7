package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testProjectID = "test-project"

func newTestFirebaseVerifier(t *testing.T) (*FirebaseVerifier, func(jwt.MapClaims) string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    fixedTime.Add(-time.Hour),
		NotAfter:     fixedTime.Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=600")
		_ = json.NewEncoder(w).Encode(map[string]string{"kid-1": certPEM})
	}))
	t.Cleanup(server.Close)

	v := NewFirebaseVerifier(testProjectID)
	v.certsURL = server.URL
	v.now = func() time.Time { return fixedTime }

	sign := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "kid-1"
		signed, err := token.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return signed
	}
	return v, sign
}

func validFirebaseClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss":            "https://securetoken.google.com/" + testProjectID,
		"aud":            testProjectID,
		"sub":            "firebase-uid",
		"iat":            fixedTime.Add(-time.Minute).Unix(),
		"exp":            fixedTime.Add(time.Hour).Unix(),
		"email":          "User@Example.com",
		"email_verified": true,
	}
}

func TestFirebaseVerifier(t *testing.T) {
	v, sign := newTestFirebaseVerifier(t)
	ctx := context.Background()

	email, err := v.VerifyIDToken(ctx, sign(validFirebaseClaims()))
	if err != nil || email != "User@Example.com" {
		t.Fatalf("valid token: email=%q err=%v", email, err)
	}

	bad := map[string]func(jwt.MapClaims){
		"wrong audience":   func(c jwt.MapClaims) { c["aud"] = "other-project" },
		"wrong issuer":     func(c jwt.MapClaims) { c["iss"] = "https://evil.example" },
		"expired":          func(c jwt.MapClaims) { c["exp"] = fixedTime.Add(-time.Second).Unix() },
		"unverified email": func(c jwt.MapClaims) { c["email_verified"] = false },
		"no subject":       func(c jwt.MapClaims) { delete(c, "sub") },
	}
	for name, mutate := range bad {
		claims := validFirebaseClaims()
		mutate(claims)
		if _, err := v.VerifyIDToken(ctx, sign(claims)); err == nil {
			t.Errorf("%s: token accepted", name)
		}
	}

	// Signed by a key Google never published
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	forged := jwt.NewWithClaims(jwt.SigningMethodRS256, validFirebaseClaims())
	forged.Header["kid"] = "kid-1"
	forgedToken, _ := forged.SignedString(otherKey)
	if _, err := v.VerifyIDToken(ctx, forgedToken); err == nil {
		t.Error("forged signature accepted")
	}
}
