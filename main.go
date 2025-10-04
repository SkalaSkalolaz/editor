package main

// go build -buildvcs=false -o editor .

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

// Version of the editor.
// Версия редактора.
const Version = "0.9.13"

// Editor represents the text editor state.
// Editor представляет состояние текстового редактора.
type Editor struct {
	screen              tcell.Screen
	filename            string
	lines               []string
	cx, cy              int
	offsetX             int
	offsetY             int
	dirty               bool
	clipboard           string
	prompt              *Prompt
	multiLinePrompt     *MultiLinePrompt
	quit                bool
	width, height       int
	llmProvider         string
	llmModel            string
	llmKey              string
	canvasWidth         int
	contentWidth        int
	contentHeight       int
	language            Language
	selectAllBeforeLLM  bool
	ctrlAState          bool
	ctrlLState          bool
	selectStartX        int
	selectStartY        int
	selecting           bool
	lineSelecting       bool
	terminalPrompt      *TerminalPrompt
	llmLastPrompt       string
	errorMessage        string
	errorShowTime       time.Time
	lastSearch          string
	llmPrefill          string
	undoStack           []EditorState
	redoStack           []EditorState
	bracketMatcher      *BracketMatcher
	contextMode         bool
	incompleteLine      bool
	canvases            map[int]*Canvas
	currentCanvas       int
	canvasWarningTime   time.Time
	githubProject       *GitHubProject
	showLineNumbers     bool
	lineNumbersWidth    int
	showStructurePanel  bool
	structurePanelWidth int
}

// ProjectContext представляет контекст всего проекта для отправки в LLM
type ProjectContext struct {
	ProjectStructure string            `json:"project_structure"`
	Files            map[string]string `json:"files"`
	CurrentFile      string            `json:"current_file"`
	Instruction      string            `json:"instruction"`
}

// detectLanguage detects the language based on the file extension.
// detectLanguage определяет язык на основе расширения файла.
func detectLanguage(filename string) Language {
	ext := strings.ToLower(filepathExtNew(filename))
	switch ext {
	case ".c", ".h":
		return LangC
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh":
		return LangCpp
	case ".s", ".asm":
		return LangAssembly
	case ".f", ".for", ".f90", ".f95", ".f03":
		return LangFortran
	case ".go":
		return LangGo
	case ".py":
		return LangPython
	case ".rb":
		return LangRuby
	case ".kt", ".kts":
		return LangKotlin
	case ".swift":
		return LangSwift
	case ".html", ".htm":
		return LangHTML
	case ".lisp", ".lsp", ".cl", ".el":
		return LangLisp
	default:
		return LangUnknown
	}
}

// NewEditor creates a new Editor instance.
// NewEditor создает новый экземпляр Editor.
func NewEditor(path string, provider string, model string) *Editor {
	e := &Editor{
		filename:      path,
		lines:         []string{""},
		dirty:         false,
		quit:          false,
		language:      LangUnknown,
		canvases:      make(map[int]*Canvas),
		currentCanvas: 1,
	}
	canvas := &Canvas{
		filename: path,
		lines:    []string{""},
		cx:       0,
		cy:       0,
		offsetX:  0,
		offsetY:  0,
		dirty:    false,
		language: LangUnknown,
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			content := string(data)
			content = strings.ReplaceAll(content, "\r\n", "\n")
			canvas.lines = strings.Split(content, "\n")
			canvas.language = detectLanguage(path)
		} else {
			canvas.lines = []string{""}
		}
	}

	e.canvases[1] = canvas
	e.syncCanvasToEditor()

	e.contentWidth = 115
	e.contentHeight = 35
	e.canvasWidth = e.contentWidth
	e.width = e.contentWidth
	e.height = e.contentHeight
	e.llmProvider = provider
	e.llmModel = model
	e.canvasWidth = 0
	e.llmLastPrompt = ""
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			content := string(data)
			content = strings.ReplaceAll(content, "\r\n", "\n")
			e.lines = strings.Split(content, "\n")
			e.language = detectLanguage(path)
		} else {
			e.lines = []string{""}
		}
	}
	e.cx, e.cy = 0, 0
	e.offsetX, e.offsetY = 0, 0
	e.bracketMatcher = NewBracketMatcher(e)
	return e
}

