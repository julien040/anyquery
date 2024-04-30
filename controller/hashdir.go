package controller

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/mod/sumdb/dirhash"
)

func hashDirectory(path string) (string, error) {
	str, err := dirhash.HashDir(path, "", dirhash.Hash1)
	if err != nil {
		return "", err
	}

	// We remove the h1: prefix
	return str[4:], nil

}

func HashDir(cmd *cobra.Command, args []string) error {
	// We get the path from the arguments
	var path string
	var err error
	if len(args) == 0 {
		path, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get the current directory: %w", err)
		}
	} else {
		path = args[0]
	}

	// We check if the path is a directory
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("could not stat the path: %w", err)
	}

	if !stat.IsDir() {
		return fmt.Errorf("the path is not a directory")
	}

	// We hash the directory
	hash, err := hashDirectory(path)

	if err != nil {
		return fmt.Errorf("could not hash the directory: %w", err)
	}

	fmt.Println(hash)

	return nil
}
