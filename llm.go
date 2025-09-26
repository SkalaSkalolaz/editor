package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
)

// llmQuery sends a query to the LLM.
// llmQuery отправляет запрос LLM.
func (e *Editor) llmQuery(instruction string) {
	defer func() {
		e.selectAllBeforeLLM = false
		e.ctrlLState = false
	}()
	if strings.TrimSpace(e.llmProvider) == "" {
		e.llmProvider = "ollama"
	}
	if strings.TrimSpace(e.llmModel) == "" {
		e.llmModel = "gemma3:4b"
	}

	payload := instruction
	if cb, err := clipboard.ReadAll(); err == nil {
		cb = strings.TrimSpace(cb)
		if cb != "" {
			payload = payload + "\nData from clipboard:\n" + cb
		}
	}
	if e.selectAllBeforeLLM {
		allText := strings.Join(e.lines, "\n")
		if strings.TrimSpace(allText) != "" {
			payload = payload + "\nExisting text:\n" + allText
		}
	}

	e.statusMessage("Sending request to LLM...")

	out, err := SendMessageToLLM(payload, e.llmProvider, e.llmModel, e.llmKey)
	if err != nil {
		e.showError("LLM error: " + err.Error())
		return
	}

	resp := string(out)
	if strings.TrimSpace(resp) == "" {
		e.showError("LLM returned an empty response")
		return
	}
	e.statusMessage("LLM response received successfully")
	e.insertLLMResponse(resp)
}

// sendCommentToLLM sends a comment to the LLM.
// sendCommentToLLM отправляет комментарий в LLM.
func (e *Editor) sendCommentToLLM() {
	linesAboveCursor := e.lines[:e.cy]
	commentLines := []string{}
	for _, line := range linesAboveCursor {
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "!") || strings.HasPrefix(line, ";") {
			commentLines = append(commentLines, line)
		}
	}
	firstComment := ""
	if len(commentLines) > 0 {
		firstComment = commentLines[0]
	}
	codeContent := strings.Join(e.lines, "\n")
	instruction := "Write code based on this description, but do not write a lengthy explanation; the existing code does not need to be repeated, only in accordance with the instruction; if necessary, only include brief comments before the code:\n"
	if firstComment != "" {
		instruction += firstComment + "\n"
	}
	instruction += "\nThe content of the editable file\n" + codeContent
	e.llmQuery(instruction)
}

// translationPrompt формирует единообразный LLM-промпт для перевода.
func (e *Editor) translationPrompt(sourceText, targetLang string) string {
	return fmt.Sprintf(
		"Text requiring translation: %s, Translate the text to %s, apart from the translated text, nothing else is required of you.",
		sourceText, targetLang)
}

func (e *Editor) llmQueryTranslate(instruction string) (string, error) {
	defer func() {
		e.selectAllBeforeLLM = false
		e.ctrlLState = false
	}()

	if strings.TrimSpace(e.llmProvider) == "" {
		e.llmProvider = "ollama"
	}
	if strings.TrimSpace(e.llmModel) == "" {
		e.llmModel = "gemma3:4b"
	}

	payload := instruction
	e.statusMessage("Sending for translation to the LLM...")

	out, err := SendMessageToLLM(payload, e.llmProvider, e.llmModel, e.llmKey)
	if err != nil {
		return "", fmt.Errorf("LLM error: %w", err)
	}

	resp := string(out)
	if strings.TrimSpace(resp) == "" {
		return "", fmt.Errorf("LLM returned an empty response")
	}
	return resp, nil
}

// insertLLMResponse inserts the LLM response into the editor.
// insertLLMResponse вставляет ответ LLM в редактор.
func (e *Editor) insertLLMResponse(resp string) {
	if e.contextMode {
		e.insertContextualLLMResponse(resp, e.incompleteLine)
		return
	}

	resp = strings.ReplaceAll(resp, "\r\n", "\n")
	respLines := strings.Split(resp, "\n")
	if len(respLines) == 0 {
		return
	}
	if strings.TrimSpace(resp) == "" {
		e.dirty = true
		e.ensureVisible()
		return
	}
	if e.cy < 0 {
		e.cy = 0
	}
	for e.cy >= len(e.lines) {
		e.lines = append(e.lines, "")
	}
	lineRunes := []rune(e.lines[e.cy])
	if e.cx > len(lineRunes) {
		e.cx = len(lineRunes)
	}
	left := string(lineRunes[:e.cx])
	right := ""
	if e.cx < len(lineRunes) {
		right = string(lineRunes[e.cx:])
	}
	e.lines[e.cy] = left + respLines[0] + right
	insertIndex := e.cy + 1
	for i := 1; i < len(respLines); i++ {
		e.lines = append(e.lines[:insertIndex], append([]string{respLines[i]}, e.lines[insertIndex:]...)...)
		insertIndex++
	}
	lastLineIndex := e.cy
	if len(respLines) > 1 {
		lastLineIndex = e.cy + len(respLines) - 1
	}
	if lastLineIndex >= len(e.lines) {
		for lastLineIndex >= len(e.lines) {
			e.lines = append(e.lines, "")
		}
	}
	e.cy = lastLineIndex
	if e.cy >= 0 && e.cy < len(e.lines) {
		e.cx = len([]rune(e.lines[e.cy]))
	}
	e.dirty = true
	e.ensureVisible()
}

func isURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func extractContentFromLLMResponse(body []byte) (string, error) {
	type aiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Content string `json:"content"`
			Text    string `json:"text"`
		} `json:"choices"`
		Text string `json:"text"`
	}
	var r aiResp
	if err := json.Unmarshal(body, &r); err == nil {
		if len(r.Choices) > 0 && r.Choices[0].Message.Content != "" {
			return r.Choices[0].Message.Content, nil
		}
		if r.Choices[0].Content != "" {
			return r.Choices[0].Content, nil
		}
		if r.Choices[0].Text != "" {
			return r.Choices[0].Text, nil
		}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err == nil {
		if t, ok := m["text"].(string); ok && t != "" {
			return t, nil
		}
		if out, ok := m["output"].(string); ok && out != "" {
			return out, nil
		}
		if data, ok := m["data"].(string); ok && data != "" {
			return data, nil
		}
		if c, ok := m["choices"].([]interface{}); ok && len(c) > 0 {
			if first, ok := c[0].(map[string]interface{}); ok {
				if msg, ok := first["message"].(map[string]interface{}); ok {
					if content, ok := msg["content"].(string); ok && content != "" {
						return content, nil
					}
				}
				if text, ok := first["text"].(string); ok && text != "" {
					return text, nil
				}
			}
		}
	}
	return "", errors.New("unable to extract content from LLM response")
}

func sendMessageToLLMUsingURL(endpoint, model, message, apiKey string) (string, error) {
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": message},
		},
		"temperature": 0.2,
		"top_p":       1.0,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	if apiKey != "" {
		if strings.HasPrefix(apiKey, "sn-") {
			req.Header.Set("Authorization", apiKey)
		} else {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM URL request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("LLM URL returned status %d: %s", resp.StatusCode, string(respBody))
	}

	content, err := extractContentFromLLMResponse(respBody)
	if err != nil {
		return "", err
	}
	return content, nil
}

