package entity

type StrategyAnalysisResult struct {
	Decision     TradingDecision
	AnalysisData map[string]interface{}
}

func NewStrategyAnalysisResult(decision TradingDecision, analysisData map[string]interface{}) *StrategyAnalysisResult {
	return &StrategyAnalysisResult{
		Decision:     decision,
		AnalysisData: analysisData,
	}
}