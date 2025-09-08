package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

// Version of the editor.
// Версия редактора.
const Version = "1.6.4"

// Language represents the programming language of the file.
// Language представляет язык программирования файла.
type Language string

// Supported languages.
// Поддерживаемые языки.
const (
	LangC        Language = "c"
	LangCpp      Language = "cpp"
	LangAssembly Language = "assembly"
	LangFortran  Language = "fortran"
	LangGo       Language = "go"
	LangPython   Language = "python"
	LangRuby     Language = "ruby"
	LangKotlin   Language = "kotlin"
	LangSwift    Language = "swift"
	LangHTML     Language = "html"
	LangLisp     Language = "lisp"
	LangUnknown  Language = "unknown"
)

// Style definitions for syntax highlighting.
// Определения стилей для подсветки синтаксиса.
var (
	styleDefault  = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	styleKeyword  = tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	styleString   = tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)
	styleComment  = tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)
	styleType     = tcell.StyleDefault.Foreground(tcell.NewRGBColor(0, 255, 255)).Background(tcell.ColorBlack)
	styleNumber   = tcell.StyleDefault.Foreground(tcell.NewRGBColor(255, 0, 255)).Background(tcell.ColorBlack)
	styleFunction = tcell.StyleDefault.Foreground(tcell.ColorBlue).Background(tcell.ColorBlack)
	styleOperator = tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)
	stylePreproc  = tcell.StyleDefault.Foreground(tcell.ColorPurple).Background(tcell.ColorBlack)
)

// printVersion prints the editor version.
// printVersion выводит версию редактора.
func printVersion() {
	fmt.Println("Editor version", Version)
}

// printUsageExtended prints the extended help information.
// printUsageExtended выводит расширенную справку.
func printUsageExtended() {
	fmt.Println("Editor - расширенная справка")
	fmt.Println("Usage: editor  [provider] [model] [path]")
	fmt.Println()
	fmt.Println("provider {default: ollama}, model {default: gemma3:4b}")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help         Показать эту справку и использование.")
	fmt.Println("  -v, --version      Показать версию программы.")
	fmt.Println()
	fmt.Println("Особенности:")
	fmt.Println("  - Текстовый редактор с поддержкой многострочного редактирования, курсорной навигации,")
	fmt.Println("    отмены/повтора (undo/redo), вырезания/копирования/вставки, поиска, перехода к строке,")
	fmt.Println("    и опциональной интеграции с LLM.")
	fmt.Println("  - Интеграция LLM: вызывается при настройке provider/model.")
	fmt.Println()
	fmt.Println("Горячие клавиши:")
	fmt.Println("  Ctrl-L  Отправить указание для LLM")
	fmt.Println("  Ctrl-P  Генерирует код программы на основе описания\n          (в виде коментария)")
	fmt.Println("  Ctrl-R  Запускает код программы, при ошибке в коде - \n          рекомендации по их исправлению")
	fmt.Println("          Поддерживаемые языки: c, cpp, assembly, fortran, go, \n          python, ruby, kotlin, swift, html, lisp")
	fmt.Println("  Ctrl-S  Сохранить файл")
	fmt.Println("  Ctrl-O  Открыть файл")
	fmt.Println("  Ctrl-N  Новый файл")
	fmt.Println("  Ctrl-Q  Выход из редактора")
	fmt.Println("  Ctrl-F  Поиск текста")
	fmt.Println("  Ctrl-G  Перейти к строке")
	fmt.Println("  Ctrl-Z  Отменить")
	fmt.Println("  Ctrl-E  Вернуть отменённое")
	fmt.Println("  Ctrl-X  Убрать текущую строку")
	fmt.Println("  Ctrl-A  Выделить все")
	fmt.Println("  Ctrl-B  Выделить по строчно (от курсора)")
	fmt.Println("  Ctrl-C  Копировать в буфер обмена")
	fmt.Println("  Ctrl-V  Вставить буфер обмена")
	fmt.Println("  Ctrl-T  Терминал ОС (печать ответа в canvas)")
	fmt.Println("Навигация:")
	fmt.Println("  Стрелки: перемещение курсора, Home/End, PgUp/PgDn — навигация по тексту")
	fmt.Println()
	fmt.Println("Примеры:")
	fmt.Println("  editor pollinations openai /path/to/file.txt")
	fmt.Println("  editor file.txt")
}

// DisplayRow represents a line of text after wrapping.
// DisplayRow представляет строку текста после переноса.
type DisplayRow struct {
	lineIndex int
	segIndex  int
	text      string
	widths    []int
}

// Prompt represents a prompt for user input.
// Prompt представляет запрос пользовательского ввода.
type Prompt struct {
	Label    string
	Value    string
	Callback func(string)
}

// MultiLinePrompt represents a multi-line prompt for user input.
// MultiLinePrompt представляет многострочный запрос пользовательского ввода.
type MultiLinePrompt struct {
	Label    string
	Value    string
	Callback func(string)
}

// Editor represents the text editor state.
// Editor представляет состояние текстового редактора.
type Editor struct {
	screen                  tcell.Screen
	filename                string
	lines                   []string
	cx, cy                  int
	offsetX                 int
	offsetY                 int
	dirty                   bool
	clipboard               string
	undoStack               [][]string
	redoStack               [][]string
	prompt                  *Prompt
	multiLinePrompt         *MultiLinePrompt
	quit                    bool
	width, height           int
	llmProvider             string
	llmModel                string
	llmKey                  string
	canvasWidth             int
	contentWidth            int
	contentHeight           int
	language                Language
	selectAllBeforeLLM      bool
	ctrlAState              bool
	ctrlLState              bool
	ctrlPState              bool
	selectStartX            int
	selectStartY            int
	selecting               bool
	lineSelecting           bool
	terminalPrompt          *TerminalPrompt
	includeCtrlAContext     bool
	includeClipboardContext bool
}

type TerminalPrompt struct {
	Value    string
	Callback func(string)
}

// NewEditor creates a new Editor instance.
// NewEditor создает новый экземпляр Editor.
func NewEditor(path string, provider string, model string) *Editor {
	e := &Editor{
		filename: path,
		lines:    []string{""},
		dirty:    false,
		quit:     false,
		language: LangUnknown,
	}
	e.contentWidth = 115
	e.contentHeight = 35
	e.canvasWidth = e.contentWidth
	e.width = e.contentWidth
	e.height = e.contentHeight
	e.llmProvider = provider
	e.llmModel = model
	e.canvasWidth = 0
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
	return e
}

// detectLanguage detects the language based on the file extension.
// detectLanguage определяет язык на основе расширения файла.
func detectLanguage(filename string) Language {
	ext := strings.ToLower(filepathExt(filename))
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

func splitArgs(raw string) []string {
	var args []string
	var cur []rune
	inDouble := false
	inSingle := false
	escaped := false

	for _, r := range raw {
		switch {
		case escaped:
			cur = append(cur, r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == ' ' && !inDouble && !inSingle:
			if len(cur) > 0 {
				args = append(args, string(cur))
				cur = nil
			}
		default:
			cur = append(cur, r)
		}
	}
	if len(cur) > 0 {
		args = append(args, string(cur))
	}
	return args
}

// filepathExt returns the file extension.
// filepathExt возвращает расширение файла.
func filepathExt(filename string) string {
	for i := len(filename) - 1; i >= 0 && filename[i] != '/'; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}

// Run starts the editor's main loop.
// Run запускает основной цикл редактора.
func (e *Editor) Run() error {
	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := s.Init(); err != nil {
		return err
	}
	defer s.Fini()
	e.screen = s
	e.refreshSize()
	for !e.quit {
		e.render()
		ev := s.PollEvent()
		switch tev := ev.(type) {
		case *tcell.EventKey:
			e.handleKey(tev)
		case *tcell.EventResize:
			e.refreshSize()
		}
	}
	return nil
}

// refreshSize updates the editor's dimensions.
// refreshSize обновляет размеры редактора.
func (e *Editor) refreshSize() {
	// e.width = e.contentWidth
	// e.height = e.contentHeight
	// e.canvasWidth = e.contentWidth
	w, h := e.screen.Size()
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}
	if w > 115 {
		w = 115
	}
	e.contentWidth = w
	e.contentHeight = h
	e.width = e.contentWidth
	e.height = e.contentHeight
	e.canvasWidth = e.contentWidth
	if e.height <= 0 {
		e.height = 1
	}
	cursorRow, _, _ := e.cursorDisplayPosition()
	_ = cursorRow
}

func (e *Editor) wrapLine(line string) []string {
	runes := []rune(line)
	if len(runes) == 0 {
		return []string{""}
	}
	var parts []string
	var currentWidth int
	var start int
	tabWidth := 4

	for i, r := range runes {
		var rw int
		if r == '\t' {
			rw = tabWidth - (currentWidth % tabWidth)
		} else if unicode.IsSpace(r) {
			rw = 1
		} else {
			rw = utf8.RuneLen(r)
		}

		if currentWidth+rw > e.contentWidth && i > start {
			parts = append(parts, string(runes[start:i]))
			start = i
			currentWidth = rw
		} else {
			currentWidth += rw
		}
	}
	parts = append(parts, string(runes[start:]))
	return parts
}

// buildDisplayBuffer builds the display buffer from the editor's lines.
// buildDisplayBuffer строит буфер отображения из строк редактора.
func (e *Editor) buildDisplayBuffer() []DisplayRow {
	var buf []DisplayRow
	for li, line := range e.lines {
		parts := e.wrapLine(line)
		if len(parts) == 0 {
			parts = []string{""}
		}
		for si, seg := range parts {
			runes := []rune(seg)
			widths := make([]int, len(runes))
			for i, r := range runes {
				widths[i] = runewidth.RuneWidth(r)
			}
			buf = append(buf, DisplayRow{
				lineIndex: li,
				segIndex:  si,
				text:      seg,
				widths:    widths,
			})
		}
	}
	return buf
}

// cursorDisplayPosition calculates the display position of the cursor.
// cursorDisplayPosition вычисляет позицию отображения курсора.
func (e *Editor) cursorDisplayPosition() (int, int, int) {
	totalBefore := 0
	for i := 0; i < e.cy; i++ {
		totalBefore += len(e.wrapLine(e.lines[i]))
	}
	segs := e.wrapLine(e.lines[e.cy])
	lineRunes := []rune(e.lines[e.cy])
	if e.cx > len(lineRunes) {
		e.cx = len(lineRunes)
	}
	segmentStartRune := 0
	for segIndex, seg := range segs {
		segRunes := []rune(seg)
		segEndRune := segmentStartRune + len(segRunes)
		if e.cx >= segmentStartRune && e.cx <= segEndRune {
			offsetInSegRunes := e.cx - segmentStartRune
			offsetInSegCells := 0
			for i := 0; i < offsetInSegRunes; i++ {
				r := segRunes[i]
				if r == '\t' {
					offsetInSegCells += 4 - (offsetInSegCells % 4)
				} else {
					offsetInSegCells += runewidth.RuneWidth(r)
				}
			}
			displayRow := totalBefore + segIndex
			return displayRow, segIndex, offsetInSegCells
		}
		segmentStartRune = segEndRune
	}
	displayRow := totalBefore + len(segs) - 1
	lastSegRunes := []rune(segs[len(segs)-1])
	offsetInSegCells := 0
	for _, r := range lastSegRunes {
		if r == '\t' {
			offsetInSegCells += 4 - (offsetInSegCells % 4)
		} else {
			offsetInSegCells += runewidth.RuneWidth(r)
		}
	}
	return displayRow, len(segs) - 1, offsetInSegCells
}

// ensureVisible ensures the cursor is visible on the screen.
// ensureVisible обеспечивает видимость курсора на экране.
func (e *Editor) ensureVisible() {
	dispIdx, _, _ := e.cursorDisplayPosition()
	visibleRows := e.contentHeight - 4
	if visibleRows < 1 {
		visibleRows = 1
	}
	if dispIdx < e.offsetY {
		e.offsetY = dispIdx
	} else if dispIdx >= e.offsetY+visibleRows {
		e.offsetY = dispIdx - visibleRows + 1
	}
}

// highlightLine highlights a line of text based on the language.
// highlightLine подсвечивает строку текста в зависимости от языка.
func (e *Editor) highlightLine(line string, lineIndex int) []HighlightedToken {
	if e.language == LangUnknown {
		return []HighlightedToken{{Text: line, Style: styleDefault}}
	}
	switch e.language {
	case LangC:
		return highlightC(line)
	case LangCpp:
		return highlightCpp(line)
	case LangAssembly:
		return highlightAssembly(line)
	case LangFortran:
		return highlightFortran(line)
	case LangGo:
		return highlightGo(line)
	case LangPython:
		return highlightPython(line)
	case LangRuby:
		return highlightRuby(line)
	case LangKotlin:
		return highlightKotlin(line)
	case LangSwift:
		return highlightSwift(line)
	case LangHTML:
		return highlightHTML(line)
	case LangLisp:
		return highlightLisp(line)
	default:
		return []HighlightedToken{{Text: line, Style: styleDefault}}
	}
}

// HighlightedToken represents a token with its style.
// HighlightedToken представляет токен с его стилем.
type HighlightedToken struct {
	Text  string
	Style tcell.Style
}

// wrapText wraps text to fit a given width.
// wrapText переносит текст в соответствии с заданной шириной.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var lines []string
	runes := []rune(text)
	start := 0
	currentWidth := 0

	for i, r := range runes {
		rw := runewidth.RuneWidth(r)
		if currentWidth+rw > width && i > start {
			lines = append(lines, string(runes[start:i]))
			start = i
			currentWidth = rw
		} else {
			currentWidth += rw
		}

		if r == '\n' {
			lines = append(lines, string(runes[start:i+1]))
			start = i + 1
			currentWidth = 0
		}
	}

	if start < len(runes) {
		lines = append(lines, string(runes[start:]))
	} else if len(text) > 0 && text[len(text)-1] == '\n' {
		lines = append(lines, "")
	} else if len(text) == 0 {
		lines = append(lines, "")
	}

	return lines
}

