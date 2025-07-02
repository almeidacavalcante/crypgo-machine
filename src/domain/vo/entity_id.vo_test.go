package vo

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewEntityId(t *testing.T) {
	id := NewEntityId()
	require.NotEmpty(t, id)
	require.Equal(t, len(id.GetValue()), 36)
	parsed, err := uuid.Parse(id.GetValue())
	require.NoError(t, err)
	require.NotEmpty(t, parsed)
	require.Equal(t, parsed.String(), id.GetValue())
}

func TestEntityId_RestoreAndInvalid(t *testing.T) {
	id := NewEntityId()
	require.NotEmpty(t, id)
	require.Equal(t, len(id.GetValue()), 36)

	restored, err := RestoreEntityId(id.GetValue())
	require.NoError(t, err)
	require.NotEmpty(t, restored)
	require.Equal(t, restored.GetValue(), id.GetValue())
	require.Equal(t, len(restored.GetValue()), 36)

	_, err = RestoreEntityId("invalid-uuid")
	require.Error(t, err)
}
