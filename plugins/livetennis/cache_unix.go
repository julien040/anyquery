//go:build !windows

package main

import (
	"os"
	"path/filepath"
)

func publishSnapshot(source, destination string) error {
	if err := os.Rename(source, destination); err != nil {
		return err
	}
	for path := filepath.Dir(destination); ; path = filepath.Dir(path) {
		directory, err := os.Open(path)
		if err != nil {
			return err
		}
		err = directory.Sync()
		closeErr := directory.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if path == filepath.Dir(path) {
			return nil
		}
	}
}