// render renders the editor to the screen.
// render отображает редактор на экране.
func (e *Editor) render() {
	e.screen.Clear()
	display := e.buildDisplayBuffer()
	total := len(display)
	topLine, bottomLine1, bottomLine2 := e.statusBar()
	if e.terminalPrompt != nil {
		user, err := user.Current()
		username := "user"
		if err == nil && user.Username != "" {
			username = user.Username
		}

		hostname, err := os.Hostname()
		if err != nil || hostname == "" {
			hostname = "host"
		}

		cwd, err := os.Getwd()
		if err != nil {
			cwd = "?"
		} else {
			cwd, err = filepath.Abs(cwd)
			if err != nil {
				cwd = "?"
			}
		}
		promptPrefix := username + "@" + hostname + " " + cwd + " % "
		fullText := promptPrefix + e.terminalPrompt.Value
		wrapped := wrapText(fullText, e.contentWidth)
		n := len(wrapped)
		if n == 0 {
			bottomLine1 = ""
			bottomLine2 = ""
		} else if n == 1 {
			bottomLine1 = ""
			bottomLine2 = wrapped[0]
		} else {
			bottomLine2 = wrapped[n-2]
			bottomLine1 = wrapped[n-1]
		}
	}

	tRunes := []rune(topLine)
	for x := 0; x < e.contentWidth; x++ {
		var ch rune = ' '
		if x < len(tRunes) {
			ch = tRunes[x]
		}
		e.screen.SetContent(x, 0, ch, nil, tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
	}
	contentRows := e.contentHeight - 3
	if contentRows < 0 {
		contentRows = 0
	}

	var selStartLine, selStartCol, selEndLine, selEndCol int
	if e.selecting {
		selStartLine, selStartCol, selEndLine, selEndCol = e.getSelectionRange()
	}

	const tabWidth = 4

	for i := 0; i < contentRows; i++ {
		di := e.offsetY + i
		if di >= total {
			for x := 0; x < e.contentWidth; x++ {
				e.screen.SetContent(x, i+1, ' ', nil, styleDefault)
			}
			continue
		}
		row := display[di]
		originalLineText := e.lines[row.lineIndex]
		tokens := e.highlightLine(originalLineText, row.lineIndex)
		needHighlight := (row.lineIndex == e.cy)

		styleSelection := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue)
		styleSelectionCurrentLine := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorNavy)
		xPos := 0
		tokenStartRuneIdx := 0
		for _, token := range tokens {
			tokenRunes := []rune(token.Text)
			tokenLenRunes := len(tokenRunes)
			tokenEndRuneIdx := tokenStartRuneIdx + tokenLenRunes
			style := token.Style

			if e.selecting && row.lineIndex >= selStartLine && row.lineIndex <= selEndLine {

				tokenStartCol := tokenStartRuneIdx
				tokenEndCol := tokenEndRuneIdx

				lineSelStartCol := 0
				lineSelEndCol := len([]rune(e.lines[row.lineIndex]))

				if row.lineIndex == selStartLine {
					lineSelStartCol = selStartCol
				}
				if row.lineIndex == selEndLine {
					lineSelEndCol = selEndCol
				}

				if tokenStartCol < lineSelEndCol && tokenEndCol > lineSelStartCol {
					if needHighlight {
						style = styleSelectionCurrentLine
					} else {
						style = styleSelection
					}
					// Более сложная логика (например, частичное выделение токена)
					// требует рисования символа за символом с разными стилями.
					// Для простоты пока применяется стиль ко всему токену,
					// если он пересекается с выделением.
				}
			}

			if needHighlight && !e.selecting {
				style = style.Background(tcell.ColorBlue)
			}
			segRunes := []rune(row.text)
			for runeOffsetInToken := 0; runeOffsetInToken < tokenLenRunes; runeOffsetInToken++ {
				originalRuneIdx := tokenStartRuneIdx + runeOffsetInToken
				segStartRune := 0
				wrappedLines := e.wrapLine(originalLineText)
				for s := 0; s < row.segIndex; s++ {
					segStartRune += len([]rune(wrappedLines[s]))
				}
				segEndRune := segStartRune + len([]rune(row.text))
				if originalRuneIdx >= segStartRune && originalRuneIdx < segEndRune {
					runeIdxInSeg := originalRuneIdx - segStartRune
					if runeIdxInSeg >= 0 && runeIdxInSeg < len(segRunes) {
						r := segRunes[runeIdxInSeg]
						if r == '\t' {
							rw := tabWidth - (xPos % tabWidth)
							if xPos+rw > e.contentWidth {
								break
							}
							for cellOffset := 0; cellOffset < rw; cellOffset++ {
								e.screen.SetContent(xPos+cellOffset, i+1, ' ', nil, style)
							}
							xPos += rw
						} else {
							rw := 1
							if runeIdxInSeg < len(row.widths) {
								rw = row.widths[runeIdxInSeg]
							} else {
								rw = runewidth.RuneWidth(r)
							}
							if xPos+rw > e.contentWidth {
								break
							}
							for cellOffset := 0; cellOffset < rw; cellOffset++ {
								drawRune := r
								if cellOffset > 0 {
									drawRune = ' '
								}
								e.screen.SetContent(xPos+cellOffset, i+1, drawRune, nil, style)
							}
							xPos += rw
						}
					}
				}
				if xPos >= e.contentWidth {
					break
				}
			}
			tokenStartRuneIdx = tokenEndRuneIdx
			if xPos >= e.contentWidth {
				break
			}
		}
		for x := xPos; x < e.contentWidth; x++ {
			style := styleDefault
			if e.selecting && row.lineIndex >= selStartLine && row.lineIndex <= selEndLine {
				lineSelStartCol := 0

				if row.lineIndex == selStartLine {
					lineSelStartCol = selStartCol
				}

				if xPos >= lineSelStartCol && (row.lineIndex < selEndLine || xPos < selEndCol) {
					if needHighlight {
						style = styleSelectionCurrentLine
					} else {
						style = styleSelection
					}
				}
			}
			if needHighlight && !e.selecting {
				style = styleDefault.Background(tcell.ColorBlue)
			}
			e.screen.SetContent(x, i+1, ' ', nil, style)
		}
	}
	curDisplayRow, _, cursorInSeg := e.cursorDisplayPosition()
	cursorY := curDisplayRow - e.offsetY + 1
	if cursorY >= 1 && cursorY < e.contentHeight-3 {
		e.screen.ShowCursor(cursorInSeg, cursorY)
	} else {
		e.screen.HideCursor()
	}
	if e.prompt != nil && e.contentHeight >= 3 {
		promptLine := e.contentHeight - 3
		plain := e.prompt.Label + ": " + e.prompt.Value
		pr := []rune(plain)
		xPos := 0
		for i := 0; i < len(pr) && xPos < e.contentWidth; i++ {
			r := pr[i]
			rw := runewidth.RuneWidth(r)
			if xPos+rw > e.contentWidth {
				break
			}
			for cellOffset := 0; cellOffset < rw; cellOffset++ {
				drawRune := r
				if cellOffset > 0 {
					drawRune = ' '
				}
				e.screen.SetContent(xPos+cellOffset, promptLine, drawRune, nil, tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
			}
			xPos += rw
		}
		for x := xPos; x < e.contentWidth; x++ {
			e.screen.SetContent(x, promptLine, ' ', nil, tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
		}
	}

	if e.multiLinePrompt != nil && e.contentHeight >= 5 {
		promptText := e.multiLinePrompt.Label + ": " + e.multiLinePrompt.Value
		wrapWidth := e.contentWidth - 2
		if wrapWidth < 1 {
			wrapWidth = 1
		}
		wrappedLines := wrapText(promptText, wrapWidth)
		numLinesToShow := len(wrappedLines)
		if numLinesToShow > 20 {
			numLinesToShow = 20
		}
		startScreenRow := e.contentHeight - 2 - numLinesToShow
		if startScreenRow < 1 {
			startScreenRow = 1
			if len(wrappedLines) > (e.contentHeight - 2) {
				wrappedLines = wrappedLines[len(wrappedLines)-(e.contentHeight-2):]
			}
			numLinesToShow = len(wrappedLines)
			if numLinesToShow > e.contentHeight-2 {
				numLinesToShow = e.contentHeight - 2
			}
		}

		for i := 0; i < numLinesToShow; i++ {
			screenRow := startScreenRow + i
			if screenRow >= e.contentHeight-1 {
				break
			}
			for x := 0; x < e.contentWidth; x++ {
				e.screen.SetContent(x, screenRow, ' ', nil, tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
			}
		}
		for i := 0; i < numLinesToShow; i++ {
			screenRow := startScreenRow + i
			if screenRow >= e.contentHeight-1 {
				break
			}
			lineText := ""
			if i < len(wrappedLines) {
				lineText = wrappedLines[i]
			}
			lineRunes := []rune(lineText)
			xPos := 1
			for j := 0; j < len(lineRunes) && xPos < e.contentWidth-1; j++ {
				r := lineRunes[j]
				rw := runewidth.RuneWidth(r)
				if xPos+rw > e.contentWidth-1 {
					break
				}
				for cellOffset := 0; cellOffset < rw; cellOffset++ {
					drawRune := r
					if cellOffset > 0 {
						drawRune = ' '
					}
					e.screen.SetContent(xPos+cellOffset, screenRow, drawRune, nil, tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
				}
				xPos += rw
			}
		}
	}
	y1 := e.contentHeight - 1
	b1 := []rune(bottomLine1)
	x := 0
	for x < e.contentWidth {
		var ch rune = ' '
		if x < len(b1) {
			ch = b1[x]
		}
		if ch == '^' && x+1 < len(b1) {
			e.screen.SetContent(x, y1, ch, nil, tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
			next := b1[x+1]
			inv := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
			if x+1 < e.contentWidth {
				e.screen.SetContent(x+1, y1, next, nil, inv)
			}
			x += 2
			continue
		}
		style := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
		if x < len(b1) {
			e.screen.SetContent(x, y1, ch, nil, style)
		} else {
			e.screen.SetContent(x, y1, ' ', nil, style)
		}
		x++
	}

	if bottomLine2 != "" {
		y2 := e.contentHeight - 2
		b2 := []rune(bottomLine2)
		x = 0
		for x < e.contentWidth {
			var ch rune = ' '
			if x < len(b2) {
				ch = b2[x]
			}
			if ch == '^' && x+1 < len(b2) {
				e.screen.SetContent(x, y2, ch, nil, tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
				next := b2[x+1]
				inv := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
				if x+1 < e.contentWidth {
					e.screen.SetContent(x+1, y2, next, nil, inv)
				}
				x += 2
				continue
			}
			style := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
			if x < len(b2) {
				e.screen.SetContent(x, y2, ch, nil, style)
			} else {
				e.screen.SetContent(x, y2, ' ', nil, style)
			}
			x++
		}
	} else {
		for i := 0; i < e.contentWidth; i++ {
			e.screen.SetContent(i, e.contentHeight-2, ' ', nil, tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
		}
	}
	e.screen.Show()
}

func (e *Editor) startSelection() {
	if !e.selecting {
		e.selecting = true
		e.selectStartX = e.cx
		e.selectStartY = e.cy
	}
}

// startLineSelection начинает или расширяет выделение строк при Shift + стрелки.
// Если выделение уже активно, продолжает его в направлении движения курсора.
func (e *Editor) startLineSelection() {
	if !e.selecting {
		e.selecting = true
		e.lineSelecting = true
		e.selectStartX = e.cx
		e.selectStartY = e.cy
	} else if !e.lineSelecting {
		e.lineSelecting = true
		e.selectStartX = e.cx
		e.selectStartY = e.cy
	}
}

// endSelection завершает любое выделение.
func (e *Editor) endSelection() {
	e.selecting = false
	e.lineSelecting = false
}

// getSelectionRange возвращает диапазон выделенных строк или символов.
// При lineSelecting — возвращает начало и конец по строкам (с полными строками).
func (e *Editor) getSelectionRange() (int, int, int, int) {
	if !e.selecting {
		return 0, 0, 0, 0
	}

	if e.lineSelecting {
		startLine := e.selectStartY
		endLine := e.cy

		if startLine > endLine {
			startLine, endLine = endLine, startLine
		}

		startCol := 0
		endCol := 0
		if endLine < len(e.lines) {
			endCol = len([]rune(e.lines[endLine]))
		}

		return startLine, startCol, endLine, endCol
	}

	startLine := e.selectStartY
	endLine := e.cy
	startCol := e.selectStartX
	endCol := e.cx

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	return startLine, startCol, endLine, endCol
}

// getSelectedText возвращает текст из текущего выделения.
// При lineSelecting — возвращает полные строки.
func (e *Editor) getSelectedText() string {
	startLine, startCol, endLine, endCol := e.getSelectionRange()
	if startLine == 0 && startCol == 0 && endLine == 0 && endCol == 0 {
		return ""
	}

	var selectedLines []string

	if e.lineSelecting {
		if startLine < 0 {
			startLine = 0
		}
		if endLine >= len(e.lines) {
			endLine = len(e.lines) - 1
		}
		if startLine > endLine {
			startLine, endLine = endLine, startLine
		}
		for i := startLine; i <= endLine; i++ {
			if i < len(e.lines) {
				selectedLines = append(selectedLines, e.lines[i])
			}
		}
	} else {
		if startLine == endLine {
			if startLine < len(e.lines) {
				lineRunes := []rune(e.lines[startLine])
				if startCol < endCol && endCol <= len(lineRunes) {
					selectedLines = append(selectedLines, string(lineRunes[startCol:endCol]))
				}
			}
		} else {
			if startLine < len(e.lines) {
				firstLineRunes := []rune(e.lines[startLine])
				if startCol < len(firstLineRunes) {
					selectedLines = append(selectedLines, string(firstLineRunes[startCol:]))
				}
			}
			for i := startLine + 1; i < endLine; i++ {
				if i < len(e.lines) {
					selectedLines = append(selectedLines, e.lines[i])
				}
			}
			if endLine < len(e.lines) {
				lastLineRunes := []rune(e.lines[endLine])
				if endCol > 0 && endCol <= len(lastLineRunes) {
					selectedLines = append(selectedLines, string(lastLineRunes[:endCol]))
				}
			}
		}
	}

	return strings.Join(selectedLines, "\n")
}

// deleteSelection удаляет выделенный текст.
func (e *Editor) deleteSelection() {
	if !e.selecting {
		return
	}

	e.pushUndo()
	startLine, startCol, endLine, endCol := e.getSelectionRange()

	if e.lineSelecting {
		if startLine < 0 {
			startLine = 0
		}
		if endLine >= len(e.lines) {
			endLine = len(e.lines) - 1
		}
		if startLine > endLine {
			startLine, endLine = endLine, startLine
		}

		e.lines = append(e.lines[:startLine], e.lines[endLine+1:]...)
		e.cy = startLine
		if e.cy >= len(e.lines) {
			e.cy = len(e.lines) - 1
		}
		if e.cy < 0 {
			e.cy = 0
			e.cx = 0
		} else {
			e.cx = 0
		}
	} else {
		if startLine == endLine {
			lineRunes := []rune(e.lines[startLine])
			e.lines[startLine] = string(append(lineRunes[:startCol], lineRunes[endCol:]...))
			e.cx = startCol
		} else {
			firstLineRunes := []rune(e.lines[startLine])
			lastLineRunes := []rune(e.lines[endLine])
			merged := string(append(firstLineRunes[:startCol], lastLineRunes[endCol:]...))
			e.lines = append(e.lines[:startLine], append([]string{merged}, e.lines[endLine+1:]...)...)
			e.cy = startLine
			e.cx = startCol
		}
	}

	e.endSelection()
	e.dirty = true
	e.redoStack = nil
}

// statusBar generates the top and bottom status bar text.
// statusBar генерирует текст верхней и нижней строки состояния.
func (e *Editor) statusBar() (string, string, string) {
	left := "EDITOR " + Version
	name := e.filename
	if name == "" {
		name = "[new file]"
	}
	langInfo := ""
	if e.language != LangUnknown {
		langInfo = " [" + string(e.language) + "]"
	}
	totalLines := len(e.lines)
	indicator := ""
	if e.ctrlPState {
		indicator = " *P"
	}
	if e.ctrlLState {
		indicator = " *L"
	}
	if e.ctrlAState {
		indicator = " *A"
	}
	center := fmt.Sprintf("%s%s  Ln %d/%d, Col %d%s", name, langInfo, e.cy+1, totalLines, e.cx+1, indicator)
	lineRunes := make([]rune, e.contentWidth)
	for i := range lineRunes {
		lineRunes[i] = ' '
	}
	leftRunes := []rune(left)
	for i, r := range leftRunes {
		if i >= e.contentWidth {
			break
		}
		lineRunes[i] = r
	}
	leftLen := len(leftRunes)
	rem := e.contentWidth - leftLen
	if rem < 0 {
		rem = 0
	}
	centerRunes := []rune(center)
	centerPos := leftLen + (rem-len(centerRunes))/2
	if centerPos < leftLen {
		centerPos = leftLen
	}
	for i, r := range centerRunes {
		pos := centerPos + i
		if pos >= e.contentWidth {
			break
		}
		lineRunes[pos] = r
	}
	top := string(lineRunes)
	bottom2 := "^L Prompt    ^R Run code  ^N New file  ^O Open file ^S Save file ^Q Quit file ^F Find text ^G Go to line"
	bottom1 := "^P Generates ^C Copy      ^V Insert    ^B Select    ^A All       ^X Remove    ^Z Cancel    ^E Return    ^T Terminal"

	return top, bottom1, bottom2
}

// pasteFromClipboard reads text from the system clipboard and inserts it at the cursor position.
// pasteFromClipboard читает текст из системного буфера обмена и вставляет его в позицию курсора.
func (e *Editor) pasteFromClipboard() {
	text, err := clipboard.ReadAll()
	if err != nil {
		e.statusMessage("Insert error: " + err.Error())
		return
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	pasteLines := strings.Split(text, "\n")
	if len(pasteLines) == 0 {
		return
	}
	e.pushUndo()

	currentLine := e.lines[e.cy]
	lineRunes := []rune(currentLine)
	cursorX := e.cx
	if cursorX > len(lineRunes) {
		cursorX = len(lineRunes)
	}

	leftPart := string(lineRunes[:cursorX])
	rightPart := string(lineRunes[cursorX:])

	if len(pasteLines) == 1 {
		e.lines[e.cy] = leftPart + pasteLines[0] + rightPart
		e.cx = cursorX + len([]rune(pasteLines[0]))
	} else {
		e.lines[e.cy] = leftPart + pasteLines[0]
		insertIndex := e.cy + 1
		for i := 1; i < len(pasteLines)-1; i++ {
			e.lines = append(e.lines[:insertIndex], append([]string{pasteLines[i]}, e.lines[insertIndex:]...)...)
			insertIndex++
		}
		lastPastedLine := pasteLines[len(pasteLines)-1]
		e.lines = append(e.lines[:insertIndex], append([]string{lastPastedLine + rightPart}, e.lines[insertIndex:]...)...)
		e.cy = insertIndex
		e.cx = len([]rune(lastPastedLine))
	}

	e.dirty = true
	e.redoStack = nil
	e.ensureVisible()
}

func (e *Editor) handleRunCode() {
	code := strings.Join(e.lines, "\n")
	analysisQuery := "Analyze this code and return a JSON object with fields: language, flags, and args to run the" +
		"code (if the argument of the program is necessary for its correct launch, then come up with it yourself, based on" +
		"the requirements of the code, but do not specify the flags of the program name, for example: -o multiplication)." +
		" The JSON must be exactly in the format: {\"language\":\"<lang>\",\"flags\":\"<flags>\",\"args\":\"<args>\"}." +
		" Provide no extra text. Code:\n" + code

	analysis, err := e.llmQueryWithoutInsert(analysisQuery)
	if err != nil {
		e.statusMessage("Parsing error: " + err.Error())
		return
	}

	type llmResponse struct {
		Language string `json:"language"`
		Flags    string `json:"flags"`
		Args     string `json:"args"`
	}
	var resp llmResponse

	if err := json.Unmarshal([]byte(analysis), &resp); err != nil {
		s := strings.TrimSpace(analysis)
		start := strings.IndexByte(s, '{')
		end := strings.LastIndexByte(s, '}')
		if start != -1 && end != -1 && end >= start {
			if err2 := json.Unmarshal([]byte(s[start:end+1]), &resp); err2 != nil {
				e.statusMessage("Invalid JSON from LLM: " + err2.Error())
				return
			}
		} else {
			e.statusMessage("Invalid JSON from LLM: " + err.Error())
			return
		}
	}

	lang := strings.TrimSpace(resp.Language)
	flags := strings.TrimSpace(resp.Flags)
	args := strings.TrimSpace(resp.Args)

	if lang == "" {
		lang = "go"
	}

	stdout, stderr, runErr := e.runCodeViaRuncode(code, lang, flags, args)
	if runErr != nil {
		errorQuery := "Analyze the error and suggest corrections for the code: Error: " + runErr.Error() +
			"\nStderr: " + stderr + "\nCode:\n" + code
		fixes, err2 := e.llmQueryWithoutInsert(errorQuery)
		if err2 != nil {
			e.statusMessage("Error obtaining corrections:" + err2.Error())
			return
		}
		e.insertLLMResponse("\n\n// Recommendations for correction:\n" + fixes)
	} else {
		e.insertLLMResponse("\n\n// No errors were detected in the code. \n// Execution result.\n" + stdout)
	}
}

func (e *Editor) llmQueryWithoutInsert(instruction string) (string, error) {
	if strings.TrimSpace(e.llmProvider) == "" {
		e.llmProvider = "ollama"
	}
	if strings.TrimSpace(e.llmModel) == "" {
		e.llmModel = "gemma3:4b"
	}
	out, err := SendMessageToLLM(instruction, e.llmProvider, e.llmModel)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// This wrapper converts the args to a string and delegates to the internal runner.
func (e *Editor) runCodeViaRuncode(code string, lang string, flags string, runArgs string) (string, string, error) {
	args := strings.TrimSpace(runArgs)
	return e.runCodeInternally(code, lang, flags, args)
}

func (e *Editor) runCodeInternally(code string, lang string, flags string, runArgs string) (string, string, error) {
	l := strings.ToLower(strings.TrimSpace(lang))
	if l == "" {
		if strings.Contains(code, "package main") && strings.Contains(code, "func main") {
			l = "go"
		} else if strings.Contains(code, "#include <stdio.h>") {
			l = "c"
		} else {
			l = "go"
		}
	}

	switch l {
	case "cpp", "c++":
		return runCppInternal(code, flags, runArgs)
	case "c":
		return runCInternal(code, flags, runArgs)
	case "assembly", "asm":
		return runAssemblyInternal(code, flags, runArgs)
	case "fortran":
		return runFortranInternal(code, flags, runArgs)
	case "go":
		return runGoInternal(code, flags, runArgs)
	case "python", "py":
		return runPythonInternal(code, flags, runArgs)
	case "ruby", "rb":
		return runRubyInternal(code, flags, runArgs)
	case "kotlin", "kt":
		return runKotlinInternal(code, flags, runArgs)
	case "swift":
		return runSwiftInternal(code, flags, runArgs)
	case "html":
		return "", "", openHTMLInBrowser(code)
	case "lisp", "common lisp":
		return runLispInternal(code, flags, runArgs)
	default:
		return "", "", fmt.Errorf("unsupported language: %s", lang)
	}
}
func (e *Editor) showTerminalPrompt() {
	e.terminalPrompt = &TerminalPrompt{
		Value:    "",
		Callback: e.executeTerminalCommand,
	}
}

func (e *Editor) executeTerminalCommand(command string) {
	if strings.TrimSpace(command) == "" {
		return
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}
	cmdName := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	cmd := exec.Command(cmdName, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := ""
	if out.Len() > 0 {
		result += out.String()
	}
	if stderr.Len() > 0 {
		result += stderr.String()
	}
	if err != nil {
		result += fmt.Sprintf("\nError: %v", err)
	}

	if result != "" {
		e.insertLLMResponse(result)
	}
}

func (e *Editor) handleTerminalInput(ev *tcell.EventKey) {
	if e.terminalPrompt == nil {
		return
	}
	switch ev.Key() {
	case tcell.KeyEsc:
		e.terminalPrompt = nil
	case tcell.KeyEnter:
		val := e.terminalPrompt.Value
		cb := e.terminalPrompt.Callback
		if cb != nil {
			cb(val)
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.terminalPrompt.Value) > 0 {
			runes := []rune(e.terminalPrompt.Value)
			e.terminalPrompt.Value = string(runes[:len(runes)-1])
		}
	default:
		r := ev.Rune()
		if r != 0 {
			e.terminalPrompt.Value += string(r)
		}
	}
}

// handleKey handles keyboard input.
// handleKey обрабатывает ввод с клавиатуры.
func (e *Editor) handleKey(ev *tcell.EventKey) {
	if e.terminalPrompt != nil {
		e.handleTerminalInput(ev)
		return
	}
	if e.multiLinePrompt != nil {
		e.handleMultiLinePromptInput(ev)
		return
	}
	if e.prompt != nil {
		e.handlePromptInput(ev)
		return
	}
	shiftPressed := ev.Modifiers()&tcell.ModShift != 0

	switch ev.Key() {
	case tcell.KeyCtrlT:
		e.showTerminalPrompt()
	case tcell.KeyCtrlR:
		e.handleRunCode()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyCtrlB:
		if !e.selecting {
			e.selecting = true
			e.startLineSelection()
			currentCy := e.cy
			endLine := currentCy + 1
			if endLine >= len(e.lines) {
				endLine = len(e.lines) - 1
			}
			e.cy = endLine
			lastLine := ""
			if endLine >= 0 && endLine < len(e.lines) {
				lastLine = e.lines[endLine]
			}
			e.cx = len([]rune(lastLine))
			e.ensureVisible()
		}
		e.ctrlAState = false
		e.selectAllBeforeLLM = true
	case tcell.KeyCtrlA:
		if !e.selecting {
			e.selecting = true
			// e.lineSelecting = true
			e.selectStartX = 0
			e.selectStartY = 0
			// e.startLineSelection()
			e.cy = len(e.lines) - 1
			if e.cy < 0 {
				e.cy = 0
			}
			lastLine := ""
			if len(e.lines) > 0 {
				lastLine = e.lines[e.cy]
			}
			e.cx = len([]rune(lastLine))
			e.ensureVisible()
		}
		e.ctrlAState = true
		e.selectAllBeforeLLM = true
	case tcell.KeyCtrlS:
		_ = e.save()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyCtrlP:
		e.ctrlPState = true
		e.sendCommentToLLM()

	case tcell.KeyCtrlQ:
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
		if e.dirty {
			e.promptShow("Write down before leaving? (y/n)", func(input string) {
				switch strings.ToLower(strings.TrimSpace(input)) {
				case "y", "yes", "д", "да":
					if err := e.save(); err != nil {
						e.promptShow("Error saving: "+err.Error(), func(string) {})
					} else {
						e.quit = true
					}
				case "n", "no", "н", "нет":
					e.quit = true
				default:
				}
			})
		} else {
			e.quit = true
		}
	case tcell.KeyCtrlF:
		e.promptShow("Search", func(input string) {
			e.findAndJump(input)
		})
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyCtrlL:
		e.multiLinePromptShow("(To send the prompt, click Ctrl-L)  ", func(input string) {
			e.llmQuery(input)
		})
		e.ctrlLState = true
	case tcell.KeyCtrlG:
		e.promptShow("Go to line", func(input string) {
			n, err := strconv.Atoi(strings.TrimSpace(input))
			if err != nil || n <= 0 {
				return
			}
			line := n - 1
			if line < 0 {
				line = 0
			}
			if line >= len(e.lines) {
				line = len(e.lines) - 1
			}
			e.cy = line
			e.cx = 0
			e.ensureVisible()
			e.endSelection()
		})
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyCtrlZ:
		e.undo()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
		e.endSelection()
	case tcell.KeyCtrlE:
		e.redo()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
		e.endSelection()
	case tcell.KeyCtrlX:
		if e.selecting {
			selectedText := e.getSelectedText()
			if selectedText != "" {
				if err := clipboard.WriteAll(selectedText); err != nil {
					e.statusMessage("Copying error of clipboard: " + err.Error())
				} else {
					e.deleteSelection()
					e.statusMessage("Cut out: " + strconv.Itoa(strings.Count(selectedText, "\n")+1) + " lines")
				}
			}
		} else {
			e.cutLine()
		}

		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyCtrlO:
		e.promptShow("Open file (path)", func(input string) {
			p := strings.TrimSpace(input)
			if p != "" {
				e.openFile(p)
			}
		})
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
		e.endSelection()
	case tcell.KeyCtrlN:
		e.newFile()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
		e.endSelection()
	case tcell.KeyCtrlC:
		// Исправление: копирование выделенного текста в буфер редактора и системный буфер
		// чтобы контекст для LLM (через includeClipboardContext) мог быть передан.
		if e.selecting {
			selectedText := e.getSelectedText()
			if selectedText != "" {
				e.clipboard = selectedText
				if err := clipboard.WriteAll(selectedText); err != nil {
					e.statusMessage("Copy error clipboard: " + err.Error())
				} else {
					e.statusMessage("Copied " + strconv.Itoa(strings.Count(selectedText, "\n")+1) + " line(s) to clipboard")
				}
			}
		} else {
			// Если ничего не выделено, можно копировать текущую строку (опционально)
			curLine := e.lines[e.cy]
			if curLine != "" {
				e.clipboard = curLine
				if err := clipboard.WriteAll(curLine); err != nil {
					e.statusMessage("Copy error clipboard: " + err.Error())
				} else {
					e.statusMessage("Copied current line to clipboard")
				}
			}
		}
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyCtrlV:
		if e.selecting {
			e.deleteSelection()
		}
		e.pasteFromClipboard()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false

	case tcell.KeyUp:
		if e.cy > 0 {
			e.cy--
			curRunes := []rune(e.lines[e.cy])
			if e.cx > len(curRunes) {
				e.cx = len(curRunes)
			}
		}
		e.ensureVisible()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyDown:
		if e.cy < len(e.lines)-1 {
			e.cy++
			curRunes := []rune(e.lines[e.cy])
			if e.cx > len(curRunes) {
				e.cx = len(curRunes)
			}
		}
		e.ensureVisible()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyLeft:
		if e.cx > 0 {
			if shiftPressed {
				e.startLineSelection()
			} else if e.selecting {
				e.endSelection()
			}
			e.cx--
		} else if e.cy > 0 {
			if shiftPressed {
				e.startLineSelection()
			} else if e.selecting {
				e.endSelection()
			}
			e.cy--
			prevRunes := []rune(e.lines[e.cy])
			e.cx = len(prevRunes)
			e.ensureVisible()
		} else if e.selecting {
			e.endSelection()
		}
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyRight:
		lineRunes := []rune(e.lines[e.cy])
		lineLen := len(lineRunes)
		if e.cx < lineLen {
			if shiftPressed {
				e.startSelection()
			} else if e.selecting {
				e.endSelection()
			}
			e.cx++
		} else if e.cy < len(e.lines)-1 {
			if shiftPressed {
				e.startSelection()
			} else if e.selecting {
				e.endSelection()
			}
			e.cy++
			e.cx = 0
			e.ensureVisible()
		} else if e.selecting {
			e.endSelection()
		}
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyHome:
		if shiftPressed {
			e.startSelection()
		} else if e.selecting {
			e.endSelection()
		}
		e.cx = 0
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyEnd:
		if shiftPressed {
			e.startSelection()
		} else if e.selecting {
			e.endSelection()
		}
		lineRunes := []rune(e.lines[e.cy])
		e.cx = len(lineRunes)
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyPgUp:
		if shiftPressed {
			e.startSelection()
		} else if e.selecting {
			e.endSelection()
		}
		step := e.height - 1
		e.offsetY -= step
		if e.offsetY < 0 {
			e.offsetY = 0
		}
		e.cy = e.offsetY
		if e.cy > len(e.lines)-1 {
			e.cy = len(e.lines) - 1
		}
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyPgDn:
		if shiftPressed {
			e.startSelection()
		} else if e.selecting {
			e.endSelection()
		}
		step := e.height - 1
		e.offsetY += step
		if e.offsetY > len(e.lines)-1 {
			e.offsetY = len(e.lines) - 1
		}
		e.cy = e.offsetY
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyEnter:
		if e.terminalPrompt != nil {
			val := e.terminalPrompt.Value
			cb := e.terminalPrompt.Callback
			e.terminalPrompt = nil
			if cb != nil {
				cb(val)
			}
		}
		if e.selecting {
			e.deleteSelection()
		}
		e.newline()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if e.selecting {
			e.deleteSelection()
		} else {
			e.backspace()
		}
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
	case tcell.KeyEscape:
		e.endSelection()
		e.ctrlAState = false
		e.ctrlPState = false
		e.ctrlLState = false
		e.terminalPrompt = nil

	default:
		if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
			if len(e.terminalPrompt.Value) > 0 {
				runes := []rune(e.terminalPrompt.Value)
				e.terminalPrompt.Value = string(runes[:len(runes)-1])
			}
			return
		}
		r := ev.Rune()
		if r != 0 && (ev.Modifiers()&tcell.ModAlt) == 0 {
			if e.selecting {
				e.deleteSelection()
			}
			e.insertRune(r)
			e.ctrlAState = false
		}
		// Если нажата любая другая клавиша (не модификатор), сбрасываем выделение
		// Это нужно для клавиш как PageUp, PageDown и др., которые могут не обрабатываться выше
		// Но нужно быть осторожным, чтобы не сбросить выделение при нажатии Shift+другая_клавиша
		// Лучше явно обрабатывать сброс в нужных местах, как выше.
		// if !shiftPressed && ev.Key() != tcell.KeyShift && e.selecting {
		//     e.endSelection()
		// }
	}
	e.ensureVisible()
}

// insertRune inserts a rune at the current cursor position.
// insertRune вставляет символ в текущую позицию курсора.
func (e *Editor) insertRune(r rune) {
	e.pushUndo()
	lineRunes := []rune(e.lines[e.cy])
	if e.cx < 0 {
		e.cx = 0
	}
	lineRunes = append(lineRunes[:e.cx], append([]rune{r}, lineRunes[e.cx:]...)...)
	e.lines[e.cy] = string(lineRunes)
	e.cx++
	e.redoStack = nil
	e.dirty = true
}

// backspace deletes the character before the cursor.
// backspace удаляет символ перед курсором.
func (e *Editor) backspace() {
	if e.cx > 0 {
		e.pushUndo()
		lineRunes := []rune(e.lines[e.cy])
		lineRunes = append(lineRunes[:e.cx-1], lineRunes[e.cx:]...)
		e.lines[e.cy] = string(lineRunes)
		e.cx--
		e.dirty = true
	} else if e.cy > 0 {
		e.pushUndo()
		prev := e.lines[e.cy-1]
		cur := e.lines[e.cy]
		merged := prev + cur
		e.lines[e.cy-1] = merged
		e.lines = append(e.lines[:e.cy], e.lines[e.cy+1:]...)
		e.cy--
		e.cx = len([]rune(prev))
		e.dirty = true
	}
}

// newline inserts a new line at the current cursor position.
// newline вставляет новую строку в текущей позиции курсора.
func (e *Editor) newline() {
	e.pushUndo()
	lineRunes := []rune(e.lines[e.cy])
	left := string(lineRunes[:e.cx])
	right := string(lineRunes[e.cx:])
	e.lines[e.cy] = left
	e.lines = append(e.lines[:e.cy+1], append([]string{right}, e.lines[e.cy+1:]...)...)
	e.cy++
	e.cx = 0
	e.dirty = true
}

// save saves the file.
// save сохраняет файл.
func (e *Editor) save() error {
	if e.filename == "" {
		e.promptShow("Save as (path)", func(input string) {
			path := strings.TrimSpace(input)
			if path == "" {
				return
			}
			e.filename = path
			_ = e.persist()
		})
		return nil
	}
	return e.persist()
}

// persist writes the content to the file.
// persist записывает содержимое в файл.
func (e *Editor) persist() error {
	content := strings.Join(e.lines, "\n")
	err := os.WriteFile(e.filename, []byte(content), 0644)
	if err == nil {
		e.dirty = false
		e.undoStack = nil
		e.redoStack = nil
	}
	return err
}

// promptShow shows a prompt to the user.
// promptShow показывает пользователю запрос.
func (e *Editor) promptShow(label string, cb func(string)) {
	e.prompt = &Prompt{
		Label:    label,
		Value:    "",
		Callback: cb,
	}
	e.multiLinePrompt = nil
}

// multiLinePromptShow shows a multi-line prompt to the user.
// multiLinePromptShow показывает пользователю многострочный запрос.
func (e *Editor) multiLinePromptShow(label string, cb func(string)) {
	e.multiLinePrompt = &MultiLinePrompt{
		Label:    label,
		Value:    "",
		Callback: cb,
	}
	e.prompt = nil
}

// handlePromptInput handles input for the prompt.
// handlePromptInput обрабатывает ввод для запроса.
func (e *Editor) handlePromptInput(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEsc:
		e.prompt = nil
	case tcell.KeyEnter:
		if e.prompt != nil {
			val := e.prompt.Value
			cb := e.prompt.Callback
			e.prompt = nil
			if cb != nil {
				cb(val)
			}
		}
	default:
		if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
			if len(e.prompt.Value) > 0 {
				runes := []rune(e.prompt.Value)
				e.prompt.Value = string(runes[:len(runes)-1])
			}
			return
		}
		r := ev.Rune()
		if r != 0 {
			e.prompt.Value += string(r)
		}
	}
}

// handleMultiLinePromptInput handles input for the multi-line prompt.
// handleMultiLinePromptInput обрабатывает ввод для многострочного запроса.
func (e *Editor) handleMultiLinePromptInput(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEsc:
		e.multiLinePrompt = nil
	case tcell.KeyEnter:
		e.multiLinePrompt.Value += "\n"
	case tcell.KeyCtrlL:
		if e.multiLinePrompt != nil {
			val := e.multiLinePrompt.Value
			cb := e.multiLinePrompt.Callback
			e.multiLinePrompt = nil
			if cb != nil {
				cb(val)
			}
		}
		e.ctrlLState = false
	default:
		if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
			if len(e.multiLinePrompt.Value) > 0 {
				runes := []rune(e.multiLinePrompt.Value)
				e.multiLinePrompt.Value = string(runes[:len(runes)-1])
			}
			return
		}
		r := ev.Rune()
		if r != 0 {
			e.multiLinePrompt.Value += string(r)
		}
	}
}

// findAndJump finds the next occurrence of a string and jumps to it.
// findAndJump находит следующее вхождение строки и переходит к нему.
func (e *Editor) findAndJump(query string) {
	q := strings.TrimSpace(query)
	if q == "" {
		return
	}
	startY := e.cy
	startX := e.cx
	totalLines := len(e.lines)
	for i := 0; i < totalLines; i++ {
		line := e.lines[(startY+i)%totalLines]
		var searchFrom int
		if i == 0 {
			searchFrom = startX + 1
		} else {
			searchFrom = 0
		}
		lineRunes := []rune(line)
		if searchFrom > len(lineRunes) {
			searchFrom = len(lineRunes)
		}
		queryRunes := []rune(q)
		lineRunesFrom := lineRunes[searchFrom:]
		found := false
		pos := -1
		for j := 0; j <= len(lineRunesFrom)-len(queryRunes); j++ {
			match := true
			for k := 0; k < len(queryRunes); k++ {
				if lineRunesFrom[j+k] != queryRunes[k] {
					match = false
					break
				}
			}
			if match {
				found = true
				pos = j
				break
			}
		}
		if found {
			idx := searchFrom + pos
			e.cy = (startY + i) % totalLines
			e.cx = idx
			e.ensureVisible()
			return
		}
	}
	e.statusMessage("Not found: " + q)
}

// statusMessage displays a message on the status bar.
// statusMessage отображает сообщение в строке состояния.
func (e *Editor) statusMessage(msg string) {
	for i := 0; i < e.contentWidth; i++ {
		e.screen.SetContent(i, e.contentHeight-1, ' ', nil, tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite))
	}
	runes := []rune(msg)
	xPos := 0
	for i := 0; i < len(runes) && xPos < e.contentWidth; i++ {
		r := runes[i]
		rw := runewidth.RuneWidth(r)
		if xPos+rw > e.contentWidth {
			break
		}
		for cellOffset := 0; cellOffset < rw; cellOffset++ {
			drawRune := r
			if cellOffset > 0 {
				drawRune = ' '
			}
			e.screen.SetContent(xPos+cellOffset, e.contentHeight-1, drawRune, nil, tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite))
		}
		xPos += rw
	}
	e.screen.Show()
}

// pushUndo pushes the current state onto the undo stack.
// pushUndo помещает текущее состояние в стек отмены.
func (e *Editor) pushUndo() {
	clone := make([]string, len(e.lines))
	copy(clone, e.lines)
	e.undoStack = append(e.undoStack, clone)
	e.redoStack = nil
}

// scrollToLine прокручивает редактор так, чтобы указанная строка (lineIdx) была видна.
// Предпочтительно строка оказывается по центру экрана.
func (e *Editor) scrollToLine(lineIdx int) {
	if lineIdx < 0 {
		lineIdx = 0
	}
	if lineIdx >= len(e.lines) {
		lineIdx = len(e.lines) - 1
	}
	if lineIdx < 0 {
		return
	}
	visibleRows := e.contentHeight - 4
	if visibleRows < 1 {
		visibleRows = 1
	}
	totalDisplayRowsBeforeTarget := 0
	for i := 0; i < lineIdx; i++ {
		totalDisplayRowsBeforeTarget += len(e.wrapLine(e.lines[i]))
	}
	targetWrappedLinesCount := len(e.wrapLine(e.lines[lineIdx]))
	if targetWrappedLinesCount == 0 {
		targetWrappedLinesCount = 1
	}
	targetDisplayRow := totalDisplayRowsBeforeTarget
	newOffsetY := targetDisplayRow - visibleRows/2
	if newOffsetY < 0 {
		newOffsetY = 0
	} else {
		totalDisplayRows := 0
		for _, line := range e.lines {
			totalDisplayRows += len(e.wrapLine(line))
		}
		if totalDisplayRows == 0 {
			totalDisplayRows = 1
		}
		maxOffsetY := totalDisplayRows - visibleRows
		if maxOffsetY < 0 {
			maxOffsetY = 0
		}
		if newOffsetY > maxOffsetY {
			newOffsetY = maxOffsetY
		}
	}

	e.offsetY = newOffsetY
}

// centerViewOnCursor центрирует вид редактора на текущей позиции курсора (e.cy).
func (e *Editor) centerViewOnCursor() {
	e.scrollToLine(e.cy)
}

// undo reverts the last change.
// undo отменяет последнее изменение.
func (e *Editor) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	savedCx := e.cx
	savedCy := e.cy
	current := make([]string, len(e.lines))
	copy(current, e.lines)
	e.redoStack = append(e.redoStack, current)
	last := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.lines = last

	if savedCy >= len(e.lines) {
		e.cy = len(e.lines) - 1
		if e.cy < 0 {
			e.cy = 0
			e.cx = 0
		} else {
			lineRunes := []rune(e.lines[e.cy])
			e.cx = len(lineRunes)
		}
	} else {
		e.cy = savedCy
		lineRunes := []rune(e.lines[e.cy])
		if savedCx > len(lineRunes) {
			e.cx = len(lineRunes)
		} else {
			e.cx = savedCx
		}
	}
	e.dirty = true
	e.centerViewOnCursor()
}

// redo reapplies the last undone change.
// redo повторно применяет последнее отмененное изменение.
func (e *Editor) redo() {
	if len(e.redoStack) == 0 {
		return
	}
	savedCx := e.cx
	savedCy := e.cy
	current := make([]string, len(e.lines))
	copy(current, e.lines)
	e.undoStack = append(e.undoStack, current)
	next := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.lines = next
	if savedCy >= len(e.lines) {
		e.cy = len(e.lines) - 1
		if e.cy < 0 {
			e.cy = 0
			e.cx = 0
		} else {
			lineRunes := []rune(e.lines[e.cy])
			e.cx = len(lineRunes)
		}
	} else {
		e.cy = savedCy
		lineRunes := []rune(e.lines[e.cy])
		if savedCx > len(lineRunes) {
			e.cx = len(lineRunes)
		} else {
			e.cx = savedCx
		}
	}
	e.dirty = true
	e.centerViewOnCursor()

}

// cutLine cuts the current line and copies it to the clipboard.
// cutLine вырезает текущую строку и копирует её в буфер обмена.
func (e *Editor) cutLine() {
	if e.cy >= 0 && e.cy < len(e.lines) {
		e.pushUndo()
		e.clipboard = e.lines[e.cy]
		if err := clipboard.WriteAll(e.clipboard); err != nil {
			e.statusMessage("Copy error clipboard: " + err.Error())
		}
		e.lines = append(e.lines[:e.cy], e.lines[e.cy+1:]...)
		if e.cy >= len(e.lines) && len(e.lines) > 0 {
			e.cy = len(e.lines) - 1
		}
		if len(e.lines) == 0 {
			e.lines = []string{""}
			e.cy = 0
			e.cx = 0
		} else {
			lineRunes := []rune(e.lines[e.cy])
			if e.cx > len(lineRunes) {
				e.cx = len(lineRunes)
			}
		}
		e.dirty = true
		e.ensureVisible()
	}
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

// llmQuery sends a query to the LLM.
// llmQuery отправляет запрос LLM.
func (e *Editor) llmQuery(instruction string) {
	defer func() {
		e.selectAllBeforeLLM = false
		e.ctrlPState = false
		e.ctrlLState = false
	}()
	if strings.TrimSpace(e.llmProvider) == "" {
		e.llmProvider = "ollama"
	}
	if strings.TrimSpace(e.llmModel) == "" {
		e.llmModel = "gemma3:4b"
	}
	if strings.TrimSpace(e.llmKey) == "" {
		e.llmKey = ""
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

	out, err := SendMessageToLLM(payload, e.llmProvider, e.llmModel)
	if err != nil {
		e.statusMessage("LLM error: " + err.Error())
		return
	}

	resp := string(out)
	if strings.TrimSpace(resp) == "" {
		e.statusMessage("LLM returned an empty response")
		return
	}
	e.insertLLMResponse(resp)
}

// insertLLMResponse inserts the LLM response into the editor.
// insertLLMResponse вставляет ответ LLM в редактор.
func (e *Editor) insertLLMResponse(resp string) {
	resp = strings.ReplaceAll(resp, "\r\n", "\n")
	respLines := strings.Split(resp, "\n")
	if len(respLines) == 0 {
		return
	}
	lineRunes := []rune(e.lines[e.cy])
	if e.cx > len(lineRunes) {
		e.cx = len(lineRunes)
	}
	left := string(lineRunes[:e.cx])
	right := string(lineRunes[e.cx:])
	e.lines[e.cy] = left + respLines[0]
	insertIndex := e.cy + 1
	for i := 1; i < len(respLines); i++ {
		e.lines = append(e.lines[:insertIndex], append([]string{respLines[i]}, e.lines[insertIndex:]...)...)
		insertIndex++
	}
	lastLineIndex := e.cy
	if len(respLines) > 1 {
		lastLineIndex = e.cy + len(respLines) - 1
		e.lines[lastLineIndex] = e.lines[lastLineIndex] + right
	} else {
		e.lines[e.cy] = e.lines[e.cy] + right
	}
	e.cy = lastLineIndex
	e.cx = len([]rune(e.lines[e.cy]))
	e.dirty = true
	e.ensureVisible()
}

// openFile opens a file.
// openFile открывает файл.
func (e *Editor) openFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		e.statusMessage("Unable to open the file: " + err.Error())
		return
	}
	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	e.filename = path
	e.lines = strings.Split(content, "\n")
	e.language = detectLanguage(path)
	e.cx, e.cy = 0, 0
	e.offsetX, e.offsetY = 0, 0
	e.dirty = false
	e.undoStack = nil
	e.redoStack = nil
	e.ensureVisible()
}

// newFile creates a new file.
// newFile создает новый файл.
func (e *Editor) newFile() {
	e.filename = ""
	e.lines = []string{""}
	e.cx, e.cy = 0, 0
	e.offsetX, e.offsetY = 0, 0
	e.dirty = false
	e.undoStack = nil
	e.redoStack = nil
	e.language = LangUnknown
	e.ensureVisible()
}

// main is the entry point of the program.
// main является точкой входа в программу.
func main() {
	provider := os.Getenv("LLM_PROVIDER")
	model := os.Getenv("LLM_MODEL")
	path := ""
	flag.StringVar(&path, "path", "", "path to file")
	flag.StringVar(&provider, "provider", provider, "LLMS provider")
	flag.StringVar(&model, "model", model, "LLMS model")
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (short)")
	flag.Usage = printUsageExtended
	flag.Parse()

	args := flag.Args()
	switch {
	case len(args) >= 3:
		provider = args[0]
		model = args[1]
		path = args[2]
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
			default:

				fmt.Println("Available models for known providers:")
				nameModelPollinations()
				nameModelLlm7()
			}
			return
		}
	case len(args) == 1:
		path = args[0]
	default:
	}

	if showVersion {
		printVersion()
		return
	}
	if path == "" && flag.NArg() > 0 && len(args) == 0 {
		path = flag.Arg(0)
	}

	editor := NewEditor(path, provider, model)
	if editor == nil {
		return
	}
	if err := editor.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Editor startup error:", err)
	}
}

