package vo

import (
	"github.com/stretchr/testify/require"
	"testing"
)

var curr = Currency{code: "USD"}

func TestNewPrice(t *testing.T) {
	p, err := NewPrice(100.50, curr)
	require.NoError(t, err)
	require.Equal(t, 100.50, p.Amount())
	require.Equal(t, "USD", p.Currency())
}

func TestNewPrice_Invalid(t *testing.T) {
	_, err := NewPrice(-10.00, curr)
	require.Error(t, err)

	_, err = NewPrice(0.00, curr)
	require.NoError(t, err) // Zero price is valid for now
}
