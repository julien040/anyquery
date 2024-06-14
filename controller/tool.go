package controller

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/mod/sumdb/dirhash"
	"golang.org/x/term"
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

func mysqlNativePassword(password []byte) string {
	// The format for the a native password is:
	// HEX( SHA1( SHA1( password ) ) )
	// Reference https://vitess.io/docs/17.0/user-guides/configuration-advanced/static-auth/#mysql-native-password

	// We hash the password
	hashedPasswordOnce := sha1.Sum(password)
	hashedPasswordTwice := sha1.Sum(hashedPasswordOnce[:]) // We convert the array to a slice

	// We convert the slice to a hexadecimal upper case string
	// prefixed with *
	return fmt.Sprintf("*%X", hashedPasswordTwice)
}

func MySQLPassword(cmd *cobra.Command, args []string) error {
	// We read the password from stdin
	var clearPassword []byte
	var err error
	if isSTDinAtty() {
		fmt.Print("Password: ")
		clearPassword, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("could not read the password: %w", err)
		}
		// Clear the line
		fmt.Printf("\r")
	} else {
		clearPassword, err = bufio.NewReader(os.Stdin).ReadBytes('\n')
		// If the last character is a newline, we remove it
		if clearPassword[len(clearPassword)-1] == '\n' {
			// Remove the newline character
			clearPassword = clearPassword[:len(clearPassword)-1]
		}
		if err != nil {
			return fmt.Errorf("could not read the password: %w", err)
		}
	}

	// We hash the password
	hashedPassword := mysqlNativePassword(clearPassword)

	fmt.Println(hashedPassword)
	return nil
}