// readProjectFiles reads all supported files from a directory
// readProjectFiles читает все поддерживаемые файлы из директории
func readProjectFiles(dirPath string) (map[string]string, error) {
	files := make(map[string]string)

	// Supported file extensions for programming languages
	supportedExts := map[string]bool{
		".c": true, ".h": true,
		".cpp": true, ".cc": true, ".cxx": true, ".hpp": true, ".hh": true,
		".s": true, ".asm": true,
		".f": true, ".for": true, ".f90": true, ".f95": true, ".f03": true,
		".go": true,
		".py": true,
		".rb": true,
		".kt": true, ".kts": true,
		".swift": true,
		".html":  true, ".htm": true,
		".lisp": true, ".lsp": true, ".cl": true, ".el": true,
	}

	projectFiles := map[string]bool{
		"README.md": true, "README": true, "README.txt": true,
		"LICENSE": true, "LICENSE.txt": true, "COPYING": true,
		"CREDITS.md": true, "CREDITS": true, "CREDITS.txt": true,
		"Makefile": true, "makefile": true,
		"Dockerfile": true,
		".gitignore": true,
		"go.mod":     true, "go.sum": true,
		"package.json": true, "package-lock.json": true,
		"requirements.txt": true, "Pipfile": true,
		"Cargo.toml": true, "Cargo.lock": true,
		"pom.xml": true, "build.gradle": true, "build.gradle.kts": true,
		"CMakeLists.txt": true,
		".env":           true, ".env.example": true,
		"docker-compose.yml": true, "docker-compose.yaml": true,
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." && info.Name() != ".." {
				return filepath.SkipDir
			}
			return nil
		}

		filename := info.Name()
		ext := strings.ToLower(filepath.Ext(filename))

		if supportedExts[ext] {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(dirPath, path)
			files[relPath] = string(content)
			return nil
		}

		if projectFiles[filename] || projectFiles[strings.ToLower(filename)] {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(dirPath, path)
			files[relPath] = string(content)
			return nil
		}

		if filename == "Makefile" || filename == "makefile" || filename == "Dockerfile" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(dirPath, path)
			files[relPath] = string(content)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// NewEditorWithProject creates a new Editor instance with project files
// NewEditorWithProject создает новый экземпляр Editor с файлами проекта
func NewEditorWithProject(dirPath string, provider string, model string) *Editor {
	e := &Editor{
		filename:      dirPath,
		lines:         []string{""},
		dirty:         false,
		quit:          false,
		language:      LangUnknown,
		canvases:      make(map[int]*Canvas),
		currentCanvas: 1,
	}

	canvas := &Canvas{
		filename: dirPath,
		lines:    []string{""},
		cx:       0,
		cy:       0,
		offsetX:  0,
		offsetY:  0,
		dirty:    false,
		language: LangUnknown,
	}

	projectFiles, err := readProjectFiles(dirPath)
	if err != nil {
		canvas.lines = []string{"Error reading project: " + err.Error(), ""}
	} else {
		canvas.lines = createProjectOverview(projectFiles)
	}

	e.canvases[1] = canvas
	e.syncCanvasToEditor()

	e.contentWidth = 115
	e.contentHeight = 35
	e.canvasWidth = e.contentWidth
	e.width = e.contentWidth
	e.height = e.contentHeight
	e.llmProvider = provider
	e.llmModel = model
	e.canvasWidth = 0
	e.llmLastPrompt = ""

	e.cx, e.cy = 0, 0
	e.offsetX, e.offsetY = 0, 0
	e.bracketMatcher = NewBracketMatcher(e)

	e.createCanvasesForProjectFiles(projectFiles, dirPath)

	return e
}