// New internal helpers: per-language implementations that compile/run locally
// and capture stdout/stderr. These do not depend on the external "run" binary.
func runCppInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.cpp")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	exe := tmp.Name() + ".out"

	compileArgs := []string{tmp.Name(), "-o", exe}
	if strings.TrimSpace(compileFlags) != "" {
		compileArgs = append(compileArgs, strings.Fields(compileFlags)...)
	}
	compile := exec.Command("g++", compileArgs...)
	var cbuf bytes.Buffer
	compile.Stdout = &cbuf
	compile.Stderr = &cbuf
	if err := compile.Run(); err != nil {
		return "", cbuf.String(), fmt.Errorf("compile error: %w", err)
	}

	run := exec.Command(exe)
	if strings.TrimSpace(runArgs) != "" {
		run.Args = append([]string{exe}, splitArgs(runArgs)...)
	} else {
		run.Args = []string{exe}
	}
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runCInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.c")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()

	exe := tmp.Name() + ".out"

	compileArgs := []string{tmp.Name(), "-o", exe}
	if strings.TrimSpace(compileFlags) != "" {
		compileArgs = append(compileArgs, strings.Fields(compileFlags)...)
	}
	compile := exec.Command("gcc", compileArgs...)
	var cbuf bytes.Buffer
	compile.Stdout = &cbuf
	compile.Stderr = &cbuf
	if err := compile.Run(); err != nil {
		return "", cbuf.String(), fmt.Errorf("compile error: %w", err)
	}

	run := exec.Command(exe)
	if strings.TrimSpace(runArgs) != "" {
		run.Args = append([]string{exe}, splitArgs(runArgs)...)
	} else {
		run.Args = []string{exe}
	}
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runAssemblyInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.asm")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()

	obj := tmp.Name() + ".o"
	exe := tmp.Name() + ".out"

	var nasmFormat string
	switch runtime.GOOS {
	case "linux", "darwin":
		nasmFormat = "elf64"
	case "windows":
		nasmFormat = "win64"
	default:
		nasmFormat = "elf64"
	}

	nasmArgs := []string{"-f", nasmFormat, tmp.Name(), "-o", obj}

	if strings.TrimSpace(compileFlags) != "" {
		nasmArgs = append(nasmArgs, splitArgs(compileFlags)...)
	}

	asm := exec.Command("nasm", nasmArgs...)
	asmOut, err := asm.CombinedOutput()
	if err != nil {
		return "", string(asmOut), fmt.Errorf("nasm error: %w", err)
	}
	defer os.Remove(obj)
	var ld *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		ld = exec.Command("ld", obj, "-o", exe)
	case "darwin":
		ld = exec.Command("cc", obj, "-o", exe)
	case "windows":
		ld = exec.Command("ld", obj, "-o", exe)
	default:
		ld = exec.Command("ld", obj, "-o", exe)
	}

	ldOut, err := ld.CombinedOutput()
	if err != nil {
		return "", string(ldOut), fmt.Errorf("link error: %w", err)
	}
	defer os.Remove(exe)
	run := exec.Command(exe)

	if strings.TrimSpace(runArgs) != "" {

		run.Args = append(run.Args, splitArgs(runArgs)...)
	}

	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr

	err = run.Run()

	return stdout.String(), stderr.String(), err
}

func runFortranInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.f90")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	exe := tmp.Name() + ".out"

	args := []string{tmp.Name(), "-o", exe}
	if compileFlags != "" {
		args = append(args, strings.Fields(compileFlags)...)
	}
	compile := exec.Command("gfortran", args...)
	compileOut, err := compile.CombinedOutput()
	if err != nil {
		return "", string(compileOut), fmt.Errorf("compile error: %w", err)
	}
	run := exec.Command(exe)
	if strings.TrimSpace(runArgs) != "" {
		run.Args = append([]string{exe}, splitArgs(runArgs)...)
	} else {
		run.Args = []string{exe}
	}
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runGoInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.go")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()

	args := []string{tmp.Name()}
	if strings.TrimSpace(runArgs) != "" {
		extra := splitArgs(runArgs)
		if len(extra) > 0 {
			args = append(args, extra...)
		}
	}
	cmd := exec.Command("go", append([]string{"run"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func runPythonInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.py")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	args := []string{tmp.Name()}
	if strings.TrimSpace(runArgs) != "" {
		args = append(args, splitArgs(runArgs)...)
	}
	cmd := exec.Command("python3", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func runRubyInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.rb")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	args := []string{tmp.Name()}
	if strings.TrimSpace(runArgs) != "" {
		args = append(args, splitArgs(runArgs)...)
	}
	cmd := exec.Command("ruby", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func runKotlinInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.kt")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	jar := tmp.Name() + ".jar"
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()

	compile := exec.Command("kotlinc", tmp.Name(), "-include-runtime", "-d", jar)
	if out, err := compile.CombinedOutput(); err != nil {
		return "", string(out), fmt.Errorf("kotlinc error: %w", err)
	}
	var run *exec.Cmd
	args := []string{"-jar", jar}
	if strings.TrimSpace(runArgs) != "" {
		args = append(args, splitArgs(runArgs)...)
	}
	run = exec.Command("java", args...)
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runSwiftInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.swift")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	exe := tmp.Name() + ".out"
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	compile := exec.Command("swiftc", tmp.Name(), "-o", exe)
	if out, err := compile.CombinedOutput(); err != nil {
		return "", string(out), fmt.Errorf("swiftc error: %w", err)
	}
	defer os.Remove(exe)
	run := exec.Command(exe)
	if strings.TrimSpace(runArgs) != "" {
		run.Args = append([]string{exe}, splitArgs(runArgs)...)
	} else {
		run.Args = []string{exe}
	}
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runLispInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.lisp")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	cmd := exec.Command("sbcl", "--script", tmp.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if strings.TrimSpace(runArgs) != "" {
		cmd.Args = append(cmd.Args, splitArgs(runArgs)...)
	}
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func openHTMLInBrowser(content string) error {
	tmp, err := ioutil.TempFile("", "runner_*.html")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open -a Safari ", tmp.Name())
	case "windows":
		cmd = exec.Command("start msedge ", tmp.Name())
	default:
		cmd = exec.Command("firefox ", tmp.Name())
	}
	return cmd.Start()
}

// getVisibleText gets the text that is currently visible on the screen.
// getVisibleText получает текст, который в данный момент отображается на экране.
func (e *Editor) getVisibleText() string {
	display := e.buildDisplayBuffer()
	visibleRows := e.contentHeight - 4
	if visibleRows < 0 {
		visibleRows = 0
	}
	start := e.offsetY
	end := start + visibleRows
	if end > len(display) {
		end = len(display)
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		b.WriteString(display[i].text)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// isAlpha checks if a byte is an alphabetic character or underscore.
// isAlpha проверяет, является ли байт алфавитным символом или подчеркиванием.
func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

// isDigit checks if a byte is a digit.
// isDigit проверяет, является ли байт цифрой.
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isOperator checks if a byte is an operator.
// isOperator проверяет, является ли байт оператором.
func isOperator(c byte) bool {
	return strings.Contains("+-*/%=<>!&|^~", string(c))
}

// highlightC highlights C code.
// highlightC подсвечивает код C.
func highlightC(line string) []HighlightedToken {
	var tokens []HighlightedToken
	keywords := map[string]bool{
		"auto": true, "break": true, "case": true, "char": true, "const": true, "continue": true,
		"default": true, "do": true, "double": true, "else": true, "enum": true, "extern": true,
		"float": true, "for": true, "goto": true, "if": true, "int": true, "long": true,
		"register": true, "return": true, "short": true, "signed": true, "sizeof": true,
		"static": true, "struct": true, "switch": true, "typedef": true, "union": true,
		"unsigned": true, "void": true, "volatile": true, "while": true,
	}
	types := map[string]bool{
		"int": true, "char": true, "float": true, "double": true, "void": true,
		"short": true, "long": true, "signed": true, "unsigned": true,
	}
	inString := false
	inChar := false
	inComment := false
	i := 0
	for i < len(line) {
		if i == 0 && line[i] == '#' {
			start := i
			for i < len(line) && line[i] != ' ' && line[i] != '\t' {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: stylePreproc})
			continue
		}
		if !inString && !inChar && i < len(line)-1 && line[i] == '/' && line[i+1] == '/' {
			tokens = append(tokens, HighlightedToken{Text: line[i:], Style: styleComment})
			break
		}
		if !inString && !inChar && i < len(line)-1 && line[i] == '/' && line[i+1] == '*' {
			start := i
			inComment = true
			i += 2
			for i < len(line)-1 {
				if line[i] == '*' && line[i+1] == '/' {
					i += 2
					inComment = false
					tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleComment})
					break
				}
				i++
			}
			if inComment {
				tokens = append(tokens, HighlightedToken{Text: line[start:], Style: styleComment})
				break
			}
			continue
		}
		if !inComment && !inChar && line[i] == '"' {
			start := i
			i++
			inString = true
			for i < len(line) {
				if line[i] == '\\' && i < len(line)-1 {
					i += 2
					continue
				}
				if line[i] == '"' {
					i++
					inString = false
					break
				}
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if !inComment && !inString && line[i] == '\'' {
			start := i
			i++
			inChar = true
			for i < len(line) {
				if line[i] == '\\' && i < len(line)-1 {
					i += 2
					continue
				}
				if line[i] == '\'' {
					i++
					inChar = false
					break
				}
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if !inComment && !inString && !inChar && isDigit(line[i]) {
			start := i
			for i < len(line) && (isDigit(line[i]) || line[i] == '.' || line[i] == 'x' || line[i] == 'X' ||
				(line[i] >= 'a' && line[i] <= 'f') || (line[i] >= 'A' && line[i] <= 'F')) {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleNumber})
			continue
		}
		if !inComment && !inString && !inChar && isAlpha(line[i]) {
			start := i
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) || line[i] == '_') {
				i++
			}
			word := line[start:i]
			if keywords[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleKeyword})
			} else if types[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleType})
			} else {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleDefault})
			}
			continue
		}
		if !inComment && !inString && !inChar && isOperator(line[i]) {
			start := i
			i++
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleOperator})
			continue
		}
		start := i
		i++
		tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
	}
	return tokens
}

// highlightCpp highlights C++ code.
// highlightCpp подсвечивает код C++.
func highlightCpp(line string) []HighlightedToken {
	tokens := highlightC(line)
	cppKeywords := map[string]bool{
		"class": true, "private": true, "protected": true, "public": true, "virtual": true,
		"override": true, "final": true, "template": true, "typename": true, "namespace": true,
		"using": true, "friend": true, "explicit": true, "inline": true, "operator": true,
		"new": true, "delete": true, "this": true, "nullptr": true, "constexpr": true,
		"decltype": true, "auto": true, "static_assert": true, "noexcept": true,
	}
	for i := range tokens {
		if tokens[i].Style == styleKeyword || tokens[i].Style == styleDefault {
			if cppKeywords[tokens[i].Text] {
				tokens[i].Style = styleKeyword
			}
		}
	}
	return tokens
}

// highlightAssembly highlights assembly code.
// highlightAssembly подсвечивает код ассемблера.
func highlightAssembly(line string) []HighlightedToken {
	var tokens []HighlightedToken
	keywords := map[string]bool{
		"mov": true, "add": true, "sub": true, "mul": true, "div": true, "cmp": true,
		"jmp": true, "je": true, "jne": true, "jg": true, "jl": true, "jge": true, "jle": true,
		"call": true, "ret": true, "push": true, "pop": true, "lea": true, "nop": true,
		"int": true, "cli": true, "sti": true, "hlt": true, "in": true, "out": true,
	}
	registers := map[string]bool{
		"eax": true, "ebx": true, "ecx": true, "edx": true, "esi": true, "edi": true,
		"ebp": true, "esp": true, "ax": true, "bx": true, "cx": true, "dx": true,
		"ah": true, "al": true, "bh": true, "bl": true, "ch": true, "cl": true, "dh": true, "dl": true,
		"r8": true, "r9": true, "r10": true, "r11": true, "r12": true, "r13": true, "r14": true, "r15": true,
		"rax": true, "rbx": true, "rcx": true, "rdx": true, "rsi": true, "rdi": true, "rbp": true, "rsp": true,
	}
	directives := map[string]bool{
		"section": true, "global": true, "extern": true, "db": true, "dw": true, "dd": true, "dq": true,
		"times": true, "equ": true,
	}
	i := 0
	for i < len(line) {
		if line[i] == ';' {
			tokens = append(tokens, HighlightedToken{Text: line[i:], Style: styleComment})
			break
		}
		if line[i] == '"' {
			start := i
			i++
			for i < len(line) && line[i] != '"' {
				if line[i] == '\\' && i < len(line)-1 {
					i++
				}
				i++
			}
			if i < len(line) {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if isDigit(line[i]) || (line[i] == '0' && i < len(line)-1 && (line[i+1] == 'x' || line[i+1] == 'b')) {
			start := i
			if line[i] == '0' && i < len(line)-1 {
				if line[i+1] == 'x' {
					i += 2
					for i < len(line) && ((line[i] >= '0' && line[i] <= '9') ||
						(line[i] >= 'a' && line[i] <= 'f') || (line[i] >= 'A' && line[i] <= 'F')) {
						i++
					}
				} else if line[i+1] == 'b' {
					i += 2
					for i < len(line) && (line[i] == '0' || line[i] == '1') {
						i++
					}
				} else {
					for i < len(line) && isDigit(line[i]) {
						i++
					}
				}
			} else {
				for i < len(line) && isDigit(line[i]) {
					i++
				}
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleNumber})
			continue
		}
		if isAlpha(line[i]) || line[i] == '_' || line[i] == '.' || line[i] == '%' {
			start := i
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) || line[i] == '_' || line[i] == '.' || line[i] == '%') {
				i++
			}
			word := line[start:i]
			if keywords[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleKeyword})
			} else if registers[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleType})
			} else if directives[word] || (len(word) > 0 && word[0] == '.') {
				tokens = append(tokens, HighlightedToken{Text: word, Style: stylePreproc})
			} else {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleDefault})
			}
			continue
		}
		start := i
		i++
		tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
	}
	return tokens
}

