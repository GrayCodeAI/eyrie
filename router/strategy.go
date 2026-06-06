package router

import (
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
)

// Strategy names a load-balancing routing strategy used to pick an entry from
// the router's primary entries. StrategyWeighted is the historical default and
// preserves the original weighted-random behavior.
type Strategy string

const (
	// StrategyWeighted selects an entry via weighted random over RouteEntry.Weight.
	// This is the default and matches the router's original behavior.
	StrategyWeighted Strategy = "weighted"
	// StrategySimpleShuffle selects uniformly at random among entries, ignoring weight.
	StrategySimpleShuffle Strategy = "simple-shuffle"
	// StrategyLeastBusy selects the entry with the fewest in-flight requests.
	StrategyLeastBusy Strategy = "least-busy"
	// StrategyLatencyBased selects the entry with the lowest observed EWMA latency.
	StrategyLatencyBased Strategy = "latency-based"
	// StrategyCostBased selects the entry with the lowest cost (RouteEntry.Cost,
	// falling back to Weight as a proxy when no cost is configured).
	StrategyCostBased Strategy = "cost-based"
	// StrategyUsageBased selects the entry with the least cumulative token usage
	// in the current window.
	StrategyUsageBased Strategy = "usage-based"
)

// ewmaAlpha is the smoothing factor for the latency EWMA. Higher values weight
// recent samples more heavily.
const ewmaAlpha = 0.3

// strategyState holds per-provider runtime telemetry used by the dynamic
// strategies (least-busy, latency-based, usage-based). It is keyed by provider
// name and lazily populated as entries are observed.
type strategyState struct {
	inFlight map[string]*atomic.Int64 // current in-flight request count per provider
	usage    map[string]*atomic.Int64 // cumulative tokens this window per provider

	latencyMu sync.RWMutex
	latencyMs map[string]float64 // EWMA latency in milliseconds per provider
}

func newStrategyState(entries []RouteEntry) *strategyState {
	s := &strategyState{
		inFlight:  make(map[string]*atomic.Int64, len(entries)),
		usage:     make(map[string]*atomic.Int64, len(entries)),
		latencyMs: make(map[string]float64, len(entries)),
	}
	for _, e := range entries {
		name := e.Provider.Name()
		if _, ok := s.inFlight[name]; !ok {
			s.inFlight[name] = &atomic.Int64{}
			s.usage[name] = &atomic.Int64{}
		}
	}
	return s
}

// beginInFlight increments the in-flight counter for name.
func (s *strategyState) beginInFlight(name string) {
	if c, ok := s.inFlight[name]; ok {
		c.Add(1)
	}
}

// endInFlight decrements the in-flight counter for name.
func (s *strategyState) endInFlight(name string) {
	if c, ok := s.inFlight[name]; ok {
		c.Add(-1)
	}
}

// recordLatency folds a new latency sample (in milliseconds) into the EWMA for name.
func (s *strategyState) recordLatency(name string, ms float64) {
	s.latencyMu.Lock()
	defer s.latencyMu.Unlock()
	prev, ok := s.latencyMs[name]
	if !ok {
		s.latencyMs[name] = ms
		return
	}
	s.latencyMs[name] = ewmaAlpha*ms + (1-ewmaAlpha)*prev
}

// latency returns the current EWMA latency for name, or +Inf if no sample has
// been recorded yet (so unmeasured providers are tried before measured-slow ones
// only after all have at least one sample; before that they sort as "unknown").
func (s *strategyState) latency(name string) (float64, bool) {
	s.latencyMu.RLock()
	defer s.latencyMu.RUnlock()
	v, ok := s.latencyMs[name]
	return v, ok
}

// recordUsage adds tokens to the cumulative usage counter for name.
func (s *strategyState) recordUsage(name string, tokens int64) {
	if c, ok := s.usage[name]; ok {
		c.Add(tokens)
	}
}

// selectIndex picks an entry index according to strat. It never returns an index
// outside [0, len(entries)). Callers must hold no router lock; this method only
// reads immutable entry data plus the atomic/locked telemetry in strategyState.
func (s *strategyState) selectIndex(strat Strategy, entries []RouteEntry, totalWeight int) int {
	switch strat {
	case StrategySimpleShuffle:
		return rand.IntN(len(entries))

	case StrategyLeastBusy:
		best, bestVal := 0, int64(math.MaxInt64)
		for i, e := range entries {
			v := int64(0)
			if c, ok := s.inFlight[e.Provider.Name()]; ok {
				v = c.Load()
			}
			if v < bestVal {
				best, bestVal = i, v
			}
		}
		return best

	case StrategyLatencyBased:
		// Prefer entries without a sample yet (treated as latency 0) so every
		// provider is probed at least once, then favor the lowest EWMA.
		best, bestVal := 0, math.Inf(1)
		for i, e := range entries {
			lat, ok := s.latency(e.Provider.Name())
			if !ok {
				lat = 0
			}
			if lat < bestVal {
				best, bestVal = i, lat
			}
		}
		return best

	case StrategyCostBased:
		best, bestVal := 0, math.MaxInt64
		for i, e := range entries {
			if e.cost() < bestVal {
				best, bestVal = i, e.cost()
			}
		}
		return best

	case StrategyUsageBased:
		best, bestVal := 0, int64(math.MaxInt64)
		for i, e := range entries {
			v := int64(0)
			if c, ok := s.usage[e.Provider.Name()]; ok {
				v = c.Load()
			}
			if v < bestVal {
				best, bestVal = i, v
			}
		}
		return best

	default: // StrategyWeighted
		return weightedIndex(entries, totalWeight)
	}
}

// weightedIndex returns an entry index chosen by weighted random over Weight.
// When totalWeight is zero it returns 0 (the first entry).
func weightedIndex(entries []RouteEntry, totalWeight int) int {
	if totalWeight == 0 {
		return 0
	}
	n := rand.IntN(totalWeight)
	cumulative := 0
	for i, e := range entries {
		cumulative += e.Weight
		if n < cumulative {
			return i
		}
	}
	return len(entries) - 1
}
