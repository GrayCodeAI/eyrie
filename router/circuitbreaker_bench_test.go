package router

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkCircuitBreaker_Allow_Closed(b *testing.B) {
	cb := NewCircuitBreaker(5, 30*time.Second)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Allow()
	}
}

func BenchmarkCircuitBreaker_Success(b *testing.B) {
	cb := NewCircuitBreaker(5, 30*time.Second)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Success()
	}
}

func BenchmarkCircuitBreaker_Failure(b *testing.B) {
	cb := NewCircuitBreaker(1000, 30*time.Second) // high threshold to stay closed
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Failure()
	}
}

func BenchmarkCircuitBreaker_Allow_Open(b *testing.B) {
	cb := NewCircuitBreaker(1, 1*time.Hour) // opens after 1 failure, long cooldown
	cb.Failure()                             // open it
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Allow()
	}
}

func BenchmarkCircuitBreaker_Contention(b *testing.B) {
	cb := NewCircuitBreaker(100, 30*time.Second)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if cb.Allow() {
				if i%10 == 0 {
					cb.Failure()
				} else {
					cb.Success()
				}
			}
			i++
		}
	})
}

func BenchmarkCircuitBreaker_FullCycle(b *testing.B) {
	cb := NewCircuitBreaker(5, 1*time.Millisecond)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Allow()
		cb.Success()
	}
}

func BenchmarkCircuitBreaker_WithLatency(b *testing.B) {
	cb := NewCircuitBreaker(5, 30*time.Second)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if cb.Allow() {
			// Simulate minimal work
			_ = fmt.Sprintf("request-%d", i)
			cb.Success()
		}
	}
}
