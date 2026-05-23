package envmanager

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
)

// ErrDatabaseInUse is returned when the SQLite file is locked by another process.
var ErrDatabaseInUse = errors.New("database file is in use by another process")

const confirmCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateConfirmCode returns a random alphanumeric code (letters and digits only).
func GenerateConfirmCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("code length must be positive")
	}
	code := make([]byte, length)
	max := big.NewInt(int64(len(confirmCharset)))
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		code[i] = confirmCharset[n.Int64()]
	}
	return string(code), nil
}

func isFileInUse(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "being used by another process") ||
		strings.Contains(msg, "used by another process") ||
		strings.Contains(msg, "resource busy") ||
		strings.Contains(msg, "database is locked")
}

// CheckDatabaseNotInUse returns an error if the database file exists but cannot be opened for write.
func CheckDatabaseNotInUse(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		if isFileInUse(err) {
			return fmt.Errorf("%w (stop \"elsa env serve\" with Ctrl+C, then run reset again)", ErrDatabaseInUse)
		}
		return err
	}
	return f.Close()
}

// DeleteDatabase removes the SQLite database and related WAL/SHM files.
func DeleteDatabase(path string) error {
	if err := CheckDatabaseNotInUse(path); err != nil {
		return err
	}

	removed := false
	for _, p := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.Remove(p); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			if isFileInUse(err) {
				return fmt.Errorf("%w: close \"elsa env serve\" or any app using %s, then retry", ErrDatabaseInUse, path)
			}
			return fmt.Errorf("remove %s: %w", p, err)
		}
		removed = true
	}
	if !removed {
		return fmt.Errorf("database not found: %s", path)
	}
	return nil
}
