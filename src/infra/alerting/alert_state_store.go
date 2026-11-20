package alerting

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"crypgo-machine/src/application/basismonitor"
)

// FileAlertStateStore keeps the alert state in a JSON file so it survives
// between runs of a scheduled job.
type FileAlertStateStore struct {
	path string
}

func NewFileAlertStateStore(path string) *FileAlertStateStore {
	return &FileAlertStateStore{path: path}
}

// Load returns the zero state when the file does not exist yet.
func (s *FileAlertStateStore) Load() (basismonitor.AlertState, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return basismonitor.AlertState{}, nil
	}
	if err != nil {
		return basismonitor.AlertState{}, fmt.Errorf("failed to read alert state file: %w", err)
	}

	var state basismonitor.AlertState
	if err := json.Unmarshal(data, &state); err != nil {
		return basismonitor.AlertState{}, fmt.Errorf("alert state file is not valid JSON: %w", err)
	}
	return state, nil
}

func (s *FileAlertStateStore) Save(state basismonitor.AlertState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode alert state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("failed to create alert state directory: %w", err)
	}
	if err := os.WriteFile(s.path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("failed to write alert state file: %w", err)
	}
	return nil
}

// MemoryAlertStateStore keeps the state for the lifetime of the process only.
// One-off runs use it so they never affect the cooldown of scheduled runs.
type MemoryAlertStateStore struct {
	state basismonitor.AlertState
}

func NewMemoryAlertStateStore() *MemoryAlertStateStore {
	return &MemoryAlertStateStore{}
}

func (s *MemoryAlertStateStore) Load() (basismonitor.AlertState, error) {
	return s.state, nil
}

func (s *MemoryAlertStateStore) Save(state basismonitor.AlertState) error {
	s.state = state
	return nil
}

// WriterNotifier prints the alert instead of sending it. Dry runs use it.
type WriterNotifier struct {
	out io.Writer
}

func NewWriterNotifier(out io.Writer) *WriterNotifier {
	return &WriterNotifier{out: out}
}

func (n *WriterNotifier) Send(message string) error {
	_, err := fmt.Fprintf(n.out, "--- alert (not sent) ---\n%s\n------------------------\n", message)
	return err
}
