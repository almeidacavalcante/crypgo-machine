package service

import (
	"testing"

	"crypgo-machine/src/domain/entity"
	vo "crypgo-machine/src/domain/vo"
)

func mustKline(close float64) vo.Kline {
	open := close - 0.5
	high := close + 0.5
	low := close - 1
	volume := 10.0
	closeTime := 1000
	k, err := vo.NewKline(open, close, high, low, volume, int64(closeTime))
	if err != nil {
		panic(err)
	}
	return k
}

func TestMovingAverageStrategy_Buy(t *testing.T) {
	klines := []vo.Kline{
		mustKline(9), mustKline(9), mustKline(10), mustKline(10), mustKline(10),
	}

	strategy := NewMovingAverageStrategy(3, 5)
	decision := strategy.Decide(klines)
	if decision != entity.Buy {
		t.Fatalf("expected Buy, got %s", decision)
	}
}

func TestMovingAverageStrategy_Sell(t *testing.T) {
	klines := []vo.Kline{
		mustKline(10), mustKline(9), mustKline(8), mustKline(8), mustKline(8),
	}

	strategy := NewMovingAverageStrategy(3, 5)
	decision := strategy.Decide(klines)
	if decision != entity.Sell {
		t.Fatalf("expected Sell, got %s", decision)
	}
}

func TestMovingAverageStrategy_Hold(t *testing.T) {
	klines := []vo.Kline{
		mustKline(10), mustKline(10), mustKline(10), mustKline(10), mustKline(10),
	}

	strategy := NewMovingAverageStrategy(3, 5)
	decision := strategy.Decide(klines)
	if decision != entity.Hold {
		t.Fatalf("expected Hold, got %s", decision)
	}
}

func TestMovingAverageStrategy_NotEnoughData(t *testing.T) {
	klines := []vo.Kline{
		mustKline(10), mustKline(11),
	}

	strategy := NewMovingAverageStrategy(3, 5)
	decision := strategy.Decide(klines)
	if decision != entity.Hold {
		t.Fatalf("expected Hold due to insufficient data, got %s", decision)
	}
}
