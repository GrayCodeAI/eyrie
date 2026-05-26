package types

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkIsTransient_Nil(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(nil)
	}
}

func BenchmarkIsTransient_TransientError(b *testing.B) {
	err := &TransientError{StatusCode: 429, Message: "rate limited"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(err)
	}
}

func BenchmarkIsTransient_DeadlineExceeded(b *testing.B) {
	err := context.DeadlineExceeded
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(err)
	}
}

func BenchmarkIsTransient_TimeoutString(b *testing.B) {
	err := fmt.Errorf("connection timeout occurred")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(err)
	}
}

func BenchmarkIsTransient_NonRetriable(b *testing.B) {
	err := fmt.Errorf("unauthorized: invalid api key")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(err)
	}
}

func BenchmarkIsTransient_HTTPStatus500(b *testing.B) {
	err := fmt.Errorf("server error: HTTP 500 internal server error")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(err)
	}
}

func BenchmarkIsTransient_HTTPStatus401(b *testing.B) {
	err := fmt.Errorf("HTTP 401 unauthorized")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(err)
	}
}

func BenchmarkIsTransient_RateLimit(b *testing.B) {
	err := fmt.Errorf("rate limit exceeded, try again later")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(err)
	}
}

func BenchmarkIsTransient_ConnectionRefused(b *testing.B) {
	err := fmt.Errorf("dial tcp: connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(err)
	}
}

func BenchmarkIsTransient_UnknownError(b *testing.B) {
	err := fmt.Errorf("something completely unexpected happened")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsTransient(err)
	}
}

func BenchmarkExtractHTTPStatus_Found(b *testing.B) {
	err := fmt.Errorf("server returned HTTP 503 service unavailable")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractHTTPStatus(err)
	}
}

func BenchmarkExtractHTTPStatus_NotFound(b *testing.B) {
	err := fmt.Errorf("something went wrong")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractHTTPStatus(err)
	}
}
