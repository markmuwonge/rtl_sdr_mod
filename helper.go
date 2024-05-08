package rtl_sdr_mod

import (
	"errors"
	"log"
	"os"
)

func deleteFile(path string) {
	if !fileExists(path) {
		return
	}

	err := os.Remove(path)
	if err != nil {
		log.Print(err)
	}
}

func fileExists(path string) bool {
	file, err := os.OpenFile(path, os.O_RDONLY, 600)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
	return true
}
