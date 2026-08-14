// llm.go
// Назначение: Интеграция с Large Language Models (ИИ-моделями) для автодополнения и анализа кода.
// Использует пакет llmclient для унифицированного взаимодействия с различными LLM-провайдерами.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/SkalaSkalolaz/llmclient"
	"github.com/atotto/clipboard"
)

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
	payload := e.llmContext.BuildPayload(instruction, e.selectAllBeforeLLM, e)

	_ = e.sendPayloadToLLM(payload)
}

func (lc *LLMContext) BuildPayload(baseInstruction string, includeEditorContent bool, editor *Editor) string {
	var sb strings.Builder

	sb.WriteString(baseInstruction)
	sb.WriteString("\n\n")

	if lc.UseInputFiles && len(lc.InputFiles) > 0 {
		sb.WriteString("INPUT FILES CONTENT:\n")
		sb.WriteString("====================\n\n")

		for filename, content := range lc.InputFiles {
			sb.WriteString(fmt.Sprintf("--- FILE: %s ---\n", filename))
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}
	}

	if includeEditorContent && editor != nil {
		allText := strings.Join(editor.lines, "\n")
		if strings.TrimSpace(allText) != "" {
			sb.WriteString("EXISTING EDITOR CONTENT:\n")
			sb.WriteString(allText)
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

func (e *Editor) llmQueryWithClipboard(instruction string) {
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

	var payload strings.Builder
	payload.WriteString(instruction)
	payload.WriteString("\n\n")

	if cb := getClipboardData(); cb != "" {
		payload.WriteString("DATA FROM CLIPBOARD:\n")
		payload.WriteString(cb)
		payload.WriteString("\n\n")
	}

	if e.llmContext.UseInputFiles && len(e.llmContext.InputFiles) > 0 {
		payload.WriteString("INPUT FILES CONTENT:\n")
		payload.WriteString("====================\n\n")

		for filename, content := range e.llmContext.InputFiles {
			payload.WriteString(fmt.Sprintf("--- FILE: %s ---\n", filename))
			payload.WriteString(content)
			payload.WriteString("\n\n")
		}
	}

	if e.selectAllBeforeLLM {
		allText := strings.Join(e.lines, "\n")
		if strings.TrimSpace(allText) != "" {
			payload.WriteString("EXISTING TEXT:\n")
			payload.WriteString(allText)
			payload.WriteString("\n\n")
		}
	}

	_ = e.sendPayloadToLLM(payload.String())
}

func (e *Editor) sendPayloadToLLM(payload string) error {
	if strings.TrimSpace(payload) == "" {
		return fmt.Errorf("empty payload provided to LLM")
	}

	e.statusMessage("Sending request to LLM...")

	out, err := SendMessageToLLM(payload, e.llmProvider, e.llmModel, e.llmKey)
	if err != nil {
		e.showError("LLM error: " + err.Error())
		return err
	}

	resp := e.validateLLMResponse(string(out))
	if strings.TrimSpace(resp) == "" {
		e.showError("LLM returned an empty response after validation")
		return fmt.Errorf("empty LLM response after validation")
	}

	e.statusMessage("LLM response received successfully")
	e.insertLLMResponse(resp)
	return nil
}

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

func SendMessageToLLM(message, provider, model, apiKey string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	provider, endpoint := resolveProviderAndEndpoint(provider)

	opts := []llmclient.SendOption{
		llmclient.WithTemperature(0.2),
	}
	if endpoint != "" {
		opts = append(opts, llmclient.WithEndpoint(endpoint))
	}

	var response string
	var err error

	if endpoint != "" {
		response, err = llmclient.SendWithContext(ctx, "custom", model, apiKey, "You are a helpful assistant.", message, opts...)
	} else {
		response, err = llmclient.SendWithContext(ctx, provider, model, apiKey, "You are a helpful assistant.", message, opts...)
	}

	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}

	return response, nil
}

func resolveProviderAndEndpoint(provider string) (string, string) {
	switch provider {
	case "pollinations":
		return "pollinations", ""
	case "openrouter":
		baseURL := os.Getenv("OPENROUTER_BASE_URL")
		if baseURL == "" {
			baseURL = "https://openrouter.ai/api/v1"
		}
		return "openrouter", baseURL + "/chat/completions"
	case "llm7":
		baseURL := os.Getenv("LLM7_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.llm7.io/v1"
		}
		return "openai-compatible", baseURL + "/chat/completions"
	case "ollama":
		return "ollama", ""
	default:
		if isURL(provider) {
			return "custom", provider
		}
		return provider, ""
	}
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func extractContentFromPossibleJSON(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty response")
	}

	reFenced := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	if m := reFenced.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}

	var obj interface{}
	if err := json.Unmarshal([]byte(s), &obj); err == nil {
		if content, ok := findContentRecursive(obj); ok {
			content = strings.TrimSpace(content)
			if content != "" {
				return content, nil
			}
		}
	}

	first := strings.IndexAny(s, "{[")
	lastBrace := strings.LastIndex(s, "}")
	lastBracket := strings.LastIndex(s, "]")
	last := lastBrace
	if lastBracket > last {
		last = lastBracket
	}

	if first != -1 && last > first {
		jsonStr := s[first : last+1]
		var innerObj interface{}
		if err := json.Unmarshal([]byte(jsonStr), &innerObj); err == nil {
			if content, ok := findContentRecursive(innerObj); ok {
				content = strings.TrimSpace(content)
				if content != "" {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no JSON content found")
}

func findContentRecursive(v interface{}) (string, bool) {
	switch t := v.(type) {
	case map[string]interface{}:
		priorityFields := []string{"content", "text", "message", "result", "output", "data"}
		for _, field := range priorityFields {
			if val, exists := t[field]; exists {
				if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
					return s, true
				}
			}
		}

		if choices, exists := t["choices"]; exists {
			if choicesSlice, ok := choices.([]interface{}); ok && len(choicesSlice) > 0 {
				if firstChoice, ok := choicesSlice[0].(map[string]interface{}); ok {
					if message, exists := firstChoice["message"]; exists {
						if messageMap, ok := message.(map[string]interface{}); ok {
							if content, exists := messageMap["content"]; exists {
								if s, ok := content.(string); ok && strings.TrimSpace(s) != "" {
									return s, true
								}
							}
						}
					}
					if delta, exists := firstChoice["delta"]; exists {
						if deltaMap, ok := delta.(map[string]interface{}); ok {
							if content, exists := deltaMap["content"]; exists {
								if s, ok := content.(string); ok && strings.TrimSpace(s) != "" {
									return s, true
								}
							}
						}
					}
					if text, exists := firstChoice["text"]; exists {
						if s, ok := text.(string); ok && strings.TrimSpace(s) != "" {
							return s, true
						}
					}
					if content, exists := firstChoice["content"]; exists {
						if s, ok := content.(string); ok && strings.TrimSpace(s) != "" {
							return s, true
						}
					}
				}
			}
		}

		for _, val := range t {
			if s, ok := findContentRecursive(val); ok {
				return s, true
			}
		}

	case []interface{}:
		for _, item := range t {
			if s, ok := findContentRecursive(item); ok {
				return s, true
			}
		}

	case string:
		str := strings.TrimSpace(t)
		if (strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}")) ||
			(strings.HasPrefix(str, "[") && strings.HasSuffix(str, "]")) {
			var inner interface{}
			if err := json.Unmarshal([]byte(str), &inner); err == nil {
				if s, ok := findContentRecursive(inner); ok {
					return s, true
				}
			}
		}
	}

	return "", false
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

	e.syncEditorToCanvas()

	e.statusMessage("Building project context...")
	projectContext := e.buildProjectContext(instruction)

	if len(projectContext.Files) == 0 {
		e.showError("No project files found to send to LLM")
		return
	}

	payload := e.formatProjectContextForLLM(projectContext)

	_ = e.sendPayloadToLLM(payload)
}

func ProcessStreamingLLM(provider, model, apiKey string, useClipboardData bool, inputFiles string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}

	instruction := strings.TrimSpace(string(input))
	var clipboardData string
	if useClipboardData {
		clipboardData = getClipboardData()
	}
	fullPayload := processStreamInput(instruction, clipboardData, inputFiles)

	if strings.TrimSpace(fullPayload) == "" {
		return fmt.Errorf("empty input provided")
	}

	done := make(chan bool, 1)
	go showThinkingIndicator(done)
	defer func() { done <- true }()

	response, err := SendMessageToLLM(fullPayload, provider, model, apiKey)
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}

	fmt.Print(response)
	return nil
}

func showThinkingIndicator(done chan bool) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	for {
		select {
		case <-done:
			fmt.Fprint(os.Stderr, "\r\r")
			return
		default:
			frame := frames[i%len(frames)]
			fmt.Fprintf(os.Stderr, "\r%s LLM is thinking...", frame)

			i++
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func processStreamInput(instruction, streamData, inputFiles string) string {
	var payload strings.Builder
	payload.WriteString(instruction)
	payload.WriteString("\n\n")
	if streamData != "" {
		payload.WriteString("DATA FROM CLIPBOARD:\n")
		payload.WriteString(streamData)
		payload.WriteString("\n\n")
	}
	if inputFiles != "" {
		files, err := readInputFiles(inputFiles)
		if err == nil && len(files) > 0 {
			payload.WriteString("INPUT FILES CONTENT:\n")
			payload.WriteString("====================\n\n")

			for filename, content := range files {
				payload.WriteString(fmt.Sprintf("--- FILE: %s ---\n", filename))
				payload.WriteString(content)
				payload.WriteString("\n\n")
			}
		}
	}

	return payload.String()
}

func readInputFiles(inputPath string) (map[string]string, error) {
	files := make(map[string]string)

	info, err := os.Stat(inputPath)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return readProjectFiles(inputPath)
	} else {
		content, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, err
		}
		files[filepath.Base(inputPath)] = string(content)
	}

	return files, nil
}

func getClipboardData() string {
	data, err := clipboard.ReadAll()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(data)
}
