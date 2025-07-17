package api

import (
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type BacktestStrategyController struct {
	backtestUseCase       *usecase.BacktestStrategyUseCase
	historicalDataService *external.BinanceHistoricalDataService
}

func NewBacktestStrategyController(backtestUseCase *usecase.BacktestStrategyUseCase, historicalDataService *external.BinanceHistoricalDataService) *BacktestStrategyController {
	return &BacktestStrategyController{
		backtestUseCase:       backtestUseCase,
		historicalDataService: historicalDataService,
	}
}

type BacktestRequest struct {
	StrategyName            string                 `json:"strategy_name"`
	Symbol                  string                 `json:"symbol"`
	Params                  map[string]interface{} `json:"params,omitempty"`                 // Strategy parameters (e.g., FastWindow, SlowWindow for MovingAverage)
	HistoricalData          []HistoricalKline      `json:"historical_data,omitempty"`        // Optional for manual data
	InitialCapital          float64                `json:"initial_capital"`
	TradeAmount             float64                `json:"trade_amount,omitempty"`           // Fixed amount per trade (optional, 0 = use all capital)
	Currency                string                 `json:"currency"`
	StartDate               string                 `json:"start_date,omitempty"`             // Optional - will use yesterday if not provided
	EndDate                 string                 `json:"end_date,omitempty"`               // Optional - will use yesterday if not provided
	TradingFees             float64                `json:"trading_fees"`                     // Percentage (e.g., 0.1 for 0.1%)
	MinimumProfitThreshold  float64                `json:"minimum_profit_threshold,omitempty"` // Minimum profit % to sell (default: 0 = sell at any profit)
	UseYesterday            bool                   `json:"use_yesterday,omitempty"`          // If true, fetch yesterday's data from Binance
	UseLastWeek             bool                   `json:"use_last_week,omitempty"`          // If true, fetch last week's data from Binance
	UseBinanceData          bool                   `json:"use_binance_data,omitempty"`       // If true, fetch data from start_date to today
	Interval                string                 `json:"interval,omitempty"`               // Interval for Binance data (1m, 30m, 1h, 4h, 1d)
}

// YesterdayBacktestRequest is a simplified request for yesterday's data
type YesterdayBacktestRequest struct {
	StrategyName   string  `json:"strategy_name"`
	Symbol         string  `json:"symbol"`
	InitialCapital float64 `json:"initial_capital"`
	Currency       string  `json:"currency"`
	TradingFees    float64 `json:"trading_fees"` // Percentage (e.g., 0.1 for 0.1%)
}

type HistoricalKline struct {
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	CloseTime int64   `json:"close_time"` // Unix timestamp in milliseconds
}

type BacktestResponse struct {
	Success bool                `json:"success"`
	Data    *BacktestResultData `json:"data,omitempty"`
	Error   string              `json:"error,omitempty"`
}

type BacktestResultData struct {
	ID              string              `json:"id"`
	StrategyName    string              `json:"strategy_name"`
	Symbol          string              `json:"symbol"`
	StartDate       string              `json:"start_date"`
	EndDate         string              `json:"end_date"`
	InitialCapital  float64             `json:"initial_capital"`
	FinalCapital    float64             `json:"final_capital"`
	TotalProfitLoss string              `json:"total_profit_loss"`
	ROI             float64             `json:"roi_percentage"`
	TotalTrades     int                 `json:"total_trades"`
	WinningTrades   int                 `json:"winning_trades"`
	LosingTrades    int                 `json:"losing_trades"`
	WinRate         string              `json:"win_rate"`
	MaxDrawdown     string              `json:"max_drawdown"`
	Trades          []BacktestTradeData `json:"trades"`
	CapitalHistory  []float64           `json:"capital_history"`
	CreatedAt       string              `json:"created_at"`
}

type BacktestTradeData struct {
	ID         string  `json:"id"`
	Decision   string  `json:"decision"`
	EntryPrice float64 `json:"entry_price"`
	ExitPrice  float64 `json:"exit_price"`
	Quantity   float64 `json:"quantity"`
	EntryTime  string  `json:"entry_time"`
	ExitTime   string  `json:"exit_time,omitempty"`
	ProfitLoss string  `json:"profit_loss,omitempty"`
	Reason     string  `json:"reason"`
}

func (c *BacktestStrategyController) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		c.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req BacktestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.sendErrorResponse(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if err := c.validateRequest(req); err != nil {
		c.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	var klines []vo.Kline
	var startDate, endDate time.Time
	var err error

	// Check if we should use historical data from Binance
	if req.UseBinanceData && req.StartDate != "" {
		// Fetch data from specific date to today
		fmt.Printf("🔍 Fetching data from %s to today for symbol: %s (interval: %s)\n", req.StartDate, req.Symbol, req.Interval)
		
		// Parse start date
		startDate, err = time.Parse(time.RFC3339, req.StartDate)
		if err != nil {
			c.sendErrorResponse(w, fmt.Sprintf("Invalid start date format: %v", err), http.StatusBadRequest)
			return
		}
		
		// Default interval to 30m if not provided
		interval := req.Interval
		if interval == "" {
			interval = "30m"
		}
		
		klines, err = c.historicalDataService.GetKlinesFromDateToToday(req.Symbol, startDate, interval)
		if err != nil {
			fmt.Printf("❌ Error fetching data from %s: %v\n", req.StartDate, err)
			c.sendErrorResponse(w, fmt.Sprintf("Failed to fetch data from %s: %v", req.StartDate, err), http.StatusInternalServerError)
			return
		}
		fmt.Printf("✅ Fetched %d klines from %s to today\n", len(klines), req.StartDate)

		// Set end date to yesterday
		now := time.Now()
		yesterday := now.AddDate(0, 0, -1)
		endDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, yesterday.Location())
		fmt.Printf("📅 Using date range: %s to %s\n", startDate.Format("2006-01-02 15:04:05"), endDate.Format("2006-01-02 15:04:05"))
	} else if req.UseLastWeek {
		// Fetch last week's data from Binance
		fmt.Printf("🔍 Fetching last week's data for symbol: %s\n", req.Symbol)
		klines, err = c.historicalDataService.GetLastWeekKlines(req.Symbol)
		if err != nil {
			fmt.Printf("❌ Error fetching last week's data: %v\n", err)
			c.sendErrorResponse(w, fmt.Sprintf("Failed to fetch last week's data: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Printf("✅ Fetched %d klines for last week\n", len(klines))

		// Set dates to last week
		now := time.Now()
		weekAgo := now.AddDate(0, 0, -7)
		yesterday := now.AddDate(0, 0, -1)
		startDate = time.Date(weekAgo.Year(), weekAgo.Month(), weekAgo.Day(), 0, 0, 0, 0, weekAgo.Location())
		endDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, yesterday.Location())
		fmt.Printf("📅 Using date range: %s to %s\n", startDate.Format("2006-01-02 15:04:05"), endDate.Format("2006-01-02 15:04:05"))
	} else if req.UseYesterday || (len(req.HistoricalData) == 0 && req.StartDate == "" && req.EndDate == "" && !req.UseBinanceData) {
		// Fetch yesterday's data from Binance
		fmt.Printf("🔍 Fetching yesterday's data for symbol: %s\n", req.Symbol)
		klines, err = c.historicalDataService.GetYesterdayKlines(req.Symbol)
		if err != nil {
			fmt.Printf("❌ Error fetching yesterday's data: %v\n", err)
			c.sendErrorResponse(w, fmt.Sprintf("Failed to fetch yesterday's data: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Printf("✅ Fetched %d klines for yesterday\n", len(klines))

		// Set dates to yesterday
		yesterday := time.Now().AddDate(0, 0, -1)
		startDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
		endDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, yesterday.Location())
		fmt.Printf("📅 Using date range: %s to %s\n", startDate.Format("2006-01-02 15:04:05"), endDate.Format("2006-01-02 15:04:05"))
	} else {
		// Use provided historical data
		klines, err = c.convertHistoricalData(req.HistoricalData)
		if err != nil {
			c.sendErrorResponse(w, fmt.Sprintf("Invalid historical data: %v", err), http.StatusBadRequest)
			return
		}

		// Parse dates
		startDate, err = time.Parse(time.RFC3339, req.StartDate)
		if err != nil {
			c.sendErrorResponse(w, fmt.Sprintf("Invalid start date format: %v", err), http.StatusBadRequest)
			return
		}

		endDate, err = time.Parse(time.RFC3339, req.EndDate)
		if err != nil {
			c.sendErrorResponse(w, fmt.Sprintf("Invalid end date format: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Create use case input
	input := usecase.InputBacktestStrategy{
		StrategyName:           req.StrategyName,
		Symbol:                 req.Symbol,
		Params:                 req.Params,
		HistoricalData:         klines,
		InitialCapital:         req.InitialCapital,
		TradeAmount:            req.TradeAmount,
		Currency:               req.Currency,
		StartDate:              startDate,
		EndDate:                endDate,
		TradingFees:            req.TradingFees,
		MinimumProfitThreshold: req.MinimumProfitThreshold,
	}

	// Execute backtest
	result, err := c.backtestUseCase.Execute(input)
	if err != nil {
		c.sendErrorResponse(w, fmt.Sprintf("Backtest failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert result to response format
	responseData := c.convertToResponseData(result)

	// Send success response
	response := BacktestResponse{
		Success: true,
		Data:    responseData,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (c *BacktestStrategyController) validateRequest(req BacktestRequest) error {
	if req.StrategyName == "" {
		return fmt.Errorf("strategy_name is required")
	}

	if req.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if req.InitialCapital <= 0 {
		return fmt.Errorf("initial_capital must be positive")
	}

	if req.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	if req.TradingFees < 0 {
		return fmt.Errorf("trading_fees cannot be negative")
	}

	// If using custom binance data, validate start_date is provided
	if req.UseBinanceData {
		if req.StartDate == "" {
			return fmt.Errorf("start_date is required when use_binance_data=true")
		}
		// Validate interval if provided
		if req.Interval != "" {
			validIntervals := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w", "1M"}
			isValid := false
			for _, valid := range validIntervals {
				if req.Interval == valid {
					isValid = true
					break
				}
			}
			if !isValid {
				return fmt.Errorf("invalid interval: %s. Valid intervals: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w, 1M", req.Interval)
			}
		}
	}

	// If not using automatic data fetching, validate that historical data and dates are provided
	if !req.UseYesterday && !req.UseLastWeek && !req.UseBinanceData && len(req.HistoricalData) == 0 && req.StartDate == "" && req.EndDate == "" {
		return fmt.Errorf("either set use_yesterday=true, use_last_week=true, use_binance_data=true with start_date, or provide historical_data with start_date and end_date")
	}

	// If providing manual data, ensure all required fields are present
	if !req.UseYesterday && !req.UseLastWeek && !req.UseBinanceData && len(req.HistoricalData) > 0 {
		if req.StartDate == "" {
			return fmt.Errorf("start_date is required when providing historical_data")
		}
		if req.EndDate == "" {
			return fmt.Errorf("end_date is required when providing historical_data")
		}
	}

	return nil
}

func (c *BacktestStrategyController) convertHistoricalData(data []HistoricalKline) ([]vo.Kline, error) {
	klines := make([]vo.Kline, len(data))

	for i, hk := range data {
		kline, err := vo.NewKline(hk.Open, hk.Close, hk.High, hk.Low, hk.Volume, hk.CloseTime)
		if err != nil {
			return nil, fmt.Errorf("invalid kline at index %d: %w", i, err)
		}
		klines[i] = kline
	}

	return klines, nil
}

func (c *BacktestStrategyController) convertToResponseData(result *entity.BacktestResult) *BacktestResultData {
	trades := make([]BacktestTradeData, len(result.GetTrades()))
	for i, trade := range result.GetTrades() {
		tradeData := BacktestTradeData{
			ID:         trade.GetId().GetValue(),
			Decision:   string(trade.GetDecision()),
			EntryPrice: trade.GetEntryPrice(),
			ExitPrice:  trade.GetExitPrice(),
			Quantity:   trade.GetQuantity(),
			EntryTime:  trade.GetEntryTime().Format(time.RFC3339),
			Reason:     trade.GetReason(),
		}

		if trade.GetExitTime() != nil {
			tradeData.ExitTime = trade.GetExitTime().Format(time.RFC3339)
		}

		if trade.GetProfitLoss() != nil {
			tradeData.ProfitLoss = trade.GetProfitLoss().String()
		}

		trades[i] = tradeData
	}

	return &BacktestResultData{
		ID:              result.GetId().GetValue(),
		StrategyName:    result.GetStrategyName(),
		Symbol:          result.GetSymbol().GetValue(),
		StartDate:       result.GetStartDate().Format(time.RFC3339),
		EndDate:         result.GetEndDate().Format(time.RFC3339),
		InitialCapital:  result.GetInitialCapital(),
		FinalCapital:    result.GetFinalCapital(),
		TotalProfitLoss: result.GetTotalProfitLoss().String(),
		ROI:             result.GetROI(),
		TotalTrades:     result.GetTotalTrades(),
		WinningTrades:   result.GetWinningTrades(),
		LosingTrades:    result.GetLosingTrades(),
		WinRate:         result.GetWinRate().String(),
		MaxDrawdown:     result.GetMaxDrawdown().String(),
		Trades:          trades,
		CapitalHistory:  result.GetCapitalHistory(),
		CreatedAt:       result.GetCreatedAt().Format(time.RFC3339),
	}
}

func (c *BacktestStrategyController) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := BacktestResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