// highlightFortran highlights Fortran code.
// highlightFortran подсвечивает код Fortran.
func highlightFortran(line string) []HighlightedToken {
	var tokens []HighlightedToken
	keywords := map[string]bool{
		"PROGRAM": true, "END": true, "IMPLICIT": true, "NONE": true, "INTEGER": true,
		"REAL": true, "DOUBLE": true, "PRECISION": true, "COMPLEX": true, "CHARACTER": true,
		"LOGICAL": true, "PARAMETER": true, "DIMENSION": true, "ALLOCATABLE": true,
		"ALLOCATE": true, "DEALLOCATE": true, "POINTER": true, "TARGET": true,
		"IF": true, "THEN": true, "ELSE": true, "ELSEIF": true, "ENDIF": true,
		"DO": true, "WHILE": true, "ENDDO": true, "FORALL": true, "ENDFORALL": true,
		"SELECT": true, "CASE": true, "ENDSELECT": true, "WHERE": true, "ELSEWHERE": true,
		"ENDWHERE": true, "CONTINUE": true, "STOP": true, "PAUSE": true, "WRITE": true,
		"READ": true, "PRINT": true, "OPEN": true, "CLOSE": true, "INQUIRE": true,
		"BACKSPACE": true, "ENDFILE": true, "REWIND": true, "FORMAT": true,
	}
	i := 0
	if len(line) > 0 && (line[0] == '!' || line[0] == 'C' || line[0] == 'c') {
		return []HighlightedToken{{Text: line, Style: styleComment}}
	}
	for i < len(line) {
		if line[i] == '!' {
			tokens = append(tokens, HighlightedToken{Text: line[i:], Style: styleComment})
			break
		}
		if line[i] == '"' || line[i] == '\'' {
			quote := line[i]
			start := i
			i++
			for i < len(line) && line[i] != quote {
				if line[i] == '\\' && i < len(line)-1 {
					i++
				}
				i++
			}
			if i < len(line) {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if isDigit(line[i]) || (line[i] == '.' && i < len(line)-1 && isDigit(line[i+1])) {
			start := i
			hasDecimal := false
			for i < len(line) && (isDigit(line[i]) || line[i] == '.' ||
				(line[i] == 'E' || line[i] == 'e') ||
				((line[i] == '+' || line[i] == '-') && (i > 0 && (line[i-1] == 'E' || line[i-1] == 'e')))) {
				if line[i] == '.' {
					if hasDecimal {
						break
					}
					hasDecimal = true
				}
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleNumber})
			continue
		}
		if isAlpha(line[i]) {
			start := i
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) || line[i] == '_') {
				i++
			}
			word := strings.ToUpper(line[start:i])
			if keywords[word] {
				tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleKeyword})
			} else {
				tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
			}
			continue
		}
		start := i
		i++
		tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
	}
	return tokens
}

