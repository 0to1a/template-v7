package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Google's public certs for Firebase ID tokens, keyed by kid
const firebaseCertsURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

var maxAgePattern = regexp.MustCompile(`max-age=(\d+)`)

// Returns the verified email behind a Firebase ID token
type IDTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (string, error)
}

type firebaseClaims struct {
	jwt.RegisteredClaims
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// Checks signature, aud, iss and exp per Firebase's ID token rules
type FirebaseVerifier struct {
	projectID string
	certsURL  string
	client    *http.Client
	now       func() time.Time

	// ponytail: one lock around the cert fetch; fine for login traffic
	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	expires time.Time
}

func NewFirebaseVerifier(projectID string) *FirebaseVerifier {
	return &FirebaseVerifier{
		projectID: projectID,
		certsURL:  firebaseCertsURL,
		client:    &http.Client{Timeout: 10 * time.Second},
		now:       time.Now,
	}
}

func (v *FirebaseVerifier) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	var claims firebaseClaims
	token, err := jwt.ParseWithClaims(idToken, &claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return v.key(ctx, kid)
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name}),
		jwt.WithAudience(v.projectID),
		jwt.WithIssuer("https://securetoken.google.com/"+v.projectID),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(func() time.Time { return v.now() }),
	)
	if err != nil || !token.Valid {
		return "", errInvalidToken
	}
	if claims.Subject == "" || claims.Email == "" || !claims.EmailVerified {
		return "", errInvalidToken
	}
	return claims.Email, nil
}

func (v *FirebaseVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.keys == nil || !v.now().Before(v.expires) {
		if err := v.refresh(ctx); err != nil {
			return nil, err
		}
	}
	key, ok := v.keys[kid]
	if !ok {
		return nil, errInvalidToken
	}
	return key, nil
}

func (v *FirebaseVerifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.certsURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth: fetching firebase certs: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: fetching firebase certs: status %d", resp.StatusCode)
	}

	var pems map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&pems); err != nil {
		return fmt.Errorf("auth: decoding firebase certs: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(pems))
	for kid, certPEM := range pems {
		block, _ := pem.Decode([]byte(certPEM))
		if block == nil {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		if key, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			keys[kid] = key
		}
	}

	maxAge := time.Hour
	if m := maxAgePattern.FindStringSubmatch(resp.Header.Get("Cache-Control")); m != nil {
		if secs, err := strconv.Atoi(m[1]); err == nil {
			maxAge = time.Duration(secs) * time.Second
		}
	}
	v.keys = keys
	v.expires = v.now().Add(maxAge)
	return nil
}
