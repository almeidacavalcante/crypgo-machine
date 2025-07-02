package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
)

type Strategy struct {
	ID     *vo.EntityId
	name   vo.Name
	params map[string]interface{}
}

func NewStrategy(name string, params map[string]interface{}) (*Strategy, error) {
	id := vo.NewEntityId()
	if err := validateStrategyParams(params); err != nil {
		return nil, fmt.Errorf("invalid strategy: %w", err)
	}
	nameInst, errName := vo.NewName(name)
	if errName != nil {
		return nil, errName
	}

	return &Strategy{
		ID:     id,
		name:   nameInst,
		params: params,
	}, nil
}

func RestoreStrategy(id *vo.EntityId, name string, params map[string]interface{}) (*Strategy, error) {
	err := validateStrategyParams(params)
	if err != nil {
		return nil, err
	}
	nameInst, errName := vo.NewName(name)
	if errName != nil {
		return nil, errName
	}

	return &Strategy{
		ID:     id,
		name:   nameInst,
		params: params,
	}, nil
}

func validateStrategyParams(params map[string]interface{}) error {
	if len(params) == 0 {
		return fmt.Errorf("strategy parameters cannot be empty")
	}
	return nil
}

func (s *Strategy) GetName() string {
	return s.name.GetValue()
}

func (s *Strategy) GetParams() map[string]interface{} {
	return s.params
}
