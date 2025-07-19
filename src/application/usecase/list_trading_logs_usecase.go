package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
)

// ListTradingLogsUseCase handles listing of trading decision logs
type ListTradingLogsUseCase struct {
	tradingDecisionLogRepository repository.TradingDecisionLogRepository
	tradingBotRepository         repository.TradingBotRepository
}

// ListTradingLogsInput represents the input for listing trading logs
type ListTradingLogsInput struct {
	Decision string `json:"decision"` // Optional filter by decision (HOLD, BUY, SELL)
	Symbol   string `json:"symbol"`   // Optional filter by symbol (BTCBRL, ETHBRL, SOLBRL)
	Limit    int    `json:"limit"`    // Number of logs to return (default 20)
	Offset   int    `json:"offset"`   // Number of logs to skip for pagination (default 0)
}

// TradingLogOutput represents a single trading log with bot information
type TradingLogOutput struct {
	ID                  string                 `json:"id"`
	BotID               string                 `json:"bot_id"`
	Symbol              string                 `json:"symbol"`
	Decision            string                 `json:"decision"`
	CurrentPrice        float64                `json:"current_price"`
	EntryPrice          *float64               `json:"entry_price,omitempty"`
	ProfitPercentage    *float64               `json:"profit_percentage,omitempty"`
	StrategyName        string                 `json:"strategy_name"`
	AnalysisData        map[string]interface{} `json:"analysis_data"`
	Timestamp           string                 `json:"timestamp"`
	IsPositioned        bool                   `json:"is_positioned"`
}

// ListTradingLogsOutput represents the output of listing trading logs
type ListTradingLogsOutput struct {
	Logs  []TradingLogOutput `json:"logs"`
	Total int                `json:"total"`
}

// NewListTradingLogsUseCase creates a new use case for listing trading logs
func NewListTradingLogsUseCase(
	tradingDecisionLogRepository repository.TradingDecisionLogRepository,
	tradingBotRepository repository.TradingBotRepository,
) *ListTradingLogsUseCase {
	return &ListTradingLogsUseCase{
		tradingDecisionLogRepository: tradingDecisionLogRepository,
		tradingBotRepository:         tradingBotRepository,
	}
}

// Execute lists trading logs with optional filters
func (uc *ListTradingLogsUseCase) Execute(input ListTradingLogsInput) (*ListTradingLogsOutput, error) {
	// Set default limit
	if input.Limit <= 0 {
		input.Limit = 20
	}

	// Use new method with filters and pagination
	logs, total, err := uc.tradingDecisionLogRepository.GetLogsWithFilters(
		input.Decision, 
		input.Symbol, 
		input.Limit, 
		input.Offset,
	)

	if err != nil {
		return nil, err
	}

	// Get all trading bots to enrich the data
	bots, err := uc.tradingBotRepository.GetAllTradingBots()
	if err != nil {
		return nil, err
	}

	// Create a map for quick bot lookup
	botMap := make(map[string]*entity.TradingBot)
	for _, bot := range bots {
		botMap[bot.Id.GetValue()] = bot
	}

	// Convert to output format
	var outputLogs []TradingLogOutput
	for _, log := range logs {
		bot := botMap[log.GetTradingBotId().GetValue()]
		
		outputLog := TradingLogOutput{
			ID:           log.GetId().GetValue(),
			BotID:        log.GetTradingBotId().GetValue(),
			Decision:     string(log.GetDecision()),
			CurrentPrice: log.GetCurrentPrice(),
			StrategyName: log.GetStrategyName(),
			AnalysisData: log.GetAnalysisData(),
			Timestamp:    log.GetTimestamp().Format("2006-01-02T15:04:05Z07:00"),
		}

		// Add bot-specific information if bot exists
		if bot != nil {
			outputLog.Symbol = bot.GetSymbol().GetValue()
			outputLog.IsPositioned = bot.GetIsPositioned()

			// Calculate profit percentage if bot is positioned
			if bot.GetIsPositioned() && bot.GetEntryPrice() > 0 {
				entryPrice := bot.GetEntryPrice()
				outputLog.EntryPrice = &entryPrice
				
				profitPercentage := ((log.GetCurrentPrice() - entryPrice) / entryPrice) * 100
				outputLog.ProfitPercentage = &profitPercentage
			}
		}

		outputLogs = append(outputLogs, outputLog)
	}

	return &ListTradingLogsOutput{
		Logs:  outputLogs,
		Total: total,
	}, nil
}