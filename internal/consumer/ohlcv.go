package consumer

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tradepb "github.com/richardtan10176/crypto_analytics/gen/trade"
)

type OHLCVAggregator struct {
	open        float64
	high        float64
	low         float64
	close       float64
	volume      float64
	makerVolume float64
	takerVolume float64
	vwapNum     float64
	vwapDenom   float64
	tradeCount  int64
	db          *pgxpool.Pool
}

func NewOHLCVAggregator(db *pgxpool.Pool) *OHLCVAggregator {
	return &OHLCVAggregator{db: db}
}

func (a *OHLCVAggregator) Update(event *tradepb.TradeEvent) {
	price, err := strconv.ParseFloat(event.Price, 64)
	if err != nil {
		return
	}
	qnt, err := strconv.ParseFloat(event.Quantity, 64)
	if err != nil {
		return
	}
	if a.tradeCount == 0 {
		a.open = price
		a.high = price
		a.low = price
	}
	a.close = price

	if price > a.high {
		a.high = price
	}
	if price < a.low {
		a.low = price
	}

	a.volume += qnt
	if event.IsMaker == true {
		a.makerVolume += qnt
	} else {
		a.takerVolume += qnt
	}

	a.tradeCount++

	a.vwapNum += price * qnt
	a.vwapDenom += qnt

}

// Flush writes the candle as a full-row upsert. Replacing every column (rather
// than merging high/low and adding volumes) keeps the write idempotent under
// at-least-once redelivery: the consumer re-aggregates the whole window in
// memory, so the incoming row is always complete.
func (a *OHLCVAggregator) Flush(ctx context.Context, symbol string, windowStart time.Time) error {
	if a.tradeCount == 0 {
		return nil
	}
	vwap := a.vwapNum / a.vwapDenom

	_, err := a.db.Exec(ctx, `
		INSERT INTO ohlcv_1m (symbol, window_start, open, high, low, close, volume,
			maker_volume, taker_volume, vwap, trade_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (symbol, window_start) DO UPDATE SET
			open = EXCLUDED.open, high = EXCLUDED.high, low = EXCLUDED.low,
			close = EXCLUDED.close, volume = EXCLUDED.volume,
			maker_volume = EXCLUDED.maker_volume, taker_volume = EXCLUDED.taker_volume,
			vwap = EXCLUDED.vwap, trade_count = EXCLUDED.trade_count`,
		symbol, windowStart, a.open, a.high, a.low, a.close, a.volume,
		a.makerVolume, a.takerVolume, vwap, a.tradeCount)
	return err
}

func (a *OHLCVAggregator) Reset() {
	a.open = 0
	a.high = 0
	a.low = 0
	a.close = 0
	a.volume = 0
	a.makerVolume = 0
	a.takerVolume = 0
	a.vwapNum = 0
	a.vwapDenom = 0
	a.tradeCount = 0
}