// highlightGo highlights Go code.
// highlightGo подсвечивает код Go.
func highlightGo(line string) []HighlightedToken {
	var tokens []HighlightedToken
	keywords := map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true, "interface": true,
		"map": true, "package": true, "range": true, "return": true, "select": true,
		"struct": true, "switch": true, "type": true, "var": true,
	}
	types := map[string]bool{
		"bool": true, "byte": true, "complex64": true, "complex128": true, "error": true,
		"float32": true, "float64": true, "int": true, "int8": true, "int16": true,
		"int32": true, "int64": true, "rune": true, "string": true, "uint": true,
		"uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	}
	inString := false
	inRawString := false
	inComment := false
	i := 0
	for i < len(line) {
		if !inString && !inRawString && i < len(line)-1 && line[i] == '/' && line[i+1] == '/' {
			tokens = append(tokens, HighlightedToken{Text: line[i:], Style: styleComment})
			break
		}
		if !inString && !inRawString && i < len(line)-1 && line[i] == '/' && line[i+1] == '*' {
			start := i
			i += 2
			for i < len(line)-1 {
				if line[i] == '*' && line[i+1] == '/' {
					i += 2
					tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleComment})
					break
				}
				i++
			}
			if i >= len(line)-1 {
				tokens = append(tokens, HighlightedToken{Text: line[start:], Style: styleComment})
				break
			}
			continue
		}
		if !inComment && !inRawString && line[i] == '"' {
			start := i
			i++
			inString = true
			for i < len(line) {
				if line[i] == '\\' && i < len(line)-1 {
					i += 2
					continue
				}
				if line[i] == '"' {
					i++
					inString = false
					break
				}
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if !inComment && !inString && line[i] == '`' {
			start := i
			i++
			inRawString = true
			for i < len(line) && line[i] != '`' {
				i++
			}
			if i < len(line) {
				i++
				inRawString = false
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if !inComment && !inString && !inRawString && line[i] == '\'' {
			start := i
			i++
			for i < len(line) {
				if line[i] == '\\' && i < len(line)-1 {
					i += 2
					continue
				}
				if line[i] == '\'' {
					i++
					break
				}
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if !inComment && !inString && !inRawString && isDigit(line[i]) {
			start := i
			for i < len(line) && (isDigit(line[i]) || line[i] == '.' || line[i] == 'x' || line[i] == 'X' ||
				(line[i] >= 'a' && line[i] <= 'f') || (line[i] >= 'A' && line[i] <= 'F') ||
				line[i] == 'e' || line[i] == 'E' || line[i] == '+' || line[i] == '-') {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleNumber})
			continue
		}
		if !inComment && !inString && !inRawString && isAlpha(line[i]) {
			start := i
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) || line[i] == '_') {
				i++
			}
			word := line[start:i]
			if keywords[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleKeyword})
			} else if types[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleType})
			} else {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleDefault})
			}
			continue
		}
		if !inComment && !inString && !inRawString && isOperator(line[i]) {
			start := i
			i++
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleOperator})
			continue
		}
		start := i
		i++
		tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
	}
	return tokens
}

// highlightPython highlights Python code.
// highlightPython подсвечивает код Python.
func highlightPython(line string) []HighlightedToken {
	var tokens []HighlightedToken
	keywords := map[string]bool{
		"and": true, "as": true, "assert": true, "break": true, "class": true, "continue": true,
		"def": true, "del": true, "elif": true, "else": true, "except": true, "exec": true,
		"finally": true, "for": true, "from": true, "global": true, "if": true, "import": true,
		"in": true, "is": true, "lambda": true, "not": true, "or": true, "pass": true,
		"print": true, "raise": true, "return": true, "try": true, "while": true, "with": true,
		"yield": true, "None": true, "True": true, "False": true,
	}
	inString := false
	inComment := false
	stringChar := byte(0)
	i := 0
	if len(line) > 0 && line[0] == '#' {
		return []HighlightedToken{{Text: line, Style: styleComment}}
	}
	for i < len(line) {
		if !inComment && !inString && (line[i] == '"' || line[i] == '\'') {
			start := i
			stringChar = line[i]
			i++
			if i < len(line)-1 && line[i] == stringChar && i+1 < len(line) && line[i+1] == stringChar {
				i += 2
				inString = true
				for i < len(line)-2 {
					if line[i] == stringChar && line[i+1] == stringChar && line[i+2] == stringChar {
						i += 3
						inString = false
						break
					}
					i++
				}
				if inString {
					i = len(line)
				}
				tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
				continue
			} else {
				inString = true
				for i < len(line) {
					if line[i] == '\\' && i < len(line)-1 {
						i += 2
						continue
					}
					if line[i] == stringChar {
						i++
						inString = false
						break
					}
					i++
				}
				tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
				continue
			}
		}
		if !inComment && !inString && isDigit(line[i]) {
			start := i
			for i < len(line) && (isDigit(line[i]) || line[i] == '.' ||
				(line[i] == 'e' || line[i] == 'E') ||
				((line[i] == '+' || line[i] == '-') && (i > 0 && (line[i-1] == 'e' || line[i-1] == 'E')))) {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleNumber})
			continue
		}
		if !inComment && !inString && isAlpha(line[i]) {
			start := i
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) || line[i] == '_') {
				i++
			}
			word := line[start:i]
			if keywords[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleKeyword})
			} else {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleDefault})
			}
			continue
		}
		start := i
		i++
		tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
	}
	return tokens
}

// highlightRuby highlights Ruby code.
// highlightRuby подсвечивает код Ruby.
func highlightRuby(line string) []HighlightedToken {
	var tokens []HighlightedToken
	keywords := map[string]bool{
		"alias": true, "and": true, "begin": true, "break": true, "case": true, "class": true,
		"def": true, "defined?": true, "do": true, "else": true, "elsif": true, "end": true,
		"ensure": true, "false": true, "for": true, "if": true, "in": true, "module": true,
		"next": true, "nil": true, "not": true, "or": true, "redo": true, "rescue": true,
		"retry": true, "return": true, "self": true, "super": true, "then": true, "true": true,
		"undef": true, "unless": true, "until": true, "when": true, "while": true, "yield": true,
	}
	inString := false
	inComment := false
	stringChar := byte(0)
	i := 0
	if len(line) > 0 && line[0] == '#' {
		return []HighlightedToken{{Text: line, Style: styleComment}}
	}
	for i < len(line) {
		if !inComment && !inString && (line[i] == '"' || line[i] == '\'' || line[i] == '`') {
			start := i
			stringChar = line[i]
			i++
			inString = true
			for i < len(line) {
				if line[i] == '\\' && i < len(line)-1 {
					i += 2
					continue
				}
				if line[i] == stringChar {
					i++
					inString = false
					break
				}
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if !inComment && !inString && line[i] == ':' && i < len(line)-1 && isAlpha(line[i+1]) {
			start := i
			i++
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) || line[i] == '_') {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleFunction})
			continue
		}
		if !inComment && !inString && isDigit(line[i]) {
			start := i
			for i < len(line) && (isDigit(line[i]) || line[i] == '.' ||
				(line[i] == 'e' || line[i] == 'E') ||
				((line[i] == '+' || line[i] == '-') && (i > 0 && (line[i-1] == 'e' || line[i-1] == 'E')))) {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleNumber})
			continue
		}
		if !inComment && !inString && isAlpha(line[i]) {
			start := i
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) || line[i] == '_') {
				i++
			}
			word := line[start:i]
			if keywords[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleKeyword})
			} else if word == "nil" || word == "true" || word == "false" {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleKeyword})
			} else {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleDefault})
			}
			continue
		}
		start := i
		i++
		tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
	}
	return tokens
}

// highlightKotlin highlights Kotlin code.
// highlightKotlin подсвечивает код Kotlin.
func highlightKotlin(line string) []HighlightedToken {
	var tokens []HighlightedToken
	keywords := map[string]bool{
		"package": true, "import": true, "class": true, "interface": true, "fun": true,
		"var": true, "val": true, "public": true, "private": true, "protected": true,
		"internal": true, "abstract": true, "final": true, "enum": true, "open": true,
		"attribute": true, "override": true, "inline": true, "vararg": true, "noinline": true,
		"crossinline": true, "reified": true, "tailrec": true, "operator": true,
		"infix": true, "external": true, "suspend": true, "const": true,
		"if": true, "else": true, "when": true, "for": true, "while": true, "do": true,
		"try": true, "catch": true, "finally": true, "throw": true, "return": true,
		"break": true, "continue": true, "object": true, "companion": true, "init": true,
		"this": true, "super": true, "typeof": true, "is": true, "as": true, "in": true,
		"out": true, "by": true, "get": true, "set": true,
	}
	types := map[string]bool{
		"Unit": true, "Int": true, "Long": true, "Byte": true, "Short": true,
		"Float": true, "Double": true, "Char": true, "Boolean": true, "String": true,
		"Array": true, "List": true, "Map": true, "Set": true, "Any": true, "Nothing": true,
	}
	inString := false
	inRawString := false
	inComment := false
	i := 0
	for i < len(line) {
		if !inString && !inRawString && i < len(line)-1 && line[i] == '/' && line[i+1] == '/' {
			tokens = append(tokens, HighlightedToken{Text: line[i:], Style: styleComment})
			break
		}
		if !inString && !inRawString && i < len(line)-1 && line[i] == '/' && line[i+1] == '*' {
			start := i
			i += 2
			for i < len(line)-1 {
				if line[i] == '*' && line[i+1] == '/' {
					i += 2
					tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleComment})
					break
				}
				i++
			}
			if i >= len(line)-1 {
				tokens = append(tokens, HighlightedToken{Text: line[start:], Style: styleComment})
				break
			}
			continue
		}
		if !inComment && !inRawString && line[i] == '"' {
			start := i
			i++
			if i < len(line)-1 && line[i] == '"' && i+1 < len(line) && line[i+1] == '"' {
				i += 2
				inRawString = true
				for i < len(line)-2 {
					if line[i] == '"' && line[i+1] == '"' && line[i+2] == '"' {
						i += 3
						inRawString = false
						break
					}
					i++
				}
				if inRawString {
					i = len(line)
				}
				tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
				continue
			} else {
				inString = true
				for i < len(line) {
					if line[i] == '\\' && i < len(line)-1 {
						i += 2
						continue
					}
					if line[i] == '"' {
						i++
						inString = false
						break
					}
					i++
				}
				tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
				continue
			}
		}
		if !inComment && !inString && !inRawString && line[i] == '\'' {
			start := i
			i++
			for i < len(line) {
				if line[i] == '\\' && i < len(line)-1 {
					i += 2
					continue
				}
				if line[i] == '\'' {
					i++
					break
				}
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if !inComment && !inString && !inRawString && isDigit(line[i]) {
			start := i
			for i < len(line) && (isDigit(line[i]) || line[i] == '.' || line[i] == 'x' || line[i] == 'X' ||
				(line[i] >= 'a' && line[i] <= 'f') || (line[i] >= 'A' && line[i] <= 'F') ||
				line[i] == 'e' || line[i] == 'E' || line[i] == '+' || line[i] == '-') {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleNumber})
			continue
		}
		if !inComment && !inString && !inRawString && isAlpha(line[i]) {
			start := i
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) || line[i] == '_') {
				i++
			}
			word := line[start:i]
			if keywords[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleKeyword})
			} else if types[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleType})
			} else {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleDefault})
			}
			continue
		}
		if !inComment && !inString && !inRawString && isOperator(line[i]) {
			start := i
			i++
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleOperator})
			continue
		}
		start := i
		i++
		tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
	}
	return tokens
}