// createProjectOverview creates a formatted overview of project files
// createProjectOverview создает форматированный обзор файлов проекта
func createProjectOverview(files map[string]string) []string {
	lines := []string{
		"PROJECT OVERVIEW",
		"================",
		"",
		"Files found: " + strconv.Itoa(len(files)),
		"",
	}

	// Sort filenames for consistent display
	var filenames []string
	for filename := range files {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)

	sourceFiles := []string{}
	configFiles := []string{}
	docFiles := []string{}

	for _, filename := range filenames {
		ext := strings.ToLower(filepath.Ext(filename))
		lowerName := strings.ToLower(filename)

		switch {
		case isSourceFile(ext):
			sourceFiles = append(sourceFiles, filename)
		case isConfigFile(filename) || strings.Contains(lowerName, "config") ||
			strings.Contains(lowerName, "makefile") || strings.Contains(lowerName, "docker"):
			configFiles = append(configFiles, filename)
		case strings.Contains(lowerName, "readme") || strings.Contains(lowerName, "license") ||
			strings.Contains(lowerName, "copying"):
			docFiles = append(docFiles, filename)
		default:
			configFiles = append(configFiles, filename)
		}
	}

	if len(sourceFiles) > 0 {
		lines = append(lines, "SOURCE FILES:")
		lines = append(lines, "-------------")
		for _, file := range sourceFiles {
			lines = append(lines, "  • "+file)
		}
		lines = append(lines, "")
	}

	if len(configFiles) > 0 {
		lines = append(lines, "CONFIGURATION FILES:")
		lines = append(lines, "---------------------")
		for _, file := range configFiles {
			lines = append(lines, "  • "+file)
		}
		lines = append(lines, "")
	}

	if len(docFiles) > 0 {
		lines = append(lines, "DOCUMENTATION:")
		lines = append(lines, "--------------")
		for _, file := range docFiles {
			lines = append(lines, "  • "+file)
		}
		lines = append(lines, "")
	}

	lines = append(lines, "Navigation: Use Ctrl+B to switch between files")
	lines = append(lines, "Press Ctrl+O and type filename to open specific file")
	lines = append(lines, "Press Ctrl+N: Create a new file in the canvas")

	return lines
}

// isSourceFile checks if file extension indicates a source code file
// isSourceFile проверяет, указывает ли расширение файла на файл исходного кода
func isSourceFile(ext string) bool {
	sourceExts := []string{".c", ".h", ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".s", ".asm",
		".f", ".for", ".f90", ".f95", ".f03", ".go", ".py", ".rb",
		".kt", ".kts", ".swift", ".html", ".htm", ".lisp", ".lsp", ".cl", ".el"}
	for _, e := range sourceExts {
		if ext == e {
			return true
		}
	}
	return false
}

// isConfigFile checks if filename indicates a configuration file
// isConfigFile проверяет, указывает ли имя файла на файл конфигурации
func isConfigFile(filename string) bool {
	configFiles := []string{"Makefile", "makefile", "Dockerfile", ".gitignore", "go.mod",
		"go.sum", "package.json", "package-lock.json", "requirements.txt",
		"Pipfile", "Cargo.toml", "Cargo.lock", "pom.xml", "build.gradle",
		"build.gradle.kts", "CMakeLists.txt", ".env", ".env.example",
		"docker-compose.yml", "docker-compose.yaml"}
	for _, f := range configFiles {
		if strings.EqualFold(filename, f) {
			return true
		}
	}
	return false
}

// createCanvasesForProjectFiles creates canvases for each project file
// createCanvasesForProjectFiles создает канвасы для каждого файла проекта
func (e *Editor) createCanvasesForProjectFiles(files map[string]string, basePath string) {
	canvasNum := 2 // Start from 2 since 1 is for overview

	for filename, content := range files {
		if canvasNum > MaxCanvases {
			break
		}

		fullPath := filepath.Join(basePath, filename)
		language := detectLanguage(fullPath)

		canvas := &Canvas{
			filename: fullPath,
			lines:    strings.Split(content, "\n"),
			cx:       0,
			cy:       0,
			offsetX:  0,
			offsetY:  0,
			dirty:    false,
			language: language,
		}

		e.canvases[canvasNum] = canvas
		canvasNum++
	}
}

func (e *Editor) handleExitWithCanvasCheck() {
	exitManager := NewExitManager(e)

	if exitManager.checkAllCanvases() {
		e.prompt = nil
		e.statusMessage(fmt.Sprintf("Found %d canvas(es) with unsaved changes", len(exitManager.canvasesToSave)))
		if len(exitManager.canvasesToSave) > 0 {
			exitManager.promptForCanvasSave(exitManager.canvasesToSave[0])
		}
	} else {
		e.quit = true
	}
}

