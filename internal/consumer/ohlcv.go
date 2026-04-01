package consumer

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tradepb "github.com/richard-crypto/crypto_analytics/gen/trade"
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

func newOHLCVAggregator(db *pgxpool.Pool) *OHLCVAggregator {
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

func (a *OHLCVAggregator) Flush(ctx context.Context, symbol string, windowStart time.Time) error {
	// TODO: implement DB write

	return nil
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
