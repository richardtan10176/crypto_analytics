package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	tradepb "github.com/richardtan10176/crypto_analytics/gen/trade"
)

type fakeAgg struct {
	updates  int
	flushes  []time.Time
	resets   int
	flushErr error
}

func (f *fakeAgg) Update(*tradepb.TradeEvent) { f.updates++ }

func (f *fakeAgg) Flush(_ context.Context, _ string, windowStart time.Time) error {
	if f.flushErr != nil {
		return f.flushErr
	}
	f.flushes = append(f.flushes, windowStart)
	return nil
}

func (f *fakeAgg) Reset() { f.resets++ }

func event(symbol string, tradeTime time.Time) *tradepb.TradeEvent {
	return &tradepb.TradeEvent{Symbol: symbol, TradeTime: tradeTime.UnixMilli()}
}

func newTestProcessor() (*StreamProcessor, *fakeAgg) {
	agg := &fakeAgg{}
	return NewStreamProcessor(func() []WindowAggregator {
		return []WindowAggregator{agg}
	}), agg
}

var minute0 = time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)

func TestSameWindowAccumulates(t *testing.T) {
	proc, agg := newTestProcessor()
	ctx := context.Background()

	proc.Process(ctx, event("BTCUSDT", minute0.Add(5*time.Second)))
	proc.Process(ctx, event("BTCUSDT", minute0.Add(40*time.Second)))

	if agg.updates != 2 {
		t.Errorf("updates = %d, want 2", agg.updates)
	}
	if len(agg.flushes) != 0 {
		t.Errorf("flushes = %d, want 0", len(agg.flushes))
	}
}

func TestNewWindowFlushesOld(t *testing.T) {
	proc, agg := newTestProcessor()
	ctx := context.Background()

	proc.Process(ctx, event("BTCUSDT", minute0.Add(5*time.Second)))
	proc.Process(ctx, event("BTCUSDT", minute0.Add(65*time.Second)))

	if len(agg.flushes) != 1 || !agg.flushes[0].Equal(minute0) {
		t.Errorf("flushes = %v, want [%v]", agg.flushes, minute0)
	}
	if agg.resets != 1 {
		t.Errorf("resets = %d, want 1", agg.resets)
	}
	if agg.updates != 2 {
		t.Errorf("updates = %d, want 2 (second event lands in new window)", agg.updates)
	}
}

func TestFlushErrorRetainsStateAndRetries(t *testing.T) {
	proc, agg := newTestProcessor()
	ctx := context.Background()

	proc.Process(ctx, event("BTCUSDT", minute0.Add(5*time.Second)))

	agg.flushErr = errors.New("db down")
	if err := proc.Process(ctx, event("BTCUSDT", minute0.Add(65*time.Second))); err == nil {
		t.Fatal("want error from failed flush")
	}
	if agg.resets != 0 {
		t.Errorf("resets = %d, want 0 after failed flush", agg.resets)
	}
	if agg.updates != 1 {
		t.Errorf("updates = %d, want 1 (event after failed flush not applied)", agg.updates)
	}

	agg.flushErr = nil
	if err := proc.Process(ctx, event("BTCUSDT", minute0.Add(70*time.Second))); err != nil {
		t.Fatalf("retry flush failed: %v", err)
	}
	if len(agg.flushes) != 1 || !agg.flushes[0].Equal(minute0) {
		t.Errorf("flushes = %v, want [%v] after retry", agg.flushes, minute0)
	}
	if agg.resets != 1 {
		t.Errorf("resets = %d, want 1 after successful retry", agg.resets)
	}
}

func TestLateEventDropped(t *testing.T) {
	proc, agg := newTestProcessor()
	ctx := context.Background()

	proc.Process(ctx, event("BTCUSDT", minute0.Add(5*time.Second)))
	if err := proc.Process(ctx, event("BTCUSDT", minute0.Add(-30*time.Second))); err != nil {
		t.Fatalf("late event returned error: %v", err)
	}

	if agg.updates != 1 {
		t.Errorf("updates = %d, want 1 (late event dropped)", agg.updates)
	}
	if len(agg.flushes) != 0 {
		t.Errorf("flushes = %d, want 0", len(agg.flushes))
	}
}
