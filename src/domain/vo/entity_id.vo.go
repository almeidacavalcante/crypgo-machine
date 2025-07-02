package vo

import (
	"github.com/google/uuid"
)

type EntityId struct {
	value string
}

func NewEntityId() *EntityId {
	id := uuid.New().String()
	return &EntityId{
		value: id,
	}
}

func (e *EntityId) GetValue() string {
	return e.value
}

func RestoreEntityId(value string) (*EntityId, error) {
	_, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &EntityId{
		value: value,
	}, nil
}