// highlightSwift highlights Swift code.
// highlightSwift подсвечивает код Swift.
func highlightSwift(line string) []HighlightedToken {
	var tokens []HighlightedToken
	keywords := map[string]bool{
		"class": true, "deinit": true, "enum": true, "extension": true, "func": true,
		"import": true, "init": true, "let": true, "protocol": true, "static": true,
		"struct": true, "subscript": true, "typealias": true, "var": true, "break": true,
		"case": true, "continue": true, "default": true, "do": true, "else": true,
		"fallthrough": true, "if": true, "in": true, "for": true, "return": true,
		"switch": true, "where": true, "while": true, "as": true, "dynamicType": true,
		"is": true, "new": true, "super": true, "self": true, "Self": true, "Type": true,
		"__COLUMN__": true, "__FILE__": true, "__FUNCTION__": true, "__LINE__": true,
		"associativity": true, "didSet": true, "get": true, "infix": true, "inout": true,
		"left": true, "mutating": true, "none": true, "nonmutating": true, "operator": true,
		"override": true, "postfix": true, "precedence": true, "prefix": true, "right": true,
		"set": true, "unowned": true, "unowned(safe)": true, "unowned(unsafe)": true,
		"weak": true, "willSet": true,
	}
	types := map[string]bool{
		"Int": true, "Float": true, "Double": true, "Bool": true, "String": true,
		"Character": true, "Void": true, "Optional": true, "Array": true, "Dictionary": true,
		"Any": true, "AnyObject": true,
	}
	inString := false
	inComment := false
	i := 0
	for i < len(line) {
		if !inString && i < len(line)-1 && line[i] == '/' && line[i+1] == '/' {
			tokens = append(tokens, HighlightedToken{Text: line[i:], Style: styleComment})
			break
		}
		if !inString && i < len(line)-1 && line[i] == '/' && line[i+1] == '*' {
			start := i
			i += 2
			for i < len(line)-1 {
				if line[i] == '*' && line[i+1] == '/' {
					i += 2
					tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleComment})
					break
				}
				i++
			}
			if i >= len(line)-1 {
				tokens = append(tokens, HighlightedToken{Text: line[start:], Style: styleComment})
				break
			}
			continue
		}
		if !inComment && line[i] == '"' {
			start := i
			i++
			if i < len(line)-1 && line[i] == '"' && i+1 < len(line) && line[i+1] == '"' {
				i += 2
				for i < len(line)-2 {
					if line[i] == '"' && line[i+1] == '"' && line[i+2] == '"' {
						i += 3
						break
					}
					i++
				}
				tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
				continue
			} else {
				for i < len(line) {
					if line[i] == '\\' && i < len(line)-1 {
						i += 2
						continue
					}
					if line[i] == '"' {
						i++
						break
					}
					i++
				}
				tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
				continue
			}
		}
		if !inComment && !inString && line[i] == '\'' {
			start := i
			i++
			for i < len(line) {
				if line[i] == '\\' && i < len(line)-1 {
					i += 2
					continue
				}
				if line[i] == '\'' {
					i++
					break
				}
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if !inComment && !inString && isDigit(line[i]) {
			start := i
			for i < len(line) && (isDigit(line[i]) || line[i] == '.' || line[i] == 'x' || line[i] == 'X' ||
				(line[i] >= 'a' && line[i] <= 'f') || (line[i] >= 'A' && line[i] <= 'F') ||
				line[i] == 'e' || line[i] == 'E' || line[i] == '+' || line[i] == '-') {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleNumber})
			continue
		}
		if !inComment && !inString && isAlpha(line[i]) {
			start := i
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) || line[i] == '_') {
				i++
			}
			word := line[start:i]
			if keywords[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleKeyword})
			} else if types[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleType})
			} else {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleDefault})
			}
			continue
		}
		if !inComment && !inString && isOperator(line[i]) {
			start := i
			i++
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleOperator})
			continue
		}
		start := i
		i++
		tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
	}
	return tokens
}

// highlightHTML highlights HTML code.
// highlightHTML подсвечивает код HTML.
func highlightHTML(line string) []HighlightedToken {
	var tokens []HighlightedToken
	inTag := false
	inComment := false
	i := 0
	for i < len(line) {
		if !inTag && i < len(line)-3 && line[i] == '<' && line[i+1] == '!' && line[i+2] == '-' && line[i+3] == '-' {
			start := i
			i += 4
			for i < len(line)-2 {
				if line[i] == '-' && line[i+1] == '-' && line[i+2] == '>' {
					i += 3
					tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleComment})
					break
				}
				i++
			}
			if i >= len(line)-2 {
				tokens = append(tokens, HighlightedToken{Text: line[start:], Style: styleComment})
				break
			}
			continue
		}
		if !inComment && line[i] == '<' {
			start := i
			i++
			inTag = true
			for i < len(line) && line[i] != '>' {
				i++
			}
			if i < len(line) {
				i++
				inTag = false
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleKeyword})
			continue
		}
		if !inTag && !inComment && line[i] == '&' {
			start := i
			i++
			for i < len(line) && line[i] != ';' {
				i++
			}
			if i < len(line) {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleFunction})
			continue
		}
		start := i
		for i < len(line) && line[i] != '<' && line[i] != '&' {
			i++
		}
		if i > start {
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
		}
	}
	return tokens
}

// highlightLisp highlights Lisp code.
// highlightLisp подсвечивает код Lisp.

func highlightLisp(line string) []HighlightedToken {
	var tokens []HighlightedToken
	keywords := map[string]bool{
		"defun": true, "defvar": true, "defparameter": true, "defconstant": true,
		"let": true, "let*": true, "setf": true, "setq": true, "if": true,
		"cond": true, "case": true, "when": true, "unless": true, "loop": true,
		"do": true, "dolist": true, "dotimes": true, "lambda": true, "quote": true,
		"function": true, "progn": true, "prog1": true, "prog2": true, "block": true,
		"return": true, "return-from": true, "catch": true, "throw": true,
		"unwind-protect": true, "multiple-value-bind": true, "labels": true,
		"flet": true, "macrolet": true, "eval-when": true,
	}
	inString := false
	inComment := false
	i := 0
	if len(line) > 0 && line[0] == ';' {
		return []HighlightedToken{{Text: line, Style: styleComment}}
	}
	for i < len(line) {
		if !inComment && line[i] == '"' {
			start := i
			i++
			for i < len(line) {
				if line[i] == '\\' && i < len(line)-1 {
					i += 2
					continue
				}
				if line[i] == '"' {
					i++
					break
				}
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleString})
			continue
		}
		if !inString && line[i] == ';' {
			tokens = append(tokens, HighlightedToken{Text: line[i:], Style: styleComment})
			break
		}
		if !inComment && !inString && isDigit(line[i]) {
			start := i
			for i < len(line) && (isDigit(line[i]) || line[i] == '.' ||
				(line[i] == 'e' || line[i] == 'E') ||
				((line[i] == '+' || line[i] == '-') && (i > 0 && (line[i-1] == 'e' || line[i-1] == 'E')))) {
				i++
			}
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleNumber})
			continue
		}
		if !inComment && !inString && (isAlpha(line[i]) || strings.Contains("+-*/<>=", string(line[i]))) {
			start := i
			for i < len(line) && (isAlpha(line[i]) || isDigit(line[i]) ||
				strings.Contains("-+*/<>=", string(line[i]))) {
				i++
			}
			word := line[start:i]
			if keywords[word] {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleKeyword})
			} else {
				tokens = append(tokens, HighlightedToken{Text: word, Style: styleFunction})
			}
			continue
		}
		if !inComment && !inString && strings.Contains("()[]{}", string(line[i])) {
			start := i
			i++
			tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleOperator})
			continue
		}
		start := i
		i++
		tokens = append(tokens, HighlightedToken{Text: line[start:i], Style: styleDefault})
	}
	return tokens
}

func SendMessageToLLM(message, provider, model string) (string, error) {
	parsePollinationsResponse := func(body []byte) (string, error) {
		var m map[string]interface{}
		if err := json.Unmarshal(body, &m); err != nil {
			return "", fmt.Errorf("pollinations: некорректный json: %w", err)
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
		return "", errors.New("pollinations: не удалось распознать текст ответа")
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
		return "", errors.New("ollama: не удалось распознать текст ответа")
	}

	sendPollinations := func() (string, error) {
		apiKey := os.Getenv("POLLINATIONS_API_KEY")
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
			Model: "openai",
			Messages: []pollinationsMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: message},
			},
			Seed: 42,
		}

		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("pollinations: не удалось сформировать тело запроса: %w", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("pollinations: создание запроса не удалось: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("pollinations: сеть ошибка: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("pollinations: чтение ответа не удалось: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("pollinations: статус %d: %s", resp.StatusCode, string(respBody))
		}
		parsed, err := parsePollinationsResponse(respBody)
		if err != nil {
			return "", fmt.Errorf("pollinations: разбор ответа не удался: %w", err)
		}
		return parsed, nil
	}

	sendLLM7 := func() (string, error) {
		apiKey := os.Getenv("LLM7_API_KEY")
		if apiKey == "" {
			apiKey = "unused"
		}

		url := "https://api.llm7.io/v1/chat/completions"

		reqBody := map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "user", "content": message},
			},
			"temperature": 0.7,
			"top_p":       1.0,
		}
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("llm7: не удалось сформировать тело запроса: %w", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("llm7: создание запроса не удалось: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("llm7: сеть ошибка: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("llm7: чтение ответа не удалось: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("llm7: статус %d: %s", resp.StatusCode, string(respBody))
		}

		type llm7ChatMessage struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		type llm7Choice struct {
			Message llm7ChatMessage `json:"message"`
		}
		type llm7Response struct {
			Choices []llm7Choice `json:"choices"`
		}
		var r llm7Response
		if err := json.Unmarshal(respBody, &r); err == nil {
			if len(r.Choices) > 0 && r.Choices[0].Message.Content != "" {
				return r.Choices[0].Message.Content, nil
			}
		}
		var f map[string]interface{}
		if err := json.Unmarshal(respBody, &f); err == nil {
			if t, ok := f["text"].(string); ok && t != "" {
				return t, nil
			}
		}
		return "", fmt.Errorf("llm7: не удалось распознать ответ")
	}

	sendOllama := func() (string, error) {
		url := "http://localhost:11434/v1/chat/completions"

		reqBody := map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "user", "content": message},
			},
			"temperature": 0.7,
			"top_p":       1.0,
		}
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("ollama: не удалось сформировать тело запроса: %w", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("ollama: создание запроса не удалось: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("ollama: сеть ошибка: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("ollama: чтение ответа не удалось: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("ollama: статус %d: %s", resp.StatusCode, string(respBody))
		}

		parsed, err := parseOllamaResponse(respBody)
		if err != nil {
			return "", fmt.Errorf("ollama: разбор ответа не удался: %w", err)
		}
		return parsed, nil
	}

	switch provider {
	case "pollinations":
		return sendPollinations()
	case "llm7":
		return sendLLM7()
	case "ollama":
		return sendOllama()
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
	fmt.Printf("Модели Pollinations:\n")
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
		fmt.Printf("Модели Lmm7:\n")
		for _, m := range models {
			desc := "не указано"
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
		fmt.Printf("Модели Lmm7:\n")
		for _, m := range wrapper.Models {
			desc := "не указано"
			if len(m.Modalities.Input) > 0 {
				desc = strings.Join(m.Modalities.Input, ", ")
			}
			fmt.Printf(" %-40s %s\n", m.ID, desc)
		}
		return
	}

	fmt.Println("Не удалось разобрать ответ")
}
