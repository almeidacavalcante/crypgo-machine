package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"fmt"
)

type StopTradingBotUseCase struct {
	tradingBotRepository repository.TradingBotRepository
}

func NewStopTradingBotUseCase(tradingBotRepository repository.TradingBotRepository) *StopTradingBotUseCase {
	return &StopTradingBotUseCase{
		tradingBotRepository: tradingBotRepository,
	}
}

type InputStopTradingBot struct {
	BotId string `json:"bot_id"`
}

func (uc *StopTradingBotUseCase) Execute(input InputStopTradingBot) error {
	if input.BotId == "" {
		return fmt.Errorf("bot_id is required")
	}

	_, err := vo.RestoreEntityId(input.BotId)
	if err != nil {
		return fmt.Errorf("invalid bot_id format: %v", err)
	}

	bot, err := uc.tradingBotRepository.GetTradeByID(input.BotId)
	if err != nil {
		return fmt.Errorf("failed to find trading bot: %v", err)
	}

	if bot == nil {
		return fmt.Errorf("trading bot not found with id: %s", input.BotId)
	}

	if bot.GetStatus() == entity.StatusStopped {
		return fmt.Errorf("trading bot is already stopped")
	}

	// Set bot status to STOPPED
	bot.Stop()

	err = uc.tradingBotRepository.Update(bot)
	if err != nil {
		return fmt.Errorf("failed to stop trading bot: %v", err)
	}

	return nil
}