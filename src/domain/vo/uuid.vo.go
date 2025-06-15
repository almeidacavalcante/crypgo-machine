package vo

import (
	"github.com/google/uuid"
)

type UUID string

func NewUUID() UUID {
	return UUID(uuid.New().String())
}

func ParseUUID(val string) (UUID, error) {
	_, err := uuid.Parse(val)
	if err != nil {
		return "", err
	}
	return UUID(val), nil
}
