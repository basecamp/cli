package credstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func (s *Store) credentialsPath() string {
	return filepath.Join(s.fallbackDir, "credentials.json")
}

func (s *Store) loadAllFromFile() (map[string][]byte, error) {
	data, err := os.ReadFile(s.credentialsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string][]byte), nil
		}
		return nil, err
	}

	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(all))
	for k, v := range all {
		result[k] = []byte(v)
	}
	return result, nil
}

func (s *Store) saveAllToFile(all map[string][]byte) error {
	if err := os.MkdirAll(s.fallbackDir, 0700); err != nil {
		return err
	}

	// Convert to json.RawMessage for proper JSON nesting
	raw := make(map[string]json.RawMessage, len(all))
	for k, v := range all {
		raw[k] = json.RawMessage(v)
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write
	tmpFile, err := os.CreateTemp(s.fallbackDir, "credentials-*.json.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	destPath := s.credentialsPath()
	if err := os.Rename(tmpPath, destPath); err != nil {
		if runtime.GOOS == "windows" {
			_ = os.Remove(destPath)
			return os.Rename(tmpPath, destPath)
		}
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func (s *Store) loadFromFile(key string) ([]byte, error) {
	all, err := s.loadAllFromFile()
	if err != nil {
		return nil, err
	}
	data, ok := all[key]
	if !ok {
		return nil, fmt.Errorf("credentials not found for %s", key)
	}
	return data, nil
}

func (s *Store) saveToFile(key string, data []byte) error {
	all, err := s.loadAllFromFile()
	if err != nil {
		return err
	}
	all[key] = data
	return s.saveAllToFile(all)
}

func (s *Store) deleteFromFile(key string) error {
	all, err := s.loadAllFromFile()
	if err != nil {
		return err
	}
	delete(all, key)
	return s.saveAllToFile(all)
}
