package utils

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ReadURLsFromFile reads URLs from a file.
// Supports:
// - JSON files (.json): array of strings, array of objects containing a "url" or "link" field,
//   or an object containing a "urls" or "downloads" array.
// - CSV files (.csv): columns with "url" or "link" header, or fields starting with http://, https://, or ftp://.
// - Plain text files (.txt): one URL per line or whitespace-separated URLs, ignoring comments (#).
// Trailing-slash-only variants are treated as the same URL so batch imports
// behave consistently across CLI and TUI entry points.
func ReadURLsFromFile(path string) ([]string, error) {
	fileData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	trimmed := bytes.TrimSpace(fileData)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("no valid URLs found in file")
	}

	// 1. Try JSON if .json extension or starts with { or [
	if ext == ".json" || (len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')) {
		if urls, ok := tryParseJSONURLs(trimmed); ok {
			if len(urls) == 0 {
				return nil, fmt.Errorf("no valid URLs found in file")
			}
			return urls, nil
		}
		// If explicit .json was requested and failed JSON parsing, report error
		if ext == ".json" {
			return nil, fmt.Errorf("invalid JSON file: %w", err)
		}
	}

	// 2. Try CSV if .csv extension
	if ext == ".csv" {
		urls, err := parseCSVURLs(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid CSV file: %w", err)
		}
		if len(urls) == 0 {
			return nil, fmt.Errorf("no valid URLs found in file")
		}
		return urls, nil
	}

	// 3. Fall back to line/whitespace parser
	urls, err := parseTextURLs(bytes.NewReader(fileData))
	if err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no valid URLs found in file")
	}
	return urls, nil
}

func tryParseJSONURLs(data []byte) ([]string, bool) {
	// Case A: array of items
	var rawList []json.RawMessage
	if err := json.Unmarshal(data, &rawList); err == nil {
		var collected []string
		for _, item := range rawList {
			var str string
			if err := json.Unmarshal(item, &str); err == nil {
				if isValidURLString(str) {
					collected = append(collected, str)
				}
				continue
			}
			var obj map[string]interface{}
			if err := json.Unmarshal(item, &obj); err == nil {
				if u := extractURLFromMap(obj); u != "" {
					collected = append(collected, u)
				}
			}
		}
		return deduplicateURLs(collected), true
	}

	// Case B: single root object
	var rootObj map[string]interface{}
	if err := json.Unmarshal(data, &rootObj); err == nil {
		var collected []string
		for _, key := range []string{"urls", "links", "downloads", "items"} {
			if arr, ok := rootObj[key].([]interface{}); ok {
				for _, elem := range arr {
					if str, ok := elem.(string); ok && isValidURLString(str) {
						collected = append(collected, str)
					} else if m, ok := elem.(map[string]interface{}); ok {
						if u := extractURLFromMap(m); u != "" {
							collected = append(collected, u)
						}
					}
				}
			}
		}
		if len(collected) == 0 {
			if u := extractURLFromMap(rootObj); u != "" {
				collected = append(collected, u)
			}
		}
		return deduplicateURLs(collected), true
	}

	return nil, false
}

func extractURLFromMap(m map[string]interface{}) string {
	for _, key := range []string{"url", "URL", "link", "Link", "uri", "URI", "target"} {
		if val, ok := m[key]; ok {
			if str, ok := val.(string); ok && isValidURLString(str) {
				return str
			}
		}
	}
	return ""
}

func parseCSVURLs(data []byte) ([]string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	// Detect header row
	urlColIdx := -1
	headerRow := records[0]
	for idx, col := range headerRow {
		clean := strings.ToLower(strings.TrimSpace(col))
		if clean == "url" || clean == "link" || clean == "uri" || clean == "download" || clean == "target" {
			urlColIdx = idx
			break
		}
	}

	var collected []string
	startRow := 0
	if urlColIdx != -1 {
		startRow = 1
	}

	for r := startRow; r < len(records); r++ {
		row := records[r]
		if urlColIdx != -1 && urlColIdx < len(row) {
			val := strings.TrimSpace(row[urlColIdx])
			if isValidURLString(val) {
				collected = append(collected, val)
			}
			continue
		}
		// No specific column identified: inspect all fields
		for _, field := range row {
			val := strings.TrimSpace(field)
			if isValidURLString(val) {
				collected = append(collected, val)
				break
			}
		}
	}

	return deduplicateURLs(collected), nil
}

func parseTextURLs(r io.Reader) ([]string, error) {
	var urls []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := -1
		for i := 0; i < len(line); i++ {
			if line[i] == '#' && i > 0 && (line[i-1] == ' ' || line[i-1] == '\t') {
				idx = i
				break
			}
		}
		if idx > 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}
		for _, u := range strings.Fields(line) {
			normalized := strings.TrimRight(u, "/")
			if !seen[normalized] {
				seen[normalized] = true
				urls = append(urls, u)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

func isValidURLString(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	parsed, err := url.Parse(s)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return scheme == "http" || scheme == "https" || scheme == "ftp"
}

func deduplicateURLs(raw []string) []string {
	var urls []string
	seen := make(map[string]bool)
	for _, u := range raw {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		normalized := strings.TrimRight(u, "/")
		if !seen[normalized] {
			seen[normalized] = true
			urls = append(urls, u)
		}
	}
	return urls
}
