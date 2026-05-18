package subscription

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadDesired reads a desired provisioning file.
func LoadDesired(path string) (Desired, error) {
	var desired Desired
	if err := loadYAML(path, &desired); err != nil {
		return Desired{}, err
	}
	return desired, nil
}

// LoadState reads a generated state file.
func LoadState(path string) (State, error) {
	var state State
	if err := loadYAML(path, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

// LoadStateIfExists reads a state file when present and otherwise returns an
// empty state. This is useful for first provisioning runs.
func LoadStateIfExists(path string) (State, error) {
	state, err := LoadState(path)
	if err == nil {
		return state, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	return State{}, err
}

// SaveState writes generated state atomically.
func SaveState(path string, state State) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".state-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp state: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp state: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod temp state: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

func loadYAML(path string, dst any) error {
	data, err := os.ReadFile(path) // #nosec G304 -- path is explicit operator input.
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
