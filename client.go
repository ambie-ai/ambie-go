// Package ambie is the official Go SDK for AMBIE.
//
// See https://ambie.ai/sdk for the full API reference.
package ambie

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultBaseURL    = "https://ambie.ai"
	defaultTimeout    = 60 * time.Second
	defaultMaxRetries = 3
	sdkVersion        = "0.1.0"
)

var retryStatus = map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true}

// Client is the AMBIE API client.
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	MaxRetries int
	UserAgent  string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the default base URL.
func WithBaseURL(u string) Option { return func(c *Client) { c.BaseURL = u } }

// WithHTTPClient injects a custom HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.HTTPClient = h } }

// WithMaxRetries sets retry attempts on 429/5xx (default 3).
func WithMaxRetries(n int) Option { return func(c *Client) { c.MaxRetries = n } }

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(ua string) Option { return func(c *Client) { c.UserAgent = ua } }

// New returns a Client authenticated with apiKey.
func New(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("apiKey is required")
	}
	c := &Client{
		APIKey:     apiKey,
		BaseURL:    defaultBaseURL,
		HTTPClient: &http.Client{Timeout: defaultTimeout},
		MaxRetries: defaultMaxRetries,
		UserAgent:  "ambie-go/" + sdkVersion,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// Error is returned for any non-2xx API response after retries.
type Error struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%d %s] %s", e.Status, e.Code, e.Message)
}

func (c *Client) do(ctx context.Context, req *http.Request, out any) error {
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		// rebuild body for retries
		var bodyBytes []byte
		if req.Body != nil && req.GetBody != nil {
			b, err := req.GetBody()
			if err != nil {
				return err
			}
			req.Body = b
			bodyBytes, _ = io.ReadAll(b)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			if attempt >= c.MaxRetries {
				return err
			}
			time.Sleep(backoff(attempt))
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			defer resp.Body.Close()
			if out == nil {
				io.Copy(io.Discard, resp.Body)
				return nil
			}
			if s, ok := out.(*string); ok {
				b, err := io.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				*s = string(b)
				return nil
			}
			return json.NewDecoder(resp.Body).Decode(out)
		}

		if retryStatus[resp.StatusCode] && attempt < c.MaxRetries {
			retryAfter, _ := strconv.ParseFloat(resp.Header.Get("Retry-After"), 64)
			resp.Body.Close()
			if retryAfter > 0 {
				time.Sleep(time.Duration(retryAfter * float64(time.Second)))
			} else {
				time.Sleep(backoff(attempt))
			}
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			continue
		}

		apiErr := &Error{
			Status:    resp.StatusCode,
			Code:      "http_error",
			Message:   "HTTP " + resp.Status,
			RequestID: resp.Header.Get("X-Request-Id"),
		}
		if b, err := io.ReadAll(resp.Body); err == nil {
			var parsed struct {
				Error   string `json:"error"`
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if json.Unmarshal(b, &parsed) == nil {
				if parsed.Code != "" {
					apiErr.Code = parsed.Code
				}
				if parsed.Error != "" {
					apiErr.Message = parsed.Error
				} else if parsed.Message != "" {
					apiErr.Message = parsed.Message
				}
			}
		}
		resp.Body.Close()
		return apiErr
	}
	return errors.New("request failed")
}

func (c *Client) requestJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if reqBody != nil {
		bodyBytes, _ := io.ReadAll(reqBody)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}
	return c.do(ctx, req, out)
}

func backoff(attempt int) time.Duration {
	return time.Duration(250*math.Pow(2, float64(attempt))) * time.Millisecond
}

// --- Transcription ----------------------------------------------------------

