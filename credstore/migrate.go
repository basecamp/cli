package credstore

import (
	"fmt"
	"os"
)

// MigrateToKeyring migrates credentials from file to keyring.
// No-op if keyring is not available.
func (s *Store) MigrateToKeyring() error {
	if !s.useKeyring {
		return nil
	}

	all, err := s.loadAllFromFile()
	if err != nil {
		return nil // No file to migrate
	}

	for key, data := range all {
		if err := s.Save(key, data); err != nil {
			return fmt.Errorf("failed to migrate %s: %w", key, err)
		}
	}

	_ = os.Remove(s.credentialsPath())
	return nil
}
