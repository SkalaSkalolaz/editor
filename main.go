package main

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
const Version = "1.0.1"

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
	searchState      *SearchState
    lastSearchPattern string
	projectFiles      []string               
    currentFileIndex  int                    
    inProjectOverview bool                   
	fileSelection *FileSelection
	autoCompleteMode  bool
    autoCompleteState *AutoCompleteState
	bracketHighlightState *BracketHighlightState
}

// BracketHighlightState управляет состоянием подсветки парных скобок
type BracketHighlightState struct {
    active        bool
    bracketPair   *BracketPair
    startTime     time.Time
}

// AutoCompleteState хранит текущее состояние inline-autocomplete
type AutoCompleteState struct {
    active      bool           
    suggestion  string         
    fetched     bool         
    requestedAt time.Time   
	context     string
}

// ProjectContext представляет контекст всего проекта для отправки в LLM
type ProjectContext struct {
	ProjectStructure string            `json:"project_structure"`
	Files            map[string]string `json:"files"`
	CurrentFile      string            `json:"current_file"`
	Instruction      string            `json:"instruction"`
}

type SearchState struct {
    pattern          string
    matches          []MatchPosition
    currentMatch     int
    active           bool
    projectMatches   map[string]int
	matchedFiles     []string
}

type MatchPosition struct {
    line    int
    start   int
    end     int
    canvas  int
}

type FileSelection struct {
    selectedFiles map[string]bool
    lastAction    string    
    anchorIndex   int        
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
	case ".uml":
	    return LangPlantUML
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
		bracketHighlightState: &BracketHighlightState{
            active: false,
        },
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
	e.searchState = &SearchState{
 	   projectMatches: make(map[string]int),
	}
	return e
}