// Transcribe uploads an audio file (or fetches a URL) and returns the result.
// If opts.CallbackURL is set the API returns 202 and you should consume the
// AsyncAccepted struct (use TranscribeAsync). Use this method only for sync.
func (c *Client) Transcribe(ctx context.Context, opts TranscribeOptions) (*TranscriptionResult, error) {
	if len(opts.Audio) == 0 && opts.URL == "" {
		return nil, errors.New("Transcribe requires Audio or URL")
	}
	body, contentType, err := transcribeMultipart(opts)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/v1/transcribe", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	bodyBytes, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}

	var result TranscriptionResult
	if err := c.do(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TranscribeAsync submits a transcribe request with a callback URL and returns
// the AsyncAccepted handle. opts.CallbackURL must be set.
func (c *Client) TranscribeAsync(ctx context.Context, opts TranscribeOptions) (*AsyncAccepted, error) {
	if opts.CallbackURL == "" {
		return nil, errors.New("TranscribeAsync requires opts.CallbackURL")
	}
	body, contentType, err := transcribeMultipart(opts)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/v1/transcribe", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	bodyBytes, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}

	var result AsyncAccepted
	if err := c.do(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func transcribeMultipart(opts TranscribeOptions) (*bytes.Buffer, string, error) {
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	if len(opts.Audio) > 0 {
		name := opts.AudioName
		if name == "" {
			name = "audio.bin"
		}
		fw, err := mw.CreateFormFile("audio", name)
		if err != nil {
			return nil, "", err
		}
		if _, err := fw.Write(opts.Audio); err != nil {
			return nil, "", err
		}
	}
	add := func(k, v string) {
		if v != "" {
			_ = mw.WriteField(k, v)
		}
	}
	addBool := func(k string, v bool) {
		if v {
			_ = mw.WriteField(k, "true")
		}
	}
	add("url", opts.URL)
	add("engine", opts.Engine)
	add("language", opts.Language)
	add("format", opts.Format)
	add("callback_url", opts.CallbackURL)
	add("client_id", opts.ClientID)
	addBool("translate", opts.Translate)
	addBool("diarize", opts.Diarize)
	addBool("summarize", opts.Summarize)
	addBool("key_phrases", opts.KeyPhrases)
	addBool("action_items", opts.ActionItems)
	addBool("chapters", opts.Chapters)
	for k, v := range opts.Extra {
		add(k, v)
	}
	if err := mw.Close(); err != nil {
		return nil, "", err
	}
	return buf, mw.FormDataContentType(), nil
}

// GetTranscribeJob polls a transcription job by request_id.
func (c *Client) GetTranscribeJob(ctx context.Context, requestID string) (*JobStatus, error) {
	var s JobStatus
	if err := c.requestJSON(ctx, "GET", "/api/v1/transcribe/"+url.PathEscape(requestID), nil, nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// --- Translation ------------------------------------------------------------

func (c *Client) Translate(ctx context.Context, opts TranslateOptions) (*TranslationResult, error) {
	var r TranslationResult
	if err := c.requestJSON(ctx, "POST", "/api/v1/translate", nil, opts, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Client) GetTranslateJob(ctx context.Context, requestID string) (*JobStatus, error) {
	var s JobStatus
	if err := c.requestJSON(ctx, "GET", "/api/v1/translate/"+url.PathEscape(requestID), nil, nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// --- TTS --------------------------------------------------------------------

func (c *Client) TTS(ctx context.Context, opts TtsOptions) (*TtsResult, error) {
	var r TtsResult
	if err := c.requestJSON(ctx, "POST", "/api/v1/tts", nil, opts, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Client) GetTtsJob(ctx context.Context, requestID string) (*JobStatus, error) {
	var s JobStatus
	if err := c.requestJSON(ctx, "GET", "/api/v1/tts/"+url.PathEscape(requestID), nil, nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// --- Sentiment --------------------------------------------------------------

func (c *Client) Sentiment(ctx context.Context, opts SentimentOptions) (*SentimentResult, error) {
	var r SentimentResult
	if err := c.requestJSON(ctx, "POST", "/api/v1/sentiment", nil, opts, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Summarize --------------------------------------------------------------

func (c *Client) Summarize(ctx context.Context, opts SummarizeOptions) (*SummarizeResult, error) {
	if opts.Text == "" && opts.URL == "" {
		return nil, errors.New("Summarize requires Text or URL")
	}
	var r SummarizeResult
	if err := c.requestJSON(ctx, "POST", "/api/v1/summarize", nil, opts, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Embeddings -------------------------------------------------------------

func (c *Client) Embeddings(ctx context.Context, opts EmbeddingsOptions) (*EmbeddingsResult, error) {
	var r EmbeddingsResult
	if err := c.requestJSON(ctx, "POST", "/api/v1/embeddings", nil, opts, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Rerank -----------------------------------------------------------------

func (c *Client) Rerank(ctx context.Context, opts RerankOptions) (*RerankResult, error) {
	var r RerankResult
	if err := c.requestJSON(ctx, "POST", "/api/v1/rerank", nil, opts, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Moderate ---------------------------------------------------------------

func (c *Client) Moderate(ctx context.Context, opts ModerateOptions) (*ModerateResult, error) {
	var r ModerateResult
	if err := c.requestJSON(ctx, "POST", "/api/v1/moderate", nil, opts, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Detect language --------------------------------------------------------

func (c *Client) DetectLanguage(ctx context.Context, opts DetectLangOptions) (*DetectLangResult, error) {
	var r DetectLangResult
	if err := c.requestJSON(ctx, "POST", "/api/v1/detect-lang", nil, opts, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
