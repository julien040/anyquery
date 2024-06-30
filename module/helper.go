package module

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
	"github.com/edsrzf/mmap-go"
	"github.com/hashicorp/go-getter"
)

// downloadFile downloads a file from a source URL to a destination path
// If the destination file already exists and its size is superior to 0,
// the file is not downloaded again
//
// The destination path is created if it doesn't exist
func downloadFile(src string, dst string) error {
	// Create the directory if it doesn't exist
	err := os.MkdirAll(path.Dir(dst), 0755)
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	needToDownload := true

	// Check if the file is already downloaded and its size superior to 0
	// If so, we don't need to download it again
	info, err := os.Stat(dst)
	if err == nil && info.Size() > 0 {
		needToDownload = false
	}

	// Download the file
	client := &getter.Client{
		Src:  src,
		Dst:  dst,
		Mode: getter.ClientModeFile,
		Pwd:  wd,
	}

	if needToDownload {
		err = client.Get()
		if err != nil {
			return fmt.Errorf("failed to download file: %s", err)
		}
	}

	return nil
}

// findCachedDestination returns the path where the cached file needs to be stored
// It's based on the sha256 of the source URL
func findCachedDestination(src string) (string, error) {
	// Hash the file path
	// This is to avoid conflicts with other files and reuse the same file
	// if it's already downloaded
	hash := sha256.Sum256([]byte(src))
	filePath := path.Join(xdg.CacheHome, "anyquery", "downloads", hex.EncodeToString(hash[:]))

	return filePath, nil
}

// openMmapedFile downloads a file from a source URL and returns a mmap of the file
func openMmapedFile(src string) (mmap.MMap, error) {
	// Find the cached destination
	filePath, err := findCachedDestination(src)

	err = downloadFile(src, filePath)
	if err != nil {
		return nil, err
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	// Map the file
	return mmap.Map(file, mmap.RDONLY, 0)
}

// To make the argument parsing more readable,
// we define a struct to hold the argument name and its value
type argParam struct {
	name  string
	value *string
}

var argRegExp *regexp.Regexp = regexp.MustCompile(`^\s*['"]?([^= '"]+?)['"]?\s*=\s*['"]?(.*?)['"]?\s*$`)

func parseArgs(params []argParam, args []string) {
	// It's quadratic but the number of arguments is small
	for i := 0; i < len(args); i++ {
		// Check if the argument is empty
		if args[i] == "" {
			continue
		}

		// Parse the argument
		matches := argRegExp.FindStringSubmatch(args[i])
		if matches == nil {
			continue
		}

		matches[1] = strings.ToLower(matches[1])

		// Check if the argument starts with the parameter name
		for j := 0; j < len(params); j++ {
			if matches[1] == params[j].name {
				*params[j].value = matches[2]
				break
			}
		}
	}
}

var sqliteValidName *regexp.Regexp = regexp.MustCompile(`[^\p{L}\p{N}_]+`)

func transformSQLiteValidName(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)
	spaceRemoved := strings.Map(func(r rune) rune {
		if r == ' ' || r == '.' || r == '-' || r == '/' {
			return '_'
		}
		return r
	}, s)

	return sqliteValidName.ReplaceAllString(spaceRemoved, "")
}
