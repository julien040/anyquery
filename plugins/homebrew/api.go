package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

func requestFormulae() (map[string]BrewFormula, error) {
	formulae := make(map[string]BrewFormula)

	// Get the file path
	cachePath := filepath.Join(xdg.CacheHome, "anyquery", "plugins", "brew", "formulae.json")

	// Check if the file exists
	var stat os.FileInfo
	var err error
	needToFetch := false
	if stat, err = os.Stat(cachePath); os.IsNotExist(err) {
		log.Println("Cache file does not exist")
		needToFetch = true
	} else {
		// Check if the file is older than 1 day
		if time.Since(stat.ModTime()).Hours() > 24 {
			needToFetch = true
			log.Println("Cache file is older than 1 day")
		}
	}

	if !needToFetch {
		log.Println("Using cache file")
		// Read the file
		file, err := os.Open(cachePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open cache file: %w", err)
		}
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&formulae)
		if err != nil {
			return nil, fmt.Errorf("failed to decode cache file: %w", err)
		}
		return formulae, nil
	}

	// Fetch the formulae
	dataAPI := []BrewFormula{}
	res, err := client.R().SetResult(&dataAPI).Get("https://formulae.brew.sh/api/formula.json")
	if err != nil {
		log.Println("Failed to fetch formulae from API")
		return nil, err
	}
	if res.IsError() {
		log.Printf("Failed to fetch formulae(code %d): %s", res.StatusCode(), res.String())
		return nil, res.Error().(error)
	}

	// Convert the data to a map
	for _, formula := range dataAPI {
		formulae[formula.Name] = formula
	}

	// Fetch the analytics for 30 days, 90 days and 365 days
	analytics := BrewAnalyticsFormulae{}

	// Fetch the analytics for 30 days
	res, err = client.R().SetResult(&analytics).Get("https://formulae.brew.sh/api/analytics/install/30d.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch analytics for 30 days: %w", err)
	}
	if res.IsError() {
		return nil, res.Error().(error)
	}

	for _, item := range analytics.Items {
		if formula, ok := formulae[item.Formula]; ok {
			// Parse the count
			item.Count = strings.ReplaceAll(item.Count, ",", "")
			count := 0
			if item.Count != "" {
				count, err = strconv.Atoi(item.Count)
				if err != nil {
					return nil, err
				}
			}
			formula.Install30days = count
			formulae[item.Formula] = formula
		}
	}

	// Do the same for 90 days
	res, err = client.R().SetResult(&analytics).Get("https://formulae.brew.sh/api/analytics/install/90d.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch analytics for 90 days: %w", err)
	}
	if res.IsError() {
		return nil, res.Error().(error)
	}

	for _, item := range analytics.Items {
		if formula, ok := formulae[item.Formula]; ok {
			// Parse the count
			item.Count = strings.ReplaceAll(item.Count, ",", "")
			count := 0
			if item.Count != "" {
				count, err = strconv.Atoi(item.Count)
				if err != nil {
					return nil, err
				}
			}
			formula.Install90days = count
			formulae[item.Formula] = formula
		}
	}

	// Do the same for 365 days
	res, err = client.R().SetResult(&analytics).Get("https://formulae.brew.sh/api/analytics/install/365d.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch analytics for 365 days: %w", err)
	}
	if res.IsError() {
		return nil, res.Error().(error)
	}

	for _, item := range analytics.Items {
		if formula, ok := formulae[item.Formula]; ok {
			// Parse the count
			item.Count = strings.ReplaceAll(item.Count, ",", "")
			count := 0
			if item.Count != "" {
				count, err = strconv.Atoi(item.Count)
				if err != nil {
					return nil, err
				}
			}
			formula.Install365days = count
			formulae[item.Formula] = formula
		}
	}

	// Write the data to the cache
	err = os.MkdirAll(filepath.Dir(cachePath), 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	file, err := os.Create(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache file: %w", err)
	}

	encoder := json.NewEncoder(file)
	err = encoder.Encode(formulae)
	if err != nil {
		return nil, err
	}

	return formulae, nil
}

