package qrdot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRejectsBadKey(t *testing.T) {
	if _, err := New("bad"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateQR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_abc" {
			t.Fatalf("auth %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "qr_1", "shortCode": "abc"})
	}))
	defer srv.Close()
	c, err := New("sk_test_abc")
	if err != nil {
		t.Fatal(err)
	}
	c.BaseURL = srv.URL
	out, err := c.QR.Create(map[string]any{"targetUrl": "https://example.com"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if out["id"] != "qr_1" {
		t.Fatalf("got %#v", out)
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "validation", "message": "bad"},
		})
	}))
	defer srv.Close()
	c, _ := New("sk_test_abc")
	c.BaseURL = srv.URL
	_, err := c.QR.Create(map[string]any{"targetUrl": "nope"}, "")
	ae, ok := err.(*APIError)
	if !ok || ae.Code != "validation" || ae.Status != 400 {
		t.Fatalf("got %#v", err)
	}
}

func TestImageBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
	}))
	defer srv.Close()
	c, _ := New("sk_test_abc")
	c.BaseURL = srv.URL
	data, mime, err := c.QR.Image("qr_1", "png")
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/png" || data[0] != 0x89 {
		t.Fatalf("%s %#v", mime, data)
	}
}

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"ok":true}`)
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%d.", ts)))
	_, _ = mac.Write(body)
	header := fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
	if !VerifyWebhookSignature(secret, body, header, 300) {
		t.Fatal("expected valid")
	}
	if VerifyWebhookSignature(secret, append(body, 'x'), header, 300) {
		t.Fatal("expected invalid")
	}
}

func TestDomainsAndWebhookParityPaths(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.RequestURI())
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()
	c, err := New("sk_test_abc")
	if err != nil {
		t.Fatal(err)
	}
	c.BaseURL = srv.URL
	_, _ = c.Domains.Create(map[string]any{"hostname": "go.example.com"})
	_, _ = c.Domains.List()
	_, _ = c.Domains.Get("dom_1")
	_, _ = c.Domains.DNS("dom_1")
	_, _ = c.Domains.DNSProvider("dom_1")
	_, _ = c.Domains.DomainConnectStart("dom_1")
	_, _ = c.Domains.ForwardDNS("dom_1", map[string]any{"email": "a@b.com"})
	_, _ = c.Domains.Refresh("dom_1")
	_, _ = c.Domains.SetDefault("dom_1")
	_ = c.Domains.Delete("dom_1")
	_, _ = c.Webhooks.ListDeliveries("wh_1", 10)
	_, _ = c.Webhooks.Replay("wh_1", map[string]any{"qr_id": "qr_1", "scan_id": "sc_1", "ts": "t"})
	_, _ = c.Assets.PresignLogo("image/png", "logo.png")
	_, _ = c.Assets.GetLogo("logo_1")
	_, _ = c.Assets.CompleteLogo("logo_1", "logo.png")
	_, _ = c.CreateQR(map[string]any{"targetUrl": "https://example.com"}, "")

	want := []string{
		"POST /v1/domains",
		"GET /v1/domains",
		"GET /v1/domains/dom_1",
		"GET /v1/domains/dom_1/dns",
		"GET /v1/domains/dom_1/dns-provider",
		"POST /v1/domains/dom_1/domain-connect/start",
		"POST /v1/domains/dom_1/dns/forward",
		"POST /v1/domains/dom_1/refresh",
		"POST /v1/domains/dom_1/default",
		"DELETE /v1/domains/dom_1",
		"GET /v1/webhooks/wh_1/deliveries?limit=10",
		"POST /v1/webhooks/wh_1/replay",
		"POST /v1/assets/logo/presign",
		"GET /v1/assets/logo/logo_1",
		"POST /v1/assets/logo/logo_1/complete",
		"POST /v1/qr",
	}
	if len(paths) != len(want) {
		t.Fatalf("got %d paths %#v want %d", len(paths), paths, len(want))
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("path[%d]=%q want %q", i, paths[i], want[i])
		}
	}
}
