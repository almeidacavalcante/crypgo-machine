package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"time"
)

type BacktestResult struct {
	id                *vo.EntityId
	strategyName      string
	symbol            vo.Symbol
	startDate         time.Time
	endDate           time.Time
	initialCapital    float64
	finalCapital      float64
	totalProfitLoss   vo.ProfitLoss
	totalTrades       int
	winningTrades     int
	losingTrades      int
	winRate           vo.WinRate
	maxDrawdown       vo.Drawdown
	trades            []*BacktestTrade
	capitalHistory    []float64 // Capital at each time point
	createdAt         time.Time
}

func NewBacktestResult(
	strategyName string,
	symbol vo.Symbol,
	startDate, endDate time.Time,
	initialCapital float64,
	currency vo.Currency,
) (*BacktestResult, error) {
	id := vo.NewEntityId()
	
	zeroPL, err := vo.NewProfitLoss(0, &currency)
	if err != nil {
		return nil, err
	}
	
	zeroWinRate, err := vo.NewWinRate(0, 0)
	if err != nil {
		return nil, err
	}
	
	zeroDrawdown, err := vo.NewDrawdown(initialCapital, initialCapital, 0)
	if err != nil {
		return nil, err
	}
	
	return &BacktestResult{
		id:                id,
		strategyName:      strategyName,
		symbol:            symbol,
		startDate:         startDate,
		endDate:           endDate,
		initialCapital:    initialCapital,
		finalCapital:      initialCapital,
		totalProfitLoss:   zeroPL,
		totalTrades:       0,
		winningTrades:     0,
		losingTrades:      0,
		winRate:           zeroWinRate,
		maxDrawdown:       zeroDrawdown,
		trades:            make([]*BacktestTrade, 0),
		capitalHistory:    []float64{initialCapital},
		createdAt:         time.Now(),
	}, nil
}

func (br *BacktestResult) AddTrade(trade *BacktestTrade) error {
	if trade == nil {
		return fmt.Errorf("trade cannot be nil")
	}
	
	if trade.IsOpen() {
		return fmt.Errorf("cannot add open trade to backtest result")
	}
	
	br.trades = append(br.trades, trade)
	br.totalTrades++
	
	// Update P&L
	if trade.GetProfitLoss() != nil {
		newPL, err := br.totalProfitLoss.Add(*trade.GetProfitLoss())
		if err != nil {
			return err
		}
		br.totalProfitLoss = newPL
		
		// Update final capital
		br.finalCapital = br.initialCapital + br.totalProfitLoss.GetValue()
		
		// Update capital history
		br.capitalHistory = append(br.capitalHistory, br.finalCapital)
		
		// Update win/loss counts
		if trade.IsWinning() {
			br.winningTrades++
		} else if trade.GetProfitLoss().IsLoss() {
			br.losingTrades++
		}
		
		// Recalculate win rate
		winRate, err := vo.NewWinRate(br.winningTrades, br.totalTrades)
		if err != nil {
			return err
		}
		br.winRate = winRate
		
		// Update max drawdown
		err = br.updateMaxDrawdown()
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (br *BacktestResult) updateMaxDrawdown() error {
	if len(br.capitalHistory) < 2 {
		return nil
	}
	
	var maxPeak float64 = br.capitalHistory[0]
	var maxDrawdownValue float64 = 0
	var maxDrawdownDuration int = 0
	var currentDrawdownDuration int = 0
	var drawdownStartValue float64 = 0
	var drawdownEndValue float64 = 0
	
	for _, capital := range br.capitalHistory {
		if capital > maxPeak {
			maxPeak = capital
			currentDrawdownDuration = 0
		} else if capital < maxPeak {
			currentDrawdownDuration++
			drawdownValue := ((maxPeak - capital) / maxPeak) * 100
			
			if drawdownValue > maxDrawdownValue {
				maxDrawdownValue = drawdownValue
				maxDrawdownDuration = currentDrawdownDuration
				drawdownStartValue = maxPeak
				drawdownEndValue = capital
			}
		}
	}
	
	if maxDrawdownValue > 0 {
		drawdown, err := vo.NewDrawdown(drawdownStartValue, drawdownEndValue, maxDrawdownDuration)
		if err != nil {
			return err
		}
		br.maxDrawdown = drawdown
	}
	
	return nil
}

// Getters
func (br *BacktestResult) GetId() *vo.EntityId {
	return br.id
}

func (br *BacktestResult) GetStrategyName() string {
	return br.strategyName
}

func (br *BacktestResult) GetSymbol() vo.Symbol {
	return br.symbol
}

func (br *BacktestResult) GetStartDate() time.Time {
	return br.startDate
}

func (br *BacktestResult) GetEndDate() time.Time {
	return br.endDate
}

func (br *BacktestResult) GetInitialCapital() float64 {
	return br.initialCapital
}

func (br *BacktestResult) GetFinalCapital() float64 {
	return br.finalCapital
}

func (br *BacktestResult) GetTotalProfitLoss() vo.ProfitLoss {
	return br.totalProfitLoss
}

func (br *BacktestResult) GetTotalTrades() int {
	return br.totalTrades
}

func (br *BacktestResult) GetWinningTrades() int {
	return br.winningTrades
}

func (br *BacktestResult) GetLosingTrades() int {
	return br.losingTrades
}

func (br *BacktestResult) GetWinRate() vo.WinRate {
	return br.winRate
}

func (br *BacktestResult) GetMaxDrawdown() vo.Drawdown {
	return br.maxDrawdown
}

func (br *BacktestResult) GetTrades() []*BacktestTrade {
	return br.trades
}

func (br *BacktestResult) GetCapitalHistory() []float64 {
	return br.capitalHistory
}

func (br *BacktestResult) GetCreatedAt() time.Time {
	return br.createdAt
}

func (br *BacktestResult) GetROI() float64 {
	if br.initialCapital == 0 {
		return 0
	}
	return ((br.finalCapital - br.initialCapital) / br.initialCapital) * 100
}