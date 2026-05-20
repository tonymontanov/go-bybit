/*
FILE: internal/auth/sign.go

DESCRIPTION:
Request signing for Bybit V5. Two flavours:

  1. REST signing (SignREST):
     preHash   = timestamp + apiKey + recvWindow + (queryString | jsonBody)
     signature = hex( HMAC_SHA256(secretKey, preHash) )

     - timestamp:  current Unix time in MILLISECONDS, as decimal string.
     - apiKey:     X-BAPI-API-KEY (bound to the Signer).
     - recvWindow: ms allowed window between client time and server receipt.
                   Default 5000 (Bybit hard cap is 60000; values > 5000 only
                   make sense for pathological VPN setups).
     - body:       for GET — canonical query string ("k1=v1&k2=v2"); for
                   POST — the exact JSON body that goes on the wire.
                   IMPORTANT: signing MUST happen on the same byte sequence
                   that is sent — re-marshalling can reorder map keys and
                   break the signature. See SignREST contract: it takes
                   the already-rendered body string.

     Output sent as headers:
       X-BAPI-API-KEY        — apiKey
       X-BAPI-SIGN           — hex(signature)
       X-BAPI-SIGN-TYPE      — "2" (HMAC)
       X-BAPI-TIMESTAMP      — ms timestamp string
       X-BAPI-RECV-WINDOW    — recvWindow string (ms)

  2. WebSocket auth (SignWS):
     preHash   = "GET/realtime" + expires
     signature = hex( HMAC_SHA256(secretKey, preHash) )

     - expires: Unix time in MILLISECONDS until which the auth message is
                considered valid. Typical practice: expires = now + 1000.
                Bybit checks server-side that expires > server_now and
                rejects otherwise.

     The signature is sent as part of the JSON message:
       {"op":"auth","args":[apiKey, expires, signature]}

The hex output is LOWERCASE — Bybit compares it case-insensitively in
practice but the docs canonical form is lowercase, so we emit lowercase
to keep diffing logs deterministic.

SECURITY:
  - Secret material is stored as []byte and never serialized. String()
    redacts the API key.
  - Pre-hash and body strings MUST NOT be logged.

DEPENDENCIES:
  - crypto/hmac, crypto/sha256: signing.
  - encoding/hex:               output encoding.
*/

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

// ErrSignerDisabled is returned when Sign* is called on a signer that has
// no credentials. The same Signer can serve public REST/WS endpoints
// without keys; private endpoints surface this error at call time.
var ErrSignerDisabled = errors.New("auth: signer is disabled (api key/secret not configured)")

// Signer holds Bybit credentials and produces V5-conformant signatures.
// Safe for concurrent use: all fields are read-only after construction.
type Signer struct {
	apiKey    string
	secretKey []byte
	enabled   bool
}

// NewSigner creates a Signer. If either apiKey or secretKey is empty the
// signer is disabled and Sign* will return ErrSignerDisabled. This lets a
// single Client hit public endpoints without credentials.
func NewSigner(apiKey, secretKey string) *Signer {
	var enabled bool = apiKey != "" && secretKey != ""
	return &Signer{
		apiKey:    apiKey,
		secretKey: []byte(secretKey),
		enabled:   enabled,
	}
}

// Enabled reports whether the signer has credentials.
func (s *Signer) Enabled() bool { return s != nil && s.enabled }

// APIKey returns the bound API key, used to populate X-BAPI-API-KEY.
func (s *Signer) APIKey() string {
	if s == nil {
		return ""
	}
	return s.apiKey
}

// MillisTimestamp returns now in milliseconds as a decimal string. Used for
// X-BAPI-TIMESTAMP and as the timestamp component of the pre-hash. If now
// is zero, time.Now() is used.
func (s *Signer) MillisTimestamp(now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	return strconv.FormatInt(now.UnixMilli(), 10)
}

/*
SignREST returns the lowercase hex HMAC-SHA256 signature for a Bybit V5
REST request, per the V5 specification:

	preHash = timestamp + apiKey + recvWindow + body

Parameters:
  - timestamp:  ms timestamp string (use MillisTimestamp).
  - recvWindow: ms recv-window string (e.g. "5000").
  - body:       for GET — canonical query string ("k1=v1&k2=v2"), without
    a leading "?"; for POST — the exact JSON body string.
    The string MUST be the exact byte sequence that the HTTP
    layer will put on the wire.

Returns ErrSignerDisabled if the signer has no credentials.
*/
func (s *Signer) SignREST(timestamp, recvWindow, body string) (string, error) {
	if !s.Enabled() {
		return "", ErrSignerDisabled
	}
	var sb strings.Builder
	sb.Grow(len(timestamp) + len(s.apiKey) + len(recvWindow) + len(body))
	sb.WriteString(timestamp)
	sb.WriteString(s.apiKey)
	sb.WriteString(recvWindow)
	sb.WriteString(body)
	return s.hmacHex(sb.String()), nil
}

/*
SignWS returns the lowercase hex HMAC-SHA256 signature for the Bybit V5
WebSocket auth message:

	preHash = "GET/realtime" + expires

Parameters:
  - expires: ms timestamp string until which the auth is valid (typical
    value: current_ms + 1000).

Returns ErrSignerDisabled if the signer has no credentials.
*/
func (s *Signer) SignWS(expires string) (string, error) {
	if !s.Enabled() {
		return "", ErrSignerDisabled
	}
	var preHash string = "GET/realtime" + expires
	return s.hmacHex(preHash), nil
}

// hmacHex computes hex(HMAC_SHA256(secret, msg)).
func (s *Signer) hmacHex(msg string) string {
	var mac = hmac.New(sha256.New, s.secretKey)
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// String returns a log-safe representation that NEVER includes the secret.
func (s *Signer) String() string {
	if s == nil || !s.enabled {
		return "auth.Signer{disabled}"
	}
	return "auth.Signer{enabled, apiKey=" + redact(s.apiKey) + "}"
}

// redact turns a string into "abcd…wxyz" — first/last 4 chars. For logs only.
func redact(v string) string {
	if len(v) <= 8 {
		return "***"
	}
	return v[:4] + "…" + v[len(v)-4:]
}
