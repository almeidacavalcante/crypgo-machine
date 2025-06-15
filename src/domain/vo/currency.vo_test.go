package vo

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewCurrency_Valid(t *testing.T) {
	c, err := NewCurrency("USD")
	require.NoError(t, err)
	require.Equal(t, "USD", c.Code())
}

func TestNewCurrency_Invalid(t *testing.T) {
	_, err := NewCurrency("US")
	require.Error(t, err)

	_, err = NewCurrency("")
	require.Error(t, err)
}