// buildProjectContext собирает контекст всего проекта
func (e *Editor) buildProjectContext(instruction string) *ProjectContext {
	context := &ProjectContext{
		Files:       make(map[string]string),
		Instruction: instruction,
		CurrentFile: e.filename,
	}

	if e.githubProject != nil {
		return e.buildGitHubProjectContext(instruction)
	}

	var structure []string
	structure = append(structure, "PROJECT STRUCTURE:")
	structure = append(structure, "=================")

	e.syncEditorToCanvas()

	for canvasNum, canvas := range e.canvases {
		if canvas.filename != "" {
			filename := canvas.filename
			if e.filename != "" {
				if relPath, err := filepath.Rel(filepath.Dir(e.filename), filename); err == nil {
					filename = relPath
				}
			} else if e.githubProject != nil && e.githubProject.LocalPath != "" {
				if relPath, err := filepath.Rel(e.githubProject.LocalPath, filename); err == nil {
					filename = relPath
				}
			}

			structure = append(structure, fmt.Sprintf("Canvas %d: %s", canvasNum, filename))
			content := strings.Join(canvas.lines, "\n")
			context.Files[filename] = content
		}
	}
	context.ProjectStructure = strings.Join(structure, "\n")

	return context
}

// buildGitHubProjectContext собирает контекст GitHub проекта
func (e *Editor) buildGitHubProjectContext(instruction string) *ProjectContext {
	context := &ProjectContext{
		Files:       make(map[string]string),
		Instruction: instruction,
		CurrentFile: e.filename,
	}

	if e.githubProject == nil {
		return context
	}
	e.syncEditorToCanvas()

	var structure []string
	structure = append(structure, fmt.Sprintf("GITHUB PROJECT: %s/%s",
		e.githubProject.Owner, e.githubProject.Repo))
	structure = append(structure, "=================")
	for canvasNum, canvas := range e.canvases {
		if canvas.filename != "" {
			relPath, err := filepath.Rel(e.githubProject.LocalPath, canvas.filename)
			if err != nil {
				relPath = canvas.filename
			}

			structure = append(structure, fmt.Sprintf("Canvas %d: %s", canvasNum, relPath))
			content := strings.Join(canvas.lines, "\n")
			context.Files[relPath] = content
		}
	}

	context.ProjectStructure = strings.Join(structure, "\n")
	return context
}