func requestCasks() (map[string]BrewCasks, error) {
	formulae := make(map[string]BrewCasks)

	// Get the file path
	cachePath := filepath.Join(xdg.CacheHome, "anyquery", "plugins", "brew", "casks.json")

	// Check if the file exists
	var stat os.FileInfo
	var err error
	needToFetch := false
	if stat, err = os.Stat(cachePath); os.IsNotExist(err) {
		log.Println("Cache file does not exist")
		needToFetch = true
	} else {
		// Check if the file is older than 1 day
		if time.Since(stat.ModTime()).Hours() > 24 {
			needToFetch = true
			log.Println("Cache file is older than 1 day")
		}
	}

	if !needToFetch {
		log.Println("Using cache file")
		// Read the file
		file, err := os.Open(cachePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open cache file: %w", err)
		}
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&formulae)
		if err != nil {
			return nil, fmt.Errorf("failed to decode cache file: %w", err)
		}
		return formulae, nil
	}

	// Fetch the formulae
	dataAPI := []BrewCasks{}
	res, err := client.R().SetResult(&dataAPI).Get("https://formulae.brew.sh/api/cask.json")
	if err != nil {
		log.Println("Failed to fetch formulae from API")
		return nil, err
	}
	if res.IsError() {
		log.Printf("Failed to fetch formulae(code %d): %s", res.StatusCode(), res.String())
		return nil, res.Error().(error)
	}

	// Convert the data to a map
	for _, cask := range dataAPI {
		formulae[cask.Token] = cask
	}

	// Fetch the analytics for 30 days, 90 days and 365 days
	analytics := BrewAnalyticsCask{}

	// Fetch the analytics for 30 days
	res, err = client.R().SetResult(&analytics).Get("https://formulae.brew.sh/api/analytics/cask-install/homebrew-cask/30d.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch analytics for 30 days: %w", err)
	}
	if res.IsError() {
		return nil, res.Error().(error)
	}

	for k, itemArr := range analytics.Formulae {
		item := itemArr[0]
		if formula, ok := formulae[k]; ok {
			// Parse the count
			item.Count = strings.ReplaceAll(item.Count, ",", "")
			count := 0
			if item.Count != "" {
				count, err = strconv.Atoi(item.Count)
				if err != nil {
					return nil, err
				}
			}
			formula.Install30days = count
			formulae[k] = formula
		}
	}

	// Do the same for 90 days
	res, err = client.R().SetResult(&analytics).Get("https://formulae.brew.sh/api/analytics/cask-install/homebrew-cask/90d.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch analytics for 90 days: %w", err)
	}
	if res.IsError() {
		return nil, res.Error().(error)
	}

	for k, itemArr := range analytics.Formulae {
		item := itemArr[0]
		if formula, ok := formulae[k]; ok {
			// Parse the count
			item.Count = strings.ReplaceAll(item.Count, ",", "")
			count := 0
			if item.Count != "" {
				count, err = strconv.Atoi(item.Count)
				if err != nil {
					return nil, err
				}
			}
			formula.Install90days = count
			formulae[k] = formula
		}
	}

	// Do the same for 365 days
	res, err = client.R().SetResult(&analytics).Get("https://formulae.brew.sh/api/analytics/cask-install/homebrew-cask/365d.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch analytics for 365 days: %w", err)
	}
	if res.IsError() {
		return nil, res.Error().(error)
	}

	for k, itemArr := range analytics.Formulae {
		item := itemArr[0]
		if formula, ok := formulae[k]; ok {
			// Parse the count
			item.Count = strings.ReplaceAll(item.Count, ",", "")
			count := 0
			if item.Count != "" {
				count, err = strconv.Atoi(item.Count)
				if err != nil {
					return nil, err
				}
			}
			formula.Install365days = count
			formulae[k] = formula
		}
	}

	// Write the data to the cache
	err = os.MkdirAll(filepath.Dir(cachePath), 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	file, err := os.Create(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache file: %w", err)
	}

	encoder := json.NewEncoder(file)
	err = encoder.Encode(formulae)
	if err != nil {
		return nil, err
	}

	return formulae, nil
}
