package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIKey string `json:"api_key"`
}

type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".woman", "config.json")
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	return &cfg, err
}

func saveConfig(cfg *Config) error {
	path := configPath()
	os.MkdirAll(filepath.Dir(path), 0700)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func askAPIKey() string {
	var key string
	fmt.Print("🔐 Enter Gemini API Key: ")
	fmt.Scanln(&key)
	return key
}

func ensureConfig() *Config {
	cfg, err := loadConfig()
	if err == nil && cfg.APIKey != "" {
		return cfg
	}

	fmt.Println("👋 First run setup")
	key := askAPIKey()

	cfg = &Config{APIKey: key}
	saveConfig(cfg)

	fmt.Println("✅ Saved!")
	return cfg
}

func callGemini(apiKey, command string) (string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash:generateContent?key=" + apiKey

	prompt := fmt.Sprintf(`
Explain CLI command: %s

Rules:
- max 10 lines
- no markdown
- simple and clear
- sections:
  What it is
  Usage
  Flags (max 3)
  Example (1 only)
- use emojis for sections
`, command)

	reqBody := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var geminiResp GeminiResponse
	err = json.Unmarshal(body, &geminiResp)
	if err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func cleanOutput(s string) string {
	replacer := strings.NewReplacer(
		"#", "",
		"##", "",
		"###", "",
		"*", "",
		"`", "",
	)

	return strings.TrimSpace(replacer.Replace(s))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("❌ Usage: woman <command>")
		fmt.Println("Example: woman ls")
		return
	}

	command := os.Args[1]

	cfg := ensureConfig()

	result, err := callGemini(cfg.APIKey, command)
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}

	fmt.Println(cleanOutput(result))
}
