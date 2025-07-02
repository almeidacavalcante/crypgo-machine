package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
)

type ListAllTradingBotsUseCase struct {
	repository repository.TradingBotRepository
}

func NewListAllTradingBotsUseCase(repo repository.TradingBotRepository) *ListAllTradingBotsUseCase {
	return &ListAllTradingBotsUseCase{
		repository: repo,
	}
}

func (u *ListAllTradingBotsUseCase) Execute() ([]*entity.TradingBot, error) {
	bots, err := u.repository.GetAllTradingBots()
	if err != nil {
		return nil, err
	}
	return bots, nil
}
