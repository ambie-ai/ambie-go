# ambie-go

Official Go SDK for [AMBIE](https://ambie.ai).

Speech-to-text in noisy environments, translation, TTS, embeddings, sentiment, summarization, content moderation, and language detection — over a single typed client.

> **⚠ Platform preview.** AMBIE's API is live and the SDK surface is stable, but the inference is currently served by commodity off-the-shelf models (Deepgram, Whisper, Llama 3.1, BGE). The proprietary AMBIE acoustic-intelligence models — the 90-95% noisy-environment accuracy the brand is named after — are in development and will replace the transcription / TTS engines in 2027. Same API surface, free upgrade when it lands. Full disclosure at [ambie.ai/preview](https://ambie.ai/preview/). Every API response carries `X-AMBIE-Preview: true` until that swap.

## Install

```bash
go get github.com/ambie-ai/ambie-go
```

Requires Go 1.21+. Zero external dependencies (stdlib only).

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"os"

	ambie "github.com/ambie-ai/ambie-go"
)

func main() {
	c, err := ambie.New(os.Getenv("AMBIE_API_KEY"))
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// Transcribe a file
	audio, _ := os.ReadFile("meeting.mp3")
	r, err := c.Transcribe(ctx, ambie.TranscribeOptions{
		Audio:     audio,
		AudioName: "meeting.mp3",
		Engine:    "deepgram",
		Diarize:   true,
		Summarize: true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Text)
	fmt.Println(r.Summary)

	// Translate text
	t, _ := c.Translate(ctx, ambie.TranslateOptions{
		Text:       "Hello, world!",
		TargetLang: "es",
	})
	fmt.Println(t.TranslatedText)
}
```

## Async mode

```go
accepted, err := c.TranscribeAsync(ctx, ambie.TranscribeOptions{
	URL:         "https://cdn.example.com/long-recording.mp3",
	CallbackURL: "https://yourserver.com/webhooks/ambie",
})
fmt.Println(accepted.RequestID, accepted.PollURL)

// Or poll status manually:
status, _ := c.GetTranscribeJob(ctx, accepted.RequestID)
```

## Webhook verification

```go
import (
	ambie "github.com/ambie-ai/ambie-go"
)

func handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	err := ambie.VerifyWebhookSignature(ambie.VerifyWebhookOptions{
		Signature: r.Header.Get("X-Ambie-Signature"),
		Body:      body,
		Secret:    os.Getenv("AMBIE_WEBHOOK_SECRET"),
	})
	if err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}
	// safe to parse and act on body
}
```

## Configuration

```go
c, err := ambie.New(apiKey,
	ambie.WithBaseURL("https://staging.ambie.ai"),
	ambie.WithMaxRetries(5),
	ambie.WithUserAgent("my-app/1.0"),
	ambie.WithHTTPClient(&http.Client{Timeout: 90 * time.Second}),
)
```

## Error handling

```go
r, err := c.Transcribe(ctx, opts)
if err != nil {
	var apiErr *ambie.Error
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.Status, apiErr.Code, apiErr.Message, apiErr.RequestID)
	}
}
```

## Coverage

Supported endpoints: `Transcribe`, `Translate`, `TTS`, `Sentiment`, `Summarize`, `Embeddings`, `Rerank`, `Moderate`, `DetectLanguage`. Each has a sync method and async polling via `Get<Endpoint>Job(ctx, requestID)`.

See the full [OpenAPI 3.1 spec](https://ambie.ai/openapi.yaml).

## License

MIT © AMBIE
