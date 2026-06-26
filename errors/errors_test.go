package errors

import "testing"

func TestStartsWithApiErrorPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"API Error: something", true},
		{"Please run /login · API Error: auth", true},
		{"some other error", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := StartsWithApiErrorPrefix(tt.input); got != tt.want {
			t.Errorf("StartsWithApiErrorPrefix(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParsePromptTooLongTokenCounts(t *testing.T) {
	t.Parallel()
	actual, limit := ParsePromptTooLongTokenCounts("prompt is too long: 137500 tokens > 135000 maximum")
	if actual == nil || *actual != 137500 {
		t.Errorf("expected actual=137500, got %v", actual)
	}
	if limit == nil || *limit != 135000 {
		t.Errorf("expected limit=135000, got %v", limit)
	}

	actual, limit = ParsePromptTooLongTokenCounts("no match here")
	if actual != nil || limit != nil {
		t.Error("expected nil for non-matching input")
	}
}

func TestIsMediaSizeError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"image is too large", true},
		{"pdf is too large", true},
		{"file is too large", true},
		{"request is too large", true},
		{"something else", false},
	}
	for _, tt := range tests {
		if got := IsMediaSizeError(tt.input); got != tt.want {
			t.Errorf("IsMediaSizeError(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestErrorMessageGetters(t *testing.T) {
	t.Parallel()
	if msg := GetPdfTooLargeErrorMessage(); msg == "" {
		t.Error("expected non-empty message")
	}
	if msg := GetImageTooLargeErrorMessage(); msg == "" {
		t.Error("expected non-empty message")
	}
	if msg := GetTokenRevokedErrorMessage(); msg == "" {
		t.Error("expected non-empty message")
	}
}
