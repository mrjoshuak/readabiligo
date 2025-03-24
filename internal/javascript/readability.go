package javascript

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ReadabilityResult represents the result from Readability.js
type ReadabilityResult struct {
	Title       string `json:"title"`
	Byline      string `json:"byline"`
	Content     string `json:"content"`
	TextContent string `json:"textContent"`
	Length      int    `json:"length"`
	Excerpt     string `json:"excerpt"`
	SiteName    string `json:"siteName"`
	Date        string `json:"date"`
}

// HaveNode checks if Node.js is available and has a compatible version
func HaveNode() bool {
	// Check if node is installed
	cmd := exec.Command("node", "-v")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse version string (e.g., "v14.17.0")
	versionStr := strings.TrimSpace(string(output))
	if len(versionStr) < 2 || versionStr[0] != 'v' {
		return false
	}

	// Extract major version number
	parts := strings.Split(versionStr[1:], ".")
	if len(parts) < 1 {
		return false
	}

	// Check if major version is >= 10
	var major int
	_, err = fmt.Sscanf(parts[0], "%d", &major)
	if err != nil || major < 10 {
		return false
	}

	// Check if the JavaScript directory exists
	jsDir := getJavaScriptDir()
	nodeModules := filepath.Join(jsDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		// Try installing node dependencies
		return runNpmInstall()
	}

	return true
}

// getJavaScriptDir returns the path to the JavaScript directory
func getJavaScriptDir() string {
	// Get the directory of the current file
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

// runNpmInstall runs npm install in the JavaScript directory
func runNpmInstall() bool {
	jsDir := getJavaScriptDir()
	cmd := exec.Command("npm", "install")
	cmd.Dir = jsDir
	err := cmd.Run()
	return err == nil
}

// ExtractArticle extracts article content using Readability.js
func ExtractArticle(html string) (*ReadabilityResult, error) {
	if !HaveNode() {
		return nil, fmt.Errorf("node executable not found or incompatible version, or npm install failed")
	}

	// Create a temporary file for the HTML input
	htmlFile, err := os.CreateTemp("", "readabiligo-*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary HTML file: %w", err)
	}
	defer os.Remove(htmlFile.Name())

	// Write the HTML to the temporary file
	_, err = io.WriteString(htmlFile, html)
	if err != nil {
		return nil, fmt.Errorf("failed to write HTML to temporary file: %w", err)
	}
	htmlFile.Close()

	// Create a temporary file for the JSON output
	jsonFile, err := os.CreateTemp("", "readabiligo-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary JSON file: %w", err)
	}
	defer os.Remove(jsonFile.Name())
	jsonFile.Close()

	// Run the Node.js script
	jsDir := getJavaScriptDir()
	cmd := exec.Command("node", "ExtractArticle.js", "-i", htmlFile.Name(), "-o", jsonFile.Name())
	cmd.Dir = jsDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run Node.js script: %w, output: %s", err, string(output))
	}

	// Read the JSON output
	jsonData, err := os.ReadFile(jsonFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON output: %w", err)
	}

	// Parse the JSON output
	var result ReadabilityResult
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON output: %w", err)
	}

	return &result, nil
}