func SendMessageToLLM(message, provider, model, apiKey string) (string, error) {
	if isURL(provider) {
		result, err := sendMessageToLLMUsingURL(provider, model, message, apiKey)
		if err != nil {
			return "", fmt.Errorf("URL provider error: %w", err)
		}
		return result, nil
	}

	parsePollinationsResponse := func(body []byte) (string, error) {
		var m map[string]interface{}
		if err := json.Unmarshal(body, &m); err != nil {
			return "", fmt.Errorf("pollinations: invalid JSON: %w", err)
		}
		if t, ok := m["text"].(string); ok && t != "" {
			return t, nil
		}
		if c, ok := m["content"].(string); ok && c != "" {
			return c, nil
		}
		if choices, ok := m["choices"].([]interface{}); ok && len(choices) > 0 {
			if first, ok := choices[0].(map[string]interface{}); ok {
				if t, ok := first["text"].(string); ok && t != "" {
					return t, nil
				}
				if msg, ok := first["message"].(map[string]interface{}); ok {
					if t, ok := msg["content"].(string); ok && t != "" {
						return t, nil
					}
				}
			}
		}
		if out, ok := m["output"].(string); ok && out != "" {
			return out, nil
		}
		if data, ok := m["data"].(string); ok && data != "" {
			return data, nil
		}
		return "", errors.New("pollinations: could not recognize the response text")
	}

	parseOllamaResponse := func(body []byte) (string, error) {
		type ollamaChatMessage struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		type ollamaChoice struct {
			Message ollamaChatMessage `json:"message"`
		}
		type ollamaResponse struct {
			Choices []ollamaChoice `json:"choices"`
		}
		var r ollamaResponse
		if err := json.Unmarshal(body, &r); err == nil {
			if len(r.Choices) > 0 && r.Choices[0].Message.Content != "" {
				return r.Choices[0].Message.Content, nil
			}
		}
		var f map[string]interface{}
		if err := json.Unmarshal(body, &f); err == nil {
			if t, ok := f["text"].(string); ok && t != "" {
				return t, nil
			}
			if t, ok := f["data"].(string); ok && t != "" {
				return t, nil
			}
		}
		return "", errors.New("ollama: could not recognize the response text")
	}

	sendPollinations := func(apiKeyArg string) (string, error) {
		apiKey = apiKeyArg
		if apiKey == "" {
			apiKey = os.Getenv("POLLINATIONS_API_KEY")
		}
		url := "https://text.pollinations.ai/openai"
		type pollinationsMessage struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		type pollinationsRequestBody struct {
			Model    string                `json:"model"`
			Messages []pollinationsMessage `json:"messages"`
			Seed     int                   `json:"seed"`
		}

		body := pollinationsRequestBody{
			Model: model,
			Messages: []pollinationsMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: message},
			},
			Seed: 42,
		}

		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("pollinations: failed to construct the request body: %w", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("pollinations: failed to create the request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("pollinations: error net: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("pollinations: failed to read the response: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("pollinations: status %d: %s", resp.StatusCode, string(respBody))
		}
		parsed, err := parsePollinationsResponse(respBody)
		if err != nil {
			return "", fmt.Errorf("pollinations: failed to parse the response: %w", err)
		}
		return parsed, nil
	}

	sendOpenRouter := func(apiKeyArg string) (string, error) {
		baseURL := os.Getenv("OPENROUTER_BASE_URL")
		if baseURL == "" {
			baseURL = "https://openrouter.ai/api/v1"
		}
		apiKey = apiKeyArg
		if apiKey == "" {
			apiKey = os.Getenv("OPENROUTER_API_KEY")
		}
		url := baseURL + "/chat/completions"
		payload := map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "user", "content": message},
			},
			"temperature": 0.2,
			"top_p":       1.0,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")

		if apiKey != "" {
			if strings.HasPrefix(apiKey, "sn-") {
				req.Header.Set("Authorization", apiKey)
			} else {
				req.Header.Set("Authorization", "Bearer "+apiKey)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("LLM URL request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("LLM URL returned status %d: %s", resp.StatusCode, string(respBody))
		}

		content, err := extractContentFromLLMResponse(respBody)
		if err != nil {
			return "", err
		}
		return content, nil
	}

	sendLLM7 := func(apiKeyArg string) (string, error) {
		baseURL := os.Getenv("LLM7_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.llm7.io/v1"
		}
		apiKey = apiKeyArg
		if apiKey == "" {
			apiKey = os.Getenv("LLM7_API_KEY")
		}
		url := baseURL + "/chat/completions"

		type llm7Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		type llm7Request struct {
			Model       string        `json:"model"`
			Messages    []llm7Message `json:"messages"`
			Temperature float64       `json:"temperature"`
			TopP        float64       `json:"top_p"`
		}

		body := llm7Request{
			Model: model,
			Messages: []llm7Message{
				{Role: "user", Content: message},
			},
			Temperature: 0.2,
			TopP:        1.0,
		}

		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("llm7: failed to form the request body: %w", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("llm7: failed to create the request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		} else {
			apiKey = "unused"
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("llm7: error net: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("llm7: failed to read the response: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("llm7: status %d: %s", resp.StatusCode, string(respBody))
		}

		type llm7Response struct {
			Choices []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		var r llm7Response
		if err := json.Unmarshal(respBody, &r); err == nil {
			if len(r.Choices) > 0 && r.Choices[0].Message.Content != "" {
				return r.Choices[0].Message.Content, nil
			}
		}
		var f map[string]interface{}
		if err := json.Unmarshal(respBody, &f); err == nil {
			if choices, ok := f["choices"].([]interface{}); ok && len(choices) > 0 {
				if first, ok := choices[0].(map[string]interface{}); ok {
					if msg, ok := first["message"].(map[string]interface{}); ok {
						if t, ok := msg["content"].(string); ok && t != "" {
							return t, nil
						}
					}
					if t, ok := first["text"].(string); ok && t != "" {
						return t, nil
					}
				}
			}
			if t, ok := f["text"].(string); ok && t != "" {
				return t, nil
			}
			if t, ok := f["data"].(string); ok && t != "" {
				return t, nil
			}
		}
		return "", errors.New("llm7: failed to recognize the response text")
	}

	sendOllama := func() (string, error) {
		url := "http://localhost:11434/v1/chat/completions"

		reqBody := map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "user", "content": message},
			},
			"temperature": 0.2,
			"top_p":       1.0,
		}
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("ollama: could not generate the request body: %w", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("ollama: failed to create the request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(context.Background(), 480*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("ollama: error net: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("ollama: reading the response failed: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("ollama: status %d: %s", resp.StatusCode, string(respBody))
		}

		parsed, err := parseOllamaResponse(respBody)
		if err != nil {
			return "", fmt.Errorf("ollama: failed to parse the response: %w", err)
		}
		return parsed, nil
	}

	switch provider {
	case "pollinations":
		result, err := sendPollinations(apiKey)
		if err != nil {
			return "", fmt.Errorf("Pollinations error: %w", err)
		}
		return result, nil
	case "llm7":
		result, err := sendLLM7(apiKey)
		if err != nil {
			return "", fmt.Errorf("LLM7 error: %w", err)
		}
		return result, nil
	case "openrouter":
		result, err := sendOpenRouter(apiKey)
		if err != nil {
			return "", fmt.Errorf("OpenRouter error: %w", err)
		}
		return result, nil
	case "ollama":
		result, err := sendOllama()
		if err != nil {
			return "", fmt.Errorf("Ollama error: %w", err)
		}
		return result, nil
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func nameModelPollinations() {
	resp, err := http.Get("https://text.pollinations.ai/models")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var models []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	err = json.Unmarshal(body, &models)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Pollinations models:\n")
	for _, model := range models {
		fmt.Printf(" %-40s  %s\n", model.Name, model.Description)
	}
}

func nameModelLlm7() {
	resp, err := http.Get("https://api.llm7.io/v1/models")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	type Mod struct {
		ID         string `json:"id"`
		Modalities struct {
			Input []string `json:"input"`
		} `json:"modalities"`
	}

	var models []Mod
	if err := json.Unmarshal(body, &models); err == nil {
		fmt.Printf("Lmm7 models:\n")
		for _, m := range models {
			desc := "Not specified"
			if len(m.Modalities.Input) > 0 {
				desc = strings.Join(m.Modalities.Input, ", ")
			}
			fmt.Printf(" %-40s %s\n", m.ID, desc)
		}
		return
	}

	var wrapper struct {
		Models []Mod `json:"models"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil {
		fmt.Printf("Lmm7 models:\n")
		for _, m := range wrapper.Models {
			desc := "Not specified"
			if len(m.Modalities.Input) > 0 {
				desc = strings.Join(m.Modalities.Input, ", ")
			}
			fmt.Printf(" %-40s %s\n", m.ID, desc)
		}
		return
	}

	fmt.Println("Failed to parse the response")
}

func nameModelOpenRouter() {
	url := "https://openrouter.ai/api/v1/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	type ApiMod struct {
		ID            string `json:"id"`
		ContextLength int    `json:"context_length"`
		Architecture  struct {
			InputModalities  []string `json:"input_modalities"`
			OutputModalities []string `json:"output_modalities"`
		} `json:"architecture"`
	}
	type DataWrapper struct {
		Data []ApiMod `json:"data"`
	}

	var dw DataWrapper
	if err := json.Unmarshal(body, &dw); err != nil {
		fmt.Println("Failed to parse the answer:", err)
		return
	}
	if len(dw.Data) == 0 {
		fmt.Println("No data of models")
		return
	}

	fmt.Printf("OpenRouter models:\n")
	for _, m := range dw.Data {
		in := "Not specified"
		if len(m.Architecture.InputModalities) > 0 {
			in = strings.Join(m.Architecture.InputModalities, ", ")
		}
		out := "Not specified"
		if len(m.Architecture.OutputModalities) > 0 {
			out = strings.Join(m.Architecture.OutputModalities, ", ")
		}
		fmt.Printf(" %-40s context=%d inputs=[%s] outputs=[%s]\n", m.ID, m.ContextLength, in, out)
	}
}

// llmQueryWithProjectContext отправляет запрос в LLM со всем контекстом проекта
func (e *Editor) llmQueryWithProjectContext(instruction string) {
	defer func() {
		e.selectAllBeforeLLM = false
		e.ctrlLState = false
	}()

	if strings.TrimSpace(e.llmProvider) == "" {
		e.llmProvider = "ollama"
	}
	if strings.TrimSpace(e.llmModel) == "" {
		e.llmModel = "gemma3:4b"
	}

	// Убедимся, что все изменения синхронизированы перед сбором контекста
	e.syncEditorToCanvas()

	e.statusMessage("Building project context...")
	projectContext := e.buildProjectContext(instruction)

	// Проверим, что файлы действительно собраны
	if len(projectContext.Files) == 0 {
		e.showError("No project files found to send to LLM")
		return
	}

	payload := e.formatProjectContextForLLM(projectContext)

	if cb, err := clipboard.ReadAll(); err == nil {
		cb = strings.TrimSpace(cb)
		if cb != "" {
			payload = payload + "\n\nAdditional data from clipboard:\n" + cb
		}
	}

	e.statusMessage("Sending project context to LLM...")

	out, err := SendMessageToLLM(payload, e.llmProvider, e.llmModel, e.llmKey)
	if err != nil {
		e.showError("LLM error: " + err.Error())
		return
	}

	resp := string(out)
	if strings.TrimSpace(resp) == "" {
		e.showError("LLM returned an empty response")
		return
	}
	e.statusMessage("LLM response received successfully")
	e.insertLLMResponse(resp)
}
