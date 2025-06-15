package service

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"testing"
)

func mustKlineWithHighLow(close, high, low float64) vo.Kline {
	k, err := vo.NewKline(close, close, high, low, 10, 1234)
	if err != nil {
		panic(err)
	}
	return k
}

func TestBreakoutStrategy_Buy(t *testing.T) {
	klines := []vo.Kline{
		mustKlineWithHighLow(10, 11, 9),
		mustKlineWithHighLow(11, 12, 10),
		mustKlineWithHighLow(12, 13, 11),
		mustKlineWithHighLow(14, 14, 13),
	}
	strategy := NewBreakoutStrategy(3)
	decision := strategy.Decide(klines)
	if decision != entity.Buy {
		t.Fatalf("expected Buy, got %s", decision)
	}
}

func TestBreakoutStrategy_Sell(t *testing.T) {
	klines := []vo.Kline{
		mustKlineWithHighLow(14, 15, 13),
		mustKlineWithHighLow(13, 14, 12),
		mustKlineWithHighLow(12, 13, 11),
		mustKlineWithHighLow(10, 12, 9),
	}
	strategy := NewBreakoutStrategy(3)
	decision := strategy.Decide(klines)
	if decision != entity.Sell {
		t.Fatalf("expected Sell, got %s", decision)
	}
}

func TestBreakoutStrategy_Hold(t *testing.T) {
	klines := []vo.Kline{
		mustKlineWithHighLow(10, 12, 8),
		mustKlineWithHighLow(11, 13, 9),
		mustKlineWithHighLow(12, 14, 10),
		mustKlineWithHighLow(13, 13, 11),
	}
	strategy := NewBreakoutStrategy(3)
	decision := strategy.Decide(klines)
	if decision != entity.Hold {
		t.Fatalf("expected Hold, got %s", decision)
	}
}

func TestBreakoutStrategy_NotEnoughData(t *testing.T) {
	klines := []vo.Kline{
		mustKlineWithHighLow(10, 11, 9),
		mustKlineWithHighLow(11, 12, 10),
	}
	strategy := NewBreakoutStrategy(3)
	decision := strategy.Decide(klines)
	if decision != entity.Hold {
		t.Fatalf("expected Hold due to insufficient data, got %s", decision)
	}
}
