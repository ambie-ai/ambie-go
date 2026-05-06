package ambie

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DefaultToleranceSeconds is the default maximum age for a webhook signature.
const DefaultToleranceSeconds = 300

// VerifyWebhookOptions configures VerifyWebhookSignature.
type VerifyWebhookOptions struct {
	Signature        string // X-Ambie-Signature header value
	Body             []byte // raw request body bytes (NOT parsed JSON)
	Secret           string // webhook secret from /signup
	ToleranceSeconds int    // 0 = use default (300)
	NowFunc          func() int64
}

// VerifyWebhookSignature verifies a Stripe-compatible webhook signature
// of the form "t=<unix_ts>,v1=<hex>". It recomputes HMAC-SHA256 of
// "{t}.{body}" with the secret and compares in constant time.
func VerifyWebhookSignature(opts VerifyWebhookOptions) error {
	tolerance := opts.ToleranceSeconds
	if tolerance == 0 {
		tolerance = DefaultToleranceSeconds
	}

	var t, v1 string
	for _, part := range strings.Split(opts.Signature, ",") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			t = v
		case "v1":
			v1 = v
		}
	}
	if t == "" || v1 == "" {
		return errors.New("invalid signature header")
	}

	ts, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	now := time.Now().Unix()
	if opts.NowFunc != nil {
		now = opts.NowFunc()
	}
	if abs(now-ts) > int64(tolerance) {
		return errors.New("signature timestamp outside tolerance")
	}

	mac := hmac.New(sha256.New, []byte(opts.Secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(opts.Body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(v1)) {
		return errors.New("signature mismatch")
	}
	return nil
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
