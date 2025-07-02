package vo

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTimeframe(t *testing.T) {
	_, err := NewTimeframe("1m")
	require.NoError(t, err)
}

func TestTimeframe_Invalid(t *testing.T) {
	_, err := NewTimeframe("2m")
	require.Error(t, err)
}
