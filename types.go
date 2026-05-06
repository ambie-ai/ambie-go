package ambie

// AsyncAccepted is the response when a callback_url is provided.
type AsyncAccepted struct {
	RequestID string `json:"request_id"`
	PollURL   string `json:"poll_url"`
	Status    string `json:"status"`
	ClientID  string `json:"client_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// JobStatus is the response from a status polling endpoint.
type JobStatus struct {
	RequestID   string         `json:"request_id"`
	Status      string         `json:"status"`
	ClientID    string         `json:"client_id,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	CompletedAt string         `json:"completed_at,omitempty"`
}

// --- Transcription ----------------------------------------------------------

// TranscribeOptions configures a transcription request. Either Audio or URL
// is required. Audio takes a filename and the raw bytes.
type TranscribeOptions struct {
	Audio       []byte
	AudioName   string // optional; defaults to "audio.bin"
	URL         string
	Engine      string // "deepgram" (default) or "whisper"
	Language    string
	Format      string // "json" (default), "text", "srt", "vtt"
	Translate   bool
	CallbackURL string
	ClientID    string
	Diarize     bool
	Summarize   bool
	KeyPhrases  bool
	ActionItems bool
	Chapters    bool
	// Extra holds any other fields documented in openapi.yaml
	// (punctuate, smart_format, multichannel, vocabulary, etc.).
	Extra map[string]string
}

type Word struct {
	Word       string  `json:"word"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float64 `json:"confidence,omitempty"`
	Speaker    int     `json:"speaker,omitempty"`
	Channel    int     `json:"channel,omitempty"`
}

type Utterance struct {
	Text       string  `json:"text"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float64 `json:"confidence,omitempty"`
	Speaker    int     `json:"speaker,omitempty"`
}

type ActionItem struct {
	Action   string `json:"action"`
	Assignee string `json:"assignee,omitempty"`
	Deadline string `json:"deadline,omitempty"`
}

type Chapter struct {
	Title   string  `json:"title"`
	Summary string  `json:"summary"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
}

type TranscriptionResult struct {
	RequestID    string       `json:"request_id"`
	ClientID     string       `json:"client_id,omitempty"`
	Engine       string       `json:"engine"`
	Language     string       `json:"language,omitempty"`
	Duration     float64      `json:"duration"`
	Text         string       `json:"text"`
	Confidence   float64      `json:"confidence,omitempty"`
	Words        []Word       `json:"words,omitempty"`
	Utterances   []Utterance  `json:"utterances,omitempty"`
	DiarizedText string       `json:"diarized_text,omitempty"`
	SpeakerCount int          `json:"speaker_count,omitempty"`
	Paragraphs   []string     `json:"paragraphs,omitempty"`
	Summary      string       `json:"summary,omitempty"`
	KeyPhrases   []string     `json:"key_phrases,omitempty"`
	ActionItems  []ActionItem `json:"action_items,omitempty"`
	Chapters     []Chapter    `json:"chapters,omitempty"`
}

// --- Translation ------------------------------------------------------------

type TranslateOptions struct {
	Text        string `json:"text"`
	TargetLang  string `json:"target_lang"`
	SourceLang  string `json:"source_lang,omitempty"`
	Formality   string `json:"formality,omitempty"`
	Context     string `json:"context,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
}

type TranslationResult struct {
	RequestID      string  `json:"request_id"`
	ClientID       string  `json:"client_id,omitempty"`
	SourceLang     string  `json:"source_lang"`
	TargetLang     string  `json:"target_lang"`
	SourceText     string  `json:"source_text"`
	TranslatedText string  `json:"translated_text"`
	Confidence     float64 `json:"confidence,omitempty"`
}

// --- TTS --------------------------------------------------------------------

type TtsOptions struct {
	Text        string  `json:"text"`
	Voice       string  `json:"voice,omitempty"`
	Engine      string  `json:"engine,omitempty"`
	Language    string  `json:"language,omitempty"`
	Encoding    string  `json:"encoding,omitempty"`
	Container   string  `json:"container,omitempty"`
	Speed       float64 `json:"speed,omitempty"`
	CallbackURL string  `json:"callback_url,omitempty"`
	ClientID    string  `json:"client_id,omitempty"`
}

type TtsResult struct {
	RequestID string  `json:"request_id"`
	ClientID  string  `json:"client_id,omitempty"`
	AudioURL  string  `json:"audio_url"`
	Duration  float64 `json:"duration,omitempty"`
	Voice     string  `json:"voice"`
	Encoding  string  `json:"encoding"`
}

// --- Sentiment --------------------------------------------------------------

type SentimentOptions struct {
	Text        any    `json:"text"` // string or []string
	CallbackURL string `json:"callback_url,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
}

type SentimentItem struct {
	Text  string  `json:"text"`
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

type SentimentResult struct {
	RequestID string          `json:"request_id"`
	ClientID  string          `json:"client_id,omitempty"`
	Results   []SentimentItem `json:"results"`
}

// --- Summarize --------------------------------------------------------------

type SummarizeOptions struct {
	Text        string `json:"text,omitempty"`
	URL         string `json:"url,omitempty"`
	MaxLength   int    `json:"max_length,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
}

type SummarizeResult struct {
	RequestID     string `json:"request_id"`
	ClientID      string `json:"client_id,omitempty"`
	Summary       string `json:"summary"`
	SourceLength  int    `json:"source_length,omitempty"`
	SummaryLength int    `json:"summary_length,omitempty"`
}

// --- Embeddings -------------------------------------------------------------

type EmbeddingsOptions struct {
	Text        any    `json:"text"` // string or []string
	Model       string `json:"model,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
}

type EmbeddingsResult struct {
	RequestID  string      `json:"request_id"`
	ClientID   string      `json:"client_id,omitempty"`
	Model      string      `json:"model"`
	Dimensions int         `json:"dimensions"`
	Embeddings [][]float64 `json:"embeddings"`
}

// --- Rerank -----------------------------------------------------------------

type RerankOptions struct {
	Query       string   `json:"query"`
	Documents   []string `json:"documents"`
	TopK        int      `json:"top_k,omitempty"`
	CallbackURL string   `json:"callback_url,omitempty"`
	ClientID    string   `json:"client_id,omitempty"`
}

type RerankItem struct {
	Index    int     `json:"index"`
	Score    float64 `json:"score"`
	Document string  `json:"document"`
}

type RerankResult struct {
	RequestID string       `json:"request_id"`
	ClientID  string       `json:"client_id,omitempty"`
	Results   []RerankItem `json:"results"`
}

// --- Moderate ---------------------------------------------------------------

type ModerateOptions struct {
	Text        string `json:"text"`
	CallbackURL string `json:"callback_url,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
}

type ModerateResult struct {
	RequestID  string             `json:"request_id"`
	ClientID   string             `json:"client_id,omitempty"`
	Flagged    bool               `json:"flagged"`
	Categories map[string]bool    `json:"categories"`
	Scores     map[string]float64 `json:"scores,omitempty"`
}

// --- Detect language --------------------------------------------------------

type DetectLangOptions struct {
	Text        string `json:"text"`
	CallbackURL string `json:"callback_url,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
}

type DetectLangAlternative struct {
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
}

type DetectLangResult struct {
	RequestID    string                  `json:"request_id"`
	ClientID     string                  `json:"client_id,omitempty"`
	Language     string                  `json:"language"`
	Confidence   float64                 `json:"confidence"`
	Alternatives []DetectLangAlternative `json:"alternatives,omitempty"`
}
