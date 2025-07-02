package vo

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewPrice(t *testing.T) {
	p, err := NewPrice(1.50, "USD")
	require.NoError(t, err)
	require.Equal(t, 1.50, p.GetAmount())
	require.Equal(t, "USD", p.GetCurrency())
}

func TestPrice_InvalidValues(t *testing.T) {
	_, err := NewPrice(0, "USD")
	require.Error(t, err)
	_, err = NewPrice(10, "USDT")
	require.Error(t, err)
}
