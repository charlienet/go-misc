package fs

import (
	"errors"
	"os"
)

func Exist(name string) bool {
	_, err := os.Stat(name)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false
		}
	}

	return true
}