// readProjectFiles reads all supported files from a directory
// readProjectFiles читает все поддерживаемые файлы из директории
func readProjectFiles(dirPath string) (map[string]string, error) {
	files := make(map[string]string)

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
		".uml": true, ".png": true,
		".txt": true, ".log": true, ".csv": true, ".tsv": true, ".json": true, ".md": true, ".xml": true,
		".yaml": true, ".ini": true, ".cfg": true, ".env": true, ".nfo": true, ".css": true, ".bat": true,
		".sh": true, 
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
        filename:          dirPath,
        lines:             []string{""},
        dirty:             false,
        quit:              false,
        language:          LangUnknown,
        canvases:          make(map[int]*Canvas),
        currentCanvas:     1,
        projectFiles:      []string{},
        currentFileIndex:  -1,
        inProjectOverview: true,
		fileSelection: &FileSelection{
            selectedFiles: make(map[string]bool),
        },
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
        overviewLines, fileList := e.createProjectOverviewWithFiles(projectFiles)
        canvas.lines = overviewLines
        e.projectFiles = fileList
        e.currentFileIndex = 0
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


// createProjectOverviewWithFiles создает обзор проекта и возвращает список файлов
// с разделением на заполненные и пустые файлы
func (e *Editor) createProjectOverviewWithFiles(files map[string]string) ([]string, []string) {
    lines := []string{
        "PROJECT OVERVIEW",
        "================",
        "",
        "Files found: " + strconv.Itoa(len(files)),
        "",
    }

    var filenames []string
    for filename := range files {
        filenames = append(filenames, filename)
    }
    sort.Strings(filenames)

    var allDisplayFiles []string
    sourceFiles := []string{}
    configFiles := []string{}
    docFiles := []string{}
    emptyFiles := []string{}

    // Разделяем файлы на категории и проверяем на пустоту
    for _, filename := range filenames {
        content := files[filename]
        isEmpty := strings.TrimSpace(content) == ""
        
        ext := strings.ToLower(filepath.Ext(filename))
        lowerName := strings.ToLower(filename)

        switch {
        case isSourceFile(ext):
            if isEmpty {
                emptyFiles = append(emptyFiles, filename)
            } else {
                sourceFiles = append(sourceFiles, filename)
            }
        case isConfigFile(filename) || strings.Contains(lowerName, "config") ||
            strings.Contains(lowerName, "makefile") || strings.Contains(lowerName, "docker"):
            if isEmpty {
                emptyFiles = append(emptyFiles, filename)
            } else {
                configFiles = append(configFiles, filename)
            }
        case strings.Contains(lowerName, "readme") || strings.Contains(lowerName, "license") ||
            strings.Contains(lowerName, "copying") || strings.Contains(lowerName, "credits") || 
            strings.Contains(lowerName, "project"):
            if isEmpty {
                emptyFiles = append(emptyFiles, filename)
            } else {
                docFiles = append(docFiles, filename)
            }
        default:
            if isEmpty {
                emptyFiles = append(emptyFiles, filename)
            } else {
                configFiles = append(configFiles, filename)
            }
        }
    }

    // Собираем файлы в порядке отображения
    if len(sourceFiles) > 0 {
        lines = append(lines, "SOURCE FILES:")
        lines = append(lines, "-------------")
        for _, file := range sourceFiles {
            lines = append(lines, "  • "+file)
            allDisplayFiles = append(allDisplayFiles, file)
        }
        lines = append(lines, "")
    }
	if len(emptyFiles) > 0 {
        lines = append(lines, "EMPTY FILES:")
        lines = append(lines, "------------")
        for _, file := range emptyFiles {
            lines = append(lines, "  • "+file)
            allDisplayFiles = append(allDisplayFiles, file)
        }
        lines = append(lines, "")
    }
    if len(configFiles) > 0 {
        lines = append(lines, "CONFIGURATION FILES:")
        lines = append(lines, "---------------------")
        for _, file := range configFiles {
            lines = append(lines, "  • "+file)
            allDisplayFiles = append(allDisplayFiles, file)
        }
        lines = append(lines, "")
    }

    if len(docFiles) > 0 {
        lines = append(lines, "DOCUMENTATION:")
        lines = append(lines, "--------------")
        for _, file := range docFiles {
            lines = append(lines, "  • "+file)
            allDisplayFiles = append(allDisplayFiles, file)
        }
        lines = append(lines, "")
    }


    lines = append(lines, "Navigation: Use Ctrl+B to switch between files")
    lines = append(lines, "Press Arrow Keys to navigate files and Enter to open")
    lines = append(lines, "Press Tab on files allows you to select content that can be used for processing in LLM")
    lines = append(lines, "Press Ctrl+O to open file by name, Ctrl+N for new file")

    return lines, allDisplayFiles
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
	canvasNum := 2 

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
// buildProjectContext собирает контекст всего проекта с исправленной логикой выбора файлов
func (e *Editor) buildProjectContext(instruction string) *ProjectContext {
    context := &ProjectContext{
        Files:       make(map[string]string),
        Instruction: instruction,
        CurrentFile: e.filename,
    }

    e.syncEditorToCanvas()

    var filesToInclude []string
    
    if selectedFiles := e.getSelectedFiles(); len(selectedFiles) > 0 {
        filesToInclude = selectedFiles
        context.Instruction = instruction + " [Processing selected files: " + strings.Join(e.getSelectedDisplayNames(), ", ") + "]"
        
        e.statusMessage(fmt.Sprintf("Preparing %d selected files for LLM", len(filesToInclude)))
    } else {
        for _, canvas := range e.canvases {
            if canvas.filename != "" && len(canvas.lines) > 0 {
                filename := e.getRelativePath(canvas.filename)
                filesToInclude = append(filesToInclude, filename)
            }
        }
        context.Instruction = instruction + " [Processing all project files]"
    }

    filesAdded := 0
    for _, filename := range filesToInclude {
        content, err := e.getFileContent(filename)
        if err == nil && content != "" {
            displayName := e.getDisplayFileName(filename)
            context.Files[displayName] = content
            filesAdded++
        } else {
            e.statusMessage(fmt.Sprintf("Warning: Could not read file %s", filename))
        }
    }

    var structure []string
    structure = append(structure, "PROJECT STRUCTURE:")
    structure = append(structure, "=================")
    structure = append(structure, fmt.Sprintf("Files included: %d", filesAdded))
    
    for filename := range context.Files {
        structure = append(structure, "• "+filename)
    }
    
    context.ProjectStructure = strings.Join(structure, "\n")

    if filesAdded == 0 {
        e.statusMessage("Warning: No file content found to send to LLM")
    } else {
        e.statusMessage(fmt.Sprintf("Prepared %d files for LLM processing", filesAdded))
    }

    return context
}

// getFileContent возвращает содержимое файла из канвасов
func (e *Editor) getFileContent(filename string) (string, error) {
    for _, canvas := range e.canvases {
        if canvas.filename == filename && len(canvas.lines) > 0 {
            return strings.Join(canvas.lines, "\n"), nil
        }
    }
    
    baseName := filepath.Base(filename)
    for _, canvas := range e.canvases {
        if canvas.filename != "" && filepath.Base(canvas.filename) == baseName && len(canvas.lines) > 0 {
            return strings.Join(canvas.lines, "\n"), nil
        }
    }
    
    return "", fmt.Errorf("file not found in canvases: %s", filename)
}

// getDisplayFileName возвращает отображаемое имя файла
func (e *Editor) getDisplayFileName(filename string) string {
    if e.filename != "" {
        if relPath, err := filepath.Rel(filepath.Dir(e.filename), filename); err == nil {
            return relPath
        }
    }
    return filepath.Base(filename)
}

// getSelectedDisplayNames возвращает отображаемые имена выбранных файлов
func (e *Editor) getSelectedDisplayNames() []string {
    if e.fileSelection == nil {
        return nil
    }
    
    var names []string
    for filename := range e.fileSelection.selectedFiles {
        names = append(names, e.getDisplayFileName(filename))
    }
    sort.Strings(names)
    return names
}

// getRelativePath возвращает относительный путь файла
func (e *Editor) getRelativePath(fullPath string) string {
    if e.filename != "" {
        if relPath, err := filepath.Rel(filepath.Dir(e.filename), fullPath); err == nil {
            return relPath
        }
    } else if e.githubProject != nil && e.githubProject.LocalPath != "" {
        if relPath, err := filepath.Rel(e.githubProject.LocalPath, fullPath); err == nil {
            return relPath
        }
    }
    return filepath.Base(fullPath)
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

    if context.CurrentFile != "" {
        sb.WriteString("CURRENTLY ACTIVE FILE:\n")
        sb.WriteString(context.CurrentFile)
        sb.WriteString("\n\n")
    }

    sb.WriteString("PROJECT FILES CONTENT:\n")
    sb.WriteString("======================\n\n")

    fileCount := 0
    for filename, content := range context.Files {
        if strings.TrimSpace(content) == "" {
            continue
        }
        
        sb.WriteString(fmt.Sprintf("--- FILE: %s ---\n", filename))
        sb.WriteString(content)
        sb.WriteString("\n\n")
        fileCount++
    }

    if fileCount == 0 {
        sb.WriteString("No file content available.\n\n")
    }

    sb.WriteString("END OF PROJECT CONTEXT\n")
    sb.WriteString("======================\n\n")
    
    sb.WriteString("Please analyze the provided project context and provide a comprehensive response based on the instruction above.")
    sb.WriteString(" Focus on the actual code and file content provided.")

    return sb.String()
}

// validateLLMResponse проверяет ответ LLM перед отправкой в редактор
func (e *Editor) validateLLMResponse(response string) string {
    response = strings.TrimSpace(response)
    
    if response == "" {
        return "LLM returned an empty response. Please try again with a different prompt or check the LLM service."
    }
    
    if strings.Contains(strings.ToLower(response), "error") && 
       (strings.Contains(strings.ToLower(response), "failed to") || 
        strings.Contains(strings.ToLower(response), "unable to")) {
        e.statusMessage("Warning: LLM response may contain error indicators")
    }
    
    return response
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
	
	var autoCompleteMode bool
    flag.BoolVar(&autoCompleteMode, "auto", false, "Enable auto-complete mode")
    flag.BoolVar(&autoCompleteMode, "a", false, "Enable auto-complete mode (short)")

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
	if autoCompleteMode && !streamMode {
        if editor != nil {
            editor.autoCompleteMode = true
        }
    }

	if err := editor.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Editor startup error:", err)
	}
}

// filepathExt возвращает расширение файла в нижнем регистре
func filepathExtNew(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}
