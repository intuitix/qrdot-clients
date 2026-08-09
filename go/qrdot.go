package qrdot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.qrdot.dev"

// APIError is returned for non-2xx API responses.
type APIError struct {
	Code    string
	Message string
	Status  int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Message)
}

// Client is the QR. API client.
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client

	QR        *QRResource
	Assets    *AssetsResource
	Analytics *AnalyticsResource
	Webhooks  *WebhooksResource
	Domains   *DomainsResource
}

// New creates a client. apiKey must start with sk_live_ (legacy sk_test_ accepted).
func New(apiKey string) (*Client, error) {
	if !strings.HasPrefix(apiKey, "sk_test_") && !strings.HasPrefix(apiKey, "sk_live_") {
		return nil, fmt.Errorf("apiKey must start with sk_live_ (legacy sk_test_ accepted)")
	}
	c := &Client{
		APIKey:     apiKey,
		BaseURL:    defaultBaseURL,
		HTTPClient: http.DefaultClient,
	}
	c.QR = &QRResource{c: c}
	c.Assets = &AssetsResource{c: c}
	c.Analytics = &AnalyticsResource{c: c}
	c.Webhooks = &WebhooksResource{c: c}
	c.Domains = &DomainsResource{c: c}
	return c, nil
}

// CreateQR is a convenience alias for QR.Create.
func (c *Client) CreateQR(payload map[string]any, idempotencyKey string) (map[string]any, error) {
	return c.QR.Create(payload, idempotencyKey)
}

func (c *Client) request(method, path string, body any, headers map[string]string, out any) error {
	data, status, _, err := c.raw(method, path, body, headers)
	if err != nil {
		return err
	}
	if status == 204 || len(data) == 0 {
		return nil
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func (c *Client) requestBytes(method, path string, body any) ([]byte, string, error) {
	data, _, ct, err := c.raw(method, path, body, nil)
	if err != nil {
		return nil, "", err
	}
	mime := "application/octet-stream"
	if ct != "" {
		mime = strings.TrimSpace(strings.Split(ct, ";")[0])
	}
	return data, mime, nil
}

func (c *Client) raw(method, path string, body any, headers map[string]string) ([]byte, int, string, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, "", err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(c.BaseURL, "/")+path, rdr)
	if err != nil {
		return nil, 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(req)
	if err != nil {
		return nil, 0, "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		code, msg := "internal", res.Status
		var env struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(data, &env) == nil && env.Error.Code != "" {
			code = env.Error.Code
			msg = env.Error.Message
		}
		return nil, res.StatusCode, "", &APIError{Code: code, Message: msg, Status: res.StatusCode}
	}
	return data, res.StatusCode, res.Header.Get("Content-Type"), nil
}

func (c *Client) putExternal(urlStr string, body []byte, headers map[string]string) (int, error) {
	req, err := http.NewRequest(http.MethodPut, urlStr, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	return res.StatusCode, nil
}

// VerifyWebhookSignature checks X-Qrdot-Signature: t=…,v1=… over "{t}.{rawBody}".
func VerifyWebhookSignature(secret string, rawBody []byte, header string, toleranceSec int) bool {
	if toleranceSec <= 0 {
		toleranceSec = 300
	}
	parts := map[string]string{}
	for _, p := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			parts[kv[0]] = kv[1]
		}
	}
	t, err := strconv.ParseInt(parts["t"], 10, 64)
	if err != nil || t == 0 || parts["v1"] == "" {
		return false
	}
	now := time.Now().Unix()
	if now-t > int64(toleranceSec) || t-now > int64(toleranceSec) {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%d.", t)))
	_, _ = mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(parts["v1"]))
}

// QRResource wraps /v1/qr.
type QRResource struct{ c *Client }

func (r *QRResource) Create(payload map[string]any, idempotencyKey string) (map[string]any, error) {
	headers := map[string]string{}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/qr", payload, headers, &out)
	return out, err
}

func (r *QRResource) List(query url.Values) (map[string]any, error) {
	path := "/v1/qr"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	var out map[string]any
	err := r.c.request(http.MethodGet, path, nil, nil, &out)
	return out, err
}

func (r *QRResource) Batch(items []map[string]any, idempotencyKey string) (map[string]any, error) {
	headers := map[string]string{}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/qr/batch", map[string]any{"items": items}, headers, &out)
	return out, err
}

func (r *QRResource) Get(id string) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodGet, "/v1/qr/"+url.PathEscape(id), nil, nil, &out)
	return out, err
}

