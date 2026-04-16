package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type minimaxProvider struct {
	config Config
}

func newMinimaxProvider(cfg Config) *minimaxProvider {
	return &minimaxProvider{config: cfg}
}

type minimaxMessage struct {
	Role    string `json:"role"`
	Name    string `json:"name,omitempty"`
	Content string `json:"content"`
}

type minimaxRequest struct {
	Model    string           `json:"model"`
	Messages []minimaxMessage `json:"messages"`
}

type minimaxResponse struct {
	Choices []struct {
		Message minimaxMessage `json:"message"`
	} `json:"choices"`
}

func stripThinking(text string) string {
	start := strings.Index(text, "<think>")
	end := strings.Index(text, "</think>")

	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(text[end+len("</think>"):])
	}
	return strings.TrimSpace(text)
}

func (p *minimaxProvider) GenerateHealthAnalysis(prompt string) (string, error) {
	if p.config.APIKey == "" {
		return "", fmt.Errorf("AI_API_KEY is required")
	}

	url := "https://api.minimax.io/v1/text/chatcompletion_v2"
	if p.config.BaseURL != "" {
		url = p.config.BaseURL + "/v1/text/chatcompletion_v2"
	}

	model := p.config.Model
	if model == "" {
		model = "MiniMax-M2.7"
	}

	// System prompt untuk format output yang rapi dan konsisten
	systemPrompt := `Anda adalah asisten analisis kesehatan anak yang profesional.

ATURAN OUTPUT (WAJIB DIIKUTI):
1. Gunakan Markdown yang bersih dan terstruktur
2. Header maksimal H3 (###)
3. Bullet points gunakan (-) bukan (*)
4. Tabel gunakan | separator yang rata
5. Beri jarak antar section dengan newline
6. Jangan gunakan emoji berlebihan
7. Highlight nilai penting dengan **bold**
8. Selalu tambahkan disclaimer медиум di akhir

FORMAT WAJUR:
## Judul Analisis

### Ringkasan Data
| Parameter | Nilai | Status |

### Saran & Rekomendasi
- Poin 1
- Poin 2

### Disclaimer
*Disclaimer: bukan diagnosis medis...*`

	reqBody := minimaxRequest{
		Model: model,
		Messages: []minimaxMessage{
			{Role: "system", Name: "System", Content: systemPrompt},
			{Role: "user", Name: "User", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	if p.config.GroupID != "" {
		req.Header.Add("Group-Id", p.config.GroupID)
	}

	resp, err := sharedClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("minimax returned status %d: %s", resp.StatusCode, string(body))
	}

	var result minimaxResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// Hapus tag <think> dari response
	rawContent := result.Choices[0].Message.Content
	return stripThinking(rawContent), nil
}