// formatProjectContextForLLM форматирует контекст проекта для отправки в LLM
func (e *Editor) formatProjectContextForLLM(context *ProjectContext) string {
	var sb strings.Builder

	sb.WriteString("PROJECT CONTEXT ANALYSIS REQUEST\n")
	sb.WriteString("================================\n\n")

	sb.WriteString("INSTRUCTION:\n")
	sb.WriteString(context.Instruction)
	sb.WriteString("\n\n")

	sb.WriteString("PROJECT STRUCTURE:\n")
	sb.WriteString(context.ProjectStructure)
	sb.WriteString("\n\n")

	sb.WriteString("CURRENTLY ACTIVE FILE:\n")
	sb.WriteString(context.CurrentFile)
	sb.WriteString("\n\n")

	sb.WriteString("PROJECT FILES CONTENT:\n")
	sb.WriteString("======================\n\n")

	for filename, content := range context.Files {
		sb.WriteString(fmt.Sprintf("--- FILE: %s ---\n", filename))
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	sb.WriteString("END OF PROJECT CONTEXT\n")
	sb.WriteString("======================\n\n")
	sb.WriteString("Please analyze the entire project context and provide a comprehensive response based on the instruction above.")

	return sb.String()
}

// printVersion prints the editor version.
// printVersion выводит версию редактора.
func printVersion() {
	fmt.Println("Editor version", Version)
}

// detectSystemLanguage возвращает код языка системы: "ru", "en" или "de"
func detectSystemLanguage() string {
	var candidates = []string{
		os.Getenv("LANG"),
		os.Getenv("LC_ALL"),
		os.Getenv("LC_MESSAGES"),
		os.Getenv("LANGUAGE"),
	}
	for _, v := range candidates {
		if v == "" {
			continue
		}
		lv := strings.ToLower(v)
		if dot := strings.IndexByte(lv, '.'); dot != -1 {
			lv = lv[:dot]
		}
		if strings.Contains(lv, "ru") {
			return "ru"
		}
		if strings.Contains(lv, "en") {
			return "en"
		}
	}
	return "en"
}

// main is the entry point of the program.
// main является точкой входа в программу.
func main() {
	provider := os.Getenv("LLM_PROVIDER")
	model := os.Getenv("LLM_MODEL")
	path := ""
	keyFromArg := ""
	githubToken := ""

	flag.StringVar(&path, "path", "", "path to file or directory")
	flag.StringVar(&provider, "provider", provider, "LLMS provider")
	flag.StringVar(&model, "model", model, "LLMS model")
	flag.StringVar(&keyFromArg, "key", keyFromArg, "LLM API key для URL-based провайдеров")
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (short)")
	flag.Usage = printUsageExtended
	var streamMode bool
	var useClipboardData bool
	var inputFiles string
	flag.BoolVar(&streamMode, "stream", false, "Stream mode: read from stdin, write to stdout")
	flag.BoolVar(&streamMode, "s", false, "Stream mode (short)")
	flag.BoolVar(&useClipboardData, "data", false, "Use clipboard data as input in stream mode")
	flag.BoolVar(&useClipboardData, "d", false, "Use clipboard data as input in stream mode (short)")
	flag.StringVar(&inputFiles, "input", "", "Use file or directory content as input in stream mode")
	flag.StringVar(&inputFiles, "i", "", "Use file or directory content as input in stream mode (short)")

	flag.Usage = printUsageExtended
	flag.Parse()

	if streamMode {
		args := flag.Args()
		if len(args) >= 2 {
			provider = args[0]
			model = args[1]
			if len(args) >= 3 {
				keyFromArg = args[2]
			}
		}

		if provider == "" {
			provider = "ollama"
		}
		if model == "" {
			model = "gemma3:4b"
		}

		err := ProcessStreamingLLM(provider, model, keyFromArg, useClipboardData, inputFiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	args := flag.Args()
	switch {
	case len(args) >= 3:
		provider = args[0]
		model = args[1]
		path = args[2]
		if len(args) >= 4 {
			keyFromArg = args[3]
		}

	case len(args) == 2:
		provider = args[0]
		model = args[1]

		if strings.EqualFold(model, "help") {
			switch strings.ToLower(provider) {
			case "pollinations":
				nameModelPollinations()
				return
			case "llm7":
				nameModelLlm7()
				return
			case "openrouter":
				nameModelOpenRouter()
				return
			default:
				fmt.Println("Available models for known providers:")
				nameModelPollinations()
				nameModelLlm7()
				nameModelOpenRouter()
			}
			return
		}
	case len(args) == 1:
		path = args[0]
	default:
	}
	if isGitHubURL(path) {
		if len(args) >= 5 {
			githubToken = args[4]
		} else if len(args) == 4 && strings.HasPrefix(args[3], "ghp_") {
			githubToken = args[3]
		}

		editor, err := loadGitHubProject(path, provider, model, githubToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading GitHub project: %v\n", err)
			os.Exit(1)
		}
		editor.llmKey = keyFromArg

		if err := editor.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Editor startup error:", err)
		}
		return
	}
	if showVersion {
		printVersion()
		return
	}
	if path == "" && flag.NArg() > 0 && len(args) == 0 {
		path = flag.Arg(0)
	}
	if path != "" {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			editor := NewEditorWithProject(path, provider, model)
			editor.llmKey = keyFromArg
			if editor == nil {
				return
			}

			if err := editor.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "Editor startup error:", err)
			}
			return
		}
	}
	editor := NewEditor(path, provider, model)
	editor.llmKey = keyFromArg
	if editor == nil {
		return
	}

	if err := editor.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Editor startup error:", err)
	}
}

// filepathExt возвращает расширение файла в нижнем регистре
func filepathExtNew(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}