func (r *QRResource) Update(id string, payload map[string]any) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodPatch, "/v1/qr/"+url.PathEscape(id), payload, nil, &out)
	return out, err
}

func (r *QRResource) Delete(id string) error {
	return r.c.request(http.MethodDelete, "/v1/qr/"+url.PathEscape(id), nil, nil, nil)
}

func (r *QRResource) Duplicate(id string) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/qr/"+url.PathEscape(id)+"/duplicate", nil, nil, &out)
	return out, err
}

func (r *QRResource) Image(id, format string) ([]byte, string, error) {
	if format == "" {
		format = "png"
	}
	return r.c.requestBytes(http.MethodGet, "/v1/qr/"+url.PathEscape(id)+"/image."+format, nil)
}

func (r *QRResource) ExportImages(ids []string, format string) ([]byte, string, error) {
	if format == "" {
		format = "png"
	}
	return r.c.requestBytes(http.MethodPost, "/v1/qr/export/images", map[string]any{"ids": ids, "format": format})
}

// AssetsResource wraps /v1/assets/logo.
type AssetsResource struct{ c *Client }

func (r *AssetsResource) PresignLogo(contentType, filename string) (map[string]any, error) {
	body := map[string]any{"content_type": contentType}
	if filename != "" {
		body["filename"] = filename
	}
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/assets/logo/presign", body, nil, &out)
	return out, err
}

func (r *AssetsResource) CompleteLogo(id, filename string) (map[string]any, error) {
	complete := map[string]any{}
	if filename != "" {
		complete["filename"] = filename
	}
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/assets/logo/"+url.PathEscape(id)+"/complete", complete, nil, &out)
	return out, err
}

func (r *AssetsResource) ListLogos() (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodGet, "/v1/assets/logo", nil, nil, &out)
	return out, err
}

func (r *AssetsResource) GetLogo(id string) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodGet, "/v1/assets/logo/"+url.PathEscape(id), nil, nil, &out)
	return out, err
}

func (r *AssetsResource) DeleteLogo(id string) error {
	return r.c.request(http.MethodDelete, "/v1/assets/logo/"+url.PathEscape(id), nil, nil, nil)
}

func (r *AssetsResource) UploadLogo(data []byte, contentType, filename string) (map[string]any, error) {
	presign, err := r.PresignLogo(contentType, filename)
	if err != nil {
		return nil, err
	}
	uploadURL, _ := presign["upload_url"].(string)
	assetID, _ := presign["asset_id"].(string)
	hdrs := map[string]string{}
	if h, ok := presign["headers"].(map[string]any); ok {
		for k, v := range h {
			hdrs[k] = fmt.Sprint(v)
		}
	}
	status, err := r.c.putExternal(uploadURL, data, hdrs)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, &APIError{Code: "internal", Message: fmt.Sprintf("Logo storage upload failed (%d)", status), Status: status}
	}
	return r.CompleteLogo(assetID, filename)
}

// AnalyticsResource wraps /v1/analytics.
type AnalyticsResource struct{ c *Client }

func (r *AnalyticsResource) Summary(query url.Values) (map[string]any, error) {
	path := "/v1/analytics/summary"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	var out map[string]any
	err := r.c.request(http.MethodGet, path, nil, nil, &out)
	return out, err
}

