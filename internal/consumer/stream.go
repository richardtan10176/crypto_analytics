package consumer

import (
	"context"
	"time"

	tradepb "github.com/richardtan10176/crypto_analytics/gen/trade"
)

// WindowAggregator accumulates trades for one symbol over one tumbling window.
type WindowAggregator interface {
	Update(event *tradepb.TradeEvent)
	Flush(ctx context.Context, symbol string, windowStart time.Time) error
	Reset()
}

type window struct {
	start time.Time
	aggs  []WindowAggregator
}

// StreamProcessor buckets trades into per-symbol 1-minute tumbling windows and
// fans each event out to all registered aggregators. It is not safe for
// concurrent use; the consumer loop drives it from a single goroutine.
type StreamProcessor struct {
	windows map[string]*window
	factory func() []WindowAggregator
}

func NewStreamProcessor(factory func() []WindowAggregator) *StreamProcessor {
	return &StreamProcessor{
		windows: make(map[string]*window),
		factory: factory,
	}
}

// Process routes an event into its symbol's window. An event in a newer window
// flushes the old one first; late events (an older window) are dropped. A
// symbol that goes idle keeps its last window open until the next trade or
// FlushAll on shutdown.
func (s *StreamProcessor) Process(ctx context.Context, event *tradepb.TradeEvent) error {
	bucket := time.UnixMilli(event.TradeTime).UTC().Truncate(time.Minute)

	w, ok := s.windows[event.Symbol]
	if !ok {
		w = &window{start: bucket, aggs: s.factory()}
		s.windows[event.Symbol] = w
	}

	if bucket.Before(w.start) {
		return nil
	}

	if bucket.After(w.start) {
		if err := s.flushWindow(ctx, event.Symbol, w); err != nil {
			return err
		}
		w.start = bucket
	}

	for _, agg := range w.aggs {
		agg.Update(event)
	}
	return nil
}

// FlushAll flushes every open window; called once on shutdown.
func (s *StreamProcessor) FlushAll(ctx context.Context) error {
	for symbol, w := range s.windows {
		if err := s.flushWindow(ctx, symbol, w); err != nil {
			return err
		}
	}
	return nil
}

// flushWindow flushes then resets each aggregator. Reset only happens after a
// successful Flush so a failed write retains its state and retries next time.
func (s *StreamProcessor) flushWindow(ctx context.Context, symbol string, w *window) error {
	for _, agg := range w.aggs {
		if err := agg.Flush(ctx, symbol, w.start); err != nil {
			return err
		}
		agg.Reset()
	}
	return nil
}
