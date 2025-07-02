package usecase

import (
	"context"
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"fmt"
	"github.com/adshao/go-binance/v2"
	"strconv"
	"time"
)

type StartTradingBotUseCase struct {
	tradingBotRepository         repository.TradingBotRepository
	tradingDecisionLogRepository repository.TradingDecisionLogRepository
	client                       external.BinanceClientInterface
}

func NewStartTradingBotUseCase(
	tradingBotRepo repository.TradingBotRepository,
	decisionLogRepo repository.TradingDecisionLogRepository,
	client external.BinanceClientInterface,
) *StartTradingBotUseCase {
	return &StartTradingBotUseCase{
		tradingBotRepository:         tradingBotRepo,
		tradingDecisionLogRepository: decisionLogRepo,
		client:                       client,
	}
}

type InputStartTradingBot struct {
	TradingBotId string
}

func (uc *StartTradingBotUseCase) Execute(input InputStartTradingBot) error {
	tradingBot, err := uc.tradingBotRepository.GetTradeByID(input.TradingBotId)
	if err != nil {
		return err
	}
	if tradingBot == nil {
		return fmt.Errorf("trading bot not found")
	}

	errStart := tradingBot.Start()
	if errStart != nil {
		return errStart
	}

	errSave := uc.tradingBotRepository.Update(tradingBot)
	if errSave != nil {
		return errSave
	}

	go uc.runStrategyLoop(tradingBot)

	return nil
}

func (uc *StartTradingBotUseCase) runStrategyLoop(tradingBot *entity.TradingBot) {
	// Execute first analysis immediately
	uc.executeAnalysisAndTrade(tradingBot)
	
	// Then start the ticker for subsequent executions
	ticker := time.NewTicker(time.Duration(tradingBot.GetIntervalSeconds()) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		uc.executeAnalysisAndTrade(tradingBot)
	}
}

func (uc *StartTradingBotUseCase) executeAnalysisAndTrade(tradingBot *entity.TradingBot) {
	currentBot, err := uc.tradingBotRepository.GetTradeByID(tradingBot.Id.GetValue())
	if err != nil || currentBot == nil || currentBot.GetStatus() != entity.StatusRunning {
		return
	}

	// Fetch market data from Binance
	klines, err := uc.getMarketData(tradingBot.GetSymbol().GetValue())
	if err != nil {
		fmt.Printf("❌ Error fetching market data for %s: %v\n", tradingBot.GetSymbol().GetValue(), err)
		return
	}

	strategy := tradingBot.GetStrategy()
	analysisResult := strategy.Decide(klines, tradingBot)

	// Create and save decision log
	currentPrice := klines[len(klines)-1].Close()
	decisionLog := entity.NewTradingDecisionLog(
		tradingBot.Id,
		analysisResult.Decision,
		strategy.GetName(),
		analysisResult.AnalysisData,
		klines,
		currentPrice,
	)

	if err := uc.tradingDecisionLogRepository.Save(decisionLog); err != nil {
		fmt.Printf("⚠️ Failed to save decision log: %v\n", err)
	}

	fmt.Printf("🤖 Strategy %s for bot %s decided: %s (analysis: %+v)\n",
		strategy.GetName(), tradingBot.Id.GetValue(), analysisResult.Decision, analysisResult.AnalysisData)

	// Execute trading decision
	uc.executeTradingDecision(tradingBot, analysisResult.Decision)
}

func (uc *StartTradingBotUseCase) getMarketData(symbol string) ([]vo.Kline, error) {
	binanceKlines, err := uc.client.NewKlinesService().
		Symbol(symbol).
		Interval("1h").
		Limit(100).
		Do(context.Background())

	if err != nil {
		return nil, err
	}

	klines := make([]vo.Kline, len(binanceKlines))
	for i, bkline := range binanceKlines {
		openPrice, _ := strconv.ParseFloat(bkline.Open, 64)
		closePrice, _ := strconv.ParseFloat(bkline.Close, 64)
		highPrice, _ := strconv.ParseFloat(bkline.High, 64)
		lowPrice, _ := strconv.ParseFloat(bkline.Low, 64)
		volumePrice, _ := strconv.ParseFloat(bkline.Volume, 64)

		kline, err := vo.NewKline(openPrice, closePrice, highPrice, lowPrice, volumePrice, bkline.CloseTime)
		if err != nil {
			return nil, err
		}
		klines[i] = kline
	}

	return klines, nil
}

func (uc *StartTradingBotUseCase) executeTradingDecision(tradingBot *entity.TradingBot, decision entity.TradingDecision) error {
	symbol := tradingBot.GetSymbol().GetValue()
	quantity := tradingBot.GetQuantity()

	switch decision {
	case entity.Buy:
		if tradingBot.GetIsPositioned() {
			return fmt.Errorf("this trading bot already has an open position")
		}
		fmt.Printf("🟢 Executing BUY order for %s, quantity: %.6f\n", symbol, quantity)
		isOrderPlaced := uc.placeBuyOrder(symbol, quantity)
		if isOrderPlaced {
			errPosition := tradingBot.GetIntoPosition()
			if errPosition != nil {
				return errPosition
			}
			errUpdate := uc.tradingBotRepository.Update(tradingBot)
			if errUpdate != nil {
				return errUpdate
			}
		}
		return nil
	case entity.Sell:
		if !tradingBot.GetIsPositioned() {
			return fmt.Errorf("this trading bot don't have an open position")
		}
		fmt.Printf("🔴 Executing SELL order for %s, quantity: %.6f\n", symbol, quantity)
		isOrderPlaced := uc.placeSellOrder(symbol, quantity)
		if isOrderPlaced {
			errPosition := tradingBot.GetOutOfPosition()
			if errPosition != nil {
				return errPosition
			}
			errUpdate := uc.tradingBotRepository.Update(tradingBot)
			if errUpdate != nil {
				return errUpdate
			}
		}
		return nil
	case entity.Hold:
		fmt.Printf("⏸ HOLDING position for %s\n", symbol)
	}

	return nil
}

func (uc *StartTradingBotUseCase) placeBuyOrder(symbol string, quantity float64) bool {
	qtyStr := strconv.FormatFloat(quantity, 'f', 6, 64)

	order, err := uc.client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeMarket).
		Quantity(qtyStr).
		Do(context.Background())

	if err != nil {
		fmt.Printf("❌ Error placing buy order: %v\n", err)
		return false
	}

	fmt.Printf("✅ Buy order placed: %+v\n", order)
	return true
}

func (uc *StartTradingBotUseCase) placeSellOrder(symbol string, quantity float64) bool {
	qtyStr := strconv.FormatFloat(quantity, 'f', 6, 64)

	order, err := uc.client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell).
		Type(binance.OrderTypeMarket).
		Quantity(qtyStr).
		Do(context.Background())

	if err != nil {
		fmt.Printf("❌ Error placing sell order: %v\n", err)
		return false
	}

	fmt.Printf("✅ Sell order placed: %+v\n", order)
	return true
}