func (r *AnalyticsResource) QR(id string, query url.Values) (map[string]any, error) {
	path := "/v1/analytics/qr/" + url.PathEscape(id)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	var out map[string]any
	err := r.c.request(http.MethodGet, path, nil, nil, &out)
	return out, err
}

func (r *AnalyticsResource) Scans(id string, query url.Values) (map[string]any, error) {
	path := "/v1/analytics/qr/" + url.PathEscape(id) + "/scans"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	var out map[string]any
	err := r.c.request(http.MethodGet, path, nil, nil, &out)
	return out, err
}

// WebhooksResource wraps /v1/webhooks.
type WebhooksResource struct{ c *Client }

func (r *WebhooksResource) Create(payload map[string]any) (map[string]any, error) {
	if _, ok := payload["events"]; !ok {
		payload["events"] = []string{"qr.scanned"}
	}
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/webhooks", payload, nil, &out)
	return out, err
}

func (r *WebhooksResource) List() (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodGet, "/v1/webhooks", nil, nil, &out)
	return out, err
}

func (r *WebhooksResource) Update(id string, patch map[string]any) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodPatch, "/v1/webhooks/"+url.PathEscape(id), patch, nil, &out)
	return out, err
}

func (r *WebhooksResource) Delete(id string) error {
	return r.c.request(http.MethodDelete, "/v1/webhooks/"+url.PathEscape(id), nil, nil, nil)
}

func (r *WebhooksResource) Test(id string, payload map[string]any) (map[string]any, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/webhooks/"+url.PathEscape(id)+"/test", payload, nil, &out)
	return out, err
}

func (r *WebhooksResource) ListDeliveries(id string, limit int) (map[string]any, error) {
	path := "/v1/webhooks/" + url.PathEscape(id) + "/deliveries"
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var out map[string]any
	err := r.c.request(http.MethodGet, path, nil, nil, &out)
	return out, err
}

func (r *WebhooksResource) Replay(id string, payload map[string]any) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/webhooks/"+url.PathEscape(id)+"/replay", payload, nil, &out)
	return out, err
}

// VerifySignature is VerifyWebhookSignature.
func (r *WebhooksResource) VerifySignature(secret string, rawBody []byte, header string) bool {
	return VerifyWebhookSignature(secret, rawBody, header, 300)
}

// DomainsResource wraps /v1/domains.
type DomainsResource struct{ c *Client }

func (r *DomainsResource) Create(payload map[string]any) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/domains", payload, nil, &out)
	return out, err
}

func (r *DomainsResource) List() (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodGet, "/v1/domains", nil, nil, &out)
	return out, err
}

func (r *DomainsResource) Get(id string) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodGet, "/v1/domains/"+url.PathEscape(id), nil, nil, &out)
	return out, err
}

func (r *DomainsResource) DNS(id string) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodGet, "/v1/domains/"+url.PathEscape(id)+"/dns", nil, nil, &out)
	return out, err
}

func (r *DomainsResource) DNSProvider(id string) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodGet, "/v1/domains/"+url.PathEscape(id)+"/dns-provider", nil, nil, &out)
	return out, err
}

func (r *DomainsResource) DomainConnectStart(id string) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/domains/"+url.PathEscape(id)+"/domain-connect/start", nil, nil, &out)
	return out, err
}

func (r *DomainsResource) ForwardDNS(id string, payload map[string]any) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/domains/"+url.PathEscape(id)+"/dns/forward", payload, nil, &out)
	return out, err
}

func (r *DomainsResource) Refresh(id string) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/domains/"+url.PathEscape(id)+"/refresh", nil, nil, &out)
	return out, err
}

func (r *DomainsResource) SetDefault(id string) (map[string]any, error) {
	var out map[string]any
	err := r.c.request(http.MethodPost, "/v1/domains/"+url.PathEscape(id)+"/default", nil, nil, &out)
	return out, err
}

func (r *DomainsResource) Delete(id string) error {
	return r.c.request(http.MethodDelete, "/v1/domains/"+url.PathEscape(id), nil, nil, nil)
}
