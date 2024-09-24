package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/go-git/go-git/v5"
	"github.com/julien040/anyquery/rpc"
)

func main() {
	plugin := rpc.NewPlugin(commitsCreator, branchesCreator, tagsCreator, remotesCreator, statusCreator,
		referencesCreator, commit_diffCreator)
	plugin.Serve()
}

func openRepository(path string) (*git.Repository, error) {
	pathToRepo := path
	// Check if the path is a URL
	// If so, clone the repository
	if parsedUrl, err := url.ParseRequestURI(path); err == nil && parsedUrl.Scheme != "" {
		// It's a URL that we will clone with git
		hash := md5.Sum([]byte(path))
		cachePath := filepath.Join(xdg.CacheHome, "anyquery", "plugins", "git", "cache", fmt.Sprintf("%x", hash))

		// Check if the directory exists and has a .git directory (to not clone it again)
		if _, err := os.Stat(filepath.Join(cachePath, ".git")); os.IsNotExist(err) {
			// Clone the repository
			cmd := exec.Command("git", "clone", path, cachePath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("error cloning repository: %s", output)
			}
		} else {
			// Pull the repository
			cmd := exec.Command("git", "pull", "--rebase")
			cmd.Dir = cachePath
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("error pulling repository %s: %s", path, output)
			}
		}

		pathToRepo = cachePath
	}

	repo, err := git.PlainOpen(pathToRepo)
	if err == git.ErrRepositoryNotExists {
		return nil, fmt.Errorf("repository does not exist at %s. If it's a URL, make sure it starts with 'https://'", path)
	} else if err != nil {
		return nil, fmt.Errorf("error opening repository: %s", err)
	}
	return repo, err
}
