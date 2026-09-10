package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	totpDigits = 6
	totpPeriod = 300 * time.Second

	// Static dev credential; remove before real deployment
	localAdminEmail = "admin@localhost"
	localAdminCode  = "123456"

	totpSecretMessagePrefix = "template-v5:login-totp:v1:"
)

// Canonical form for all email lookups and storage
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isLocalAdmin(normalizedEmail string) bool {
	return normalizedEmail == localAdminEmail
}

// No stored secret: rotating JWT_SECRET invalidates OTPs
func deriveTOTPSecret(jwtSecret []byte, publicUUID string) []byte {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(totpSecretMessagePrefix + publicUUID))
	return mac.Sum(nil)
}

// RFC 4226 HOTP using HMAC-SHA256
func hotp(secret []byte, counter uint64) string {
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], counter)

	mac := hmac.New(sha256.New, secret)
	mac.Write(counterBytes[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0F
	binCode := (uint32(sum[offset]&0x7F) << 24) |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])

	mod := uint32(1)
	for range totpDigits {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", totpDigits, binCode%mod)
}

// Skew=0: a code is replayable within its 5-min step
func currentTOTP(secret []byte, now time.Time) string {
	counter := uint64(now.Unix()) / uint64(totpPeriod.Seconds())
	return hotp(secret, counter)
}

// Takes time as a param so tests stay deterministic
func generateCode(jwtSecret []byte, publicUUID, normalizedEmail string, now time.Time) string {
	if isLocalAdmin(normalizedEmail) {
		return localAdminCode
	}
	secret := deriveTOTPSecret(jwtSecret, publicUUID)
	return currentTOTP(secret, now)
}

// Constant-time compare avoids a timing oracle
func verifyCode(jwtSecret []byte, publicUUID, normalizedEmail, code string, now time.Time) bool {
	expected := generateCode(jwtSecret, publicUUID, normalizedEmail, now)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1
}
