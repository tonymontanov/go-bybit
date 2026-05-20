/*
FILE: internal/auth/sign_test.go

DESCRIPTION:
Property tests + canonical-vector tests for the Bybit V5 signer.

The canonical vectors here are computed manually with a known
secret/timestamp/body. They are STABLE — if the signature output ever
changes, either:
  a) the algorithm changed (bug or deliberate change requiring a major
     version bump), or
  b) the test data was edited (only valid if a corresponding regeneration
     is documented in the commit).

If you add a new vector, compute it with:

	echo -n "<timestamp><apiKey><recvWindow><body>" | \
	  openssl dgst -sha256 -hmac "<secret>" -hex
*/

package auth

import (
	"strings"
	"testing"
	"time"
)

func TestSigner_DisabledByDefault(t *testing.T) {
	var s *Signer = NewSigner("", "")
	if s.Enabled() {
		t.Fatalf("signer must be disabled with empty creds")
	}
	var _, err = s.SignREST("1700000000000", "5000", "")
	if err != ErrSignerDisabled {
		t.Fatalf("expected ErrSignerDisabled, got %v", err)
	}
	_, err = s.SignWS("1700000000000")
	if err != ErrSignerDisabled {
		t.Fatalf("expected ErrSignerDisabled for SignWS, got %v", err)
	}
}

func TestSigner_REST_CanonicalVector(t *testing.T) {
	// This vector is reproducible from the shell:
	//   echo -n '1700000000000XXXAPIKEYXX5000{"category":"linear"}' | \
	//     openssl dgst -sha256 -hmac 'TESTSECRET' -hex
	// Update procedure: if the algorithm changes (which would be a major
	// version bump of the SDK), recompute "want" via openssl as above.
	const apiKey = "XXXAPIKEYXX"
	const secret = "TESTSECRET"
	const ts = "1700000000000"
	const recv = "5000"
	const body = `{"category":"linear"}`
	const want = "3f59ef320508c272f8261ebe20f7f127db7c5dc8edd9dc67b2f86ec92c47b50b"

	var s *Signer = NewSigner(apiKey, secret)
	var got, err = s.SignREST(ts, recv, body)
	if err != nil {
		t.Fatalf("SignREST: %v", err)
	}
	if got != want {
		t.Fatalf("SignREST mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestSigner_WS_CanonicalVector(t *testing.T) {
	// Reproducible from the shell:
	//   echo -n 'GET/realtime1700000000000' | \
	//     openssl dgst -sha256 -hmac 'TESTSECRET' -hex
	const secret = "TESTSECRET"
	const expires = "1700000000000"
	const want = "89bb7f322bfadeac451e9126155aa704ac6257367b3a9ba7e5902dc4c50cf16b"

	var s *Signer = NewSigner("KEY", secret)
	var got, err = s.SignWS(expires)
	if err != nil {
		t.Fatalf("SignWS: %v", err)
	}
	if got != want {
		t.Fatalf("SignWS mismatch:\n got %s\nwant %s", got, want)
	}
	if got != strings.ToLower(got) {
		t.Fatalf("ws signature must be lowercase hex, got %q", got)
	}
	var got2, _ = s.SignWS(expires)
	if got != got2 {
		t.Fatalf("SignWS must be deterministic for the same input")
	}
}

func TestSigner_MillisTimestamp(t *testing.T) {
	var s *Signer = NewSigner("K", "S")
	var fixed time.Time = time.UnixMilli(1700000000123)
	var got string = s.MillisTimestamp(fixed)
	if got != "1700000000123" {
		t.Fatalf("MillisTimestamp(fixed): got %q, want 1700000000123", got)
	}

	var dynamic string = s.MillisTimestamp(time.Time{})
	if len(dynamic) < 13 {
		t.Fatalf("MillisTimestamp(zero) must produce >=13 digits, got %q", dynamic)
	}
}

func TestSigner_StringRedactsKey(t *testing.T) {
	var s *Signer = NewSigner("ABCDEFGHIJKLMNOP", "secret")
	var got string = s.String()
	if strings.Contains(got, "secret") {
		t.Fatalf("String must not contain secret: %s", got)
	}
	if !strings.Contains(got, "ABCD") || !strings.Contains(got, "MNOP") {
		t.Fatalf("String must contain redacted key prefix/suffix: %s", got)
	}
}
