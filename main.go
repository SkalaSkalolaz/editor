package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

// Version of the editor.
// Версия редактора.
const Version = "1.1.1"

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
	fmt.Println("Usage: editor [path] [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -provider string   LLM provider ")
	fmt.Println("  -model string      LLM model ")
	fmt.Println("  -h, --help        Показать эту справку и использование.")
	fmt.Println("  -v, --version     Показать версию программы.")
	fmt.Println()
	fmt.Println("Особенности:")
	fmt.Println("  - Терминальный текстовый редактор с поддержкой многострочного редактирования, курсорной навигации,")
	fmt.Println("    отмены/повтора (undo/redo), вырезания/копирования/вставки, поиска, перехода к строке,")
	fmt.Println("    и опциональной интеграции с LLM через cogitor.")
	fmt.Println("  - Интеграция LLM: вызывается внешний cogitor при настройке provider/model.")
	fmt.Println()
	fmt.Println("Горячие клавиши:")
	fmt.Println("  Ctrl-S  Сохранить файл")
	fmt.Println("  Ctrl-Q  Выход из редактора")
	fmt.Println("  Ctrl-F  Поиск текста")
	fmt.Println("  Ctrl-G  Перейти к строке (Goto)")
	fmt.Println("  Ctrl-U  Undo (Отменить)")
	fmt.Println("  Ctrl-Y  Redo (Вернуть отменённое)")
	fmt.Println("  Ctrl-K  Cut текущей строки")
	fmt.Println("  Ctrl-O  Открыть файл")
	fmt.Println("  Ctrl-N  Новый файл")
	fmt.Println("Навигация:")
	fmt.Println("  Стрелки: перемещение курсора, Home/End, PgUp/PgDn — навигация по тексту")
	fmt.Println()
	fmt.Println("Примеры:")
	fmt.Println("  editor -provider llm7 -model openai -path /path/to/file.txt")
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

// Editor represents the text editor state.
// Editor представляет состояние текстового редактора.
type Editor struct {
	screen        tcell.Screen
	filename      string
	lines         []string
	cx, cy        int
	offsetX       int
	offsetY       int
	dirty         bool
	clipboard     string
	undoStack     [][]string
	redoStack     [][]string
	prompt        *Prompt
	quit          bool
	width, height int
	llmProvider   string
	llmModel      string
	llmKey        string
	canvasWidth   int
	contentWidth  int
	contentHeight int
	language      Language
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
	e.contentHeight = 34
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
	e.width = e.contentWidth
	e.height = e.contentHeight
	e.canvasWidth = e.contentWidth
	if e.height <= 0 {
		e.height = 1
	}
	cursorRow, _, _ := e.cursorDisplayPosition()
	_ = cursorRow
}

// wrapLine wraps a line of text to fit the content width.
// wrapLine переносит строку текста в соответствии с шириной контента.
func (e *Editor) wrapLine(line string) []string {
	runes := []rune(line)
	if len(runes) == 0 {
		return []string{""}
	}
	var parts []string
	var currentWidth int
	var start int
	for i, r := range runes {
		rw := runewidth.RuneWidth(r)
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
				offsetInSegCells += runewidth.RuneWidth(segRunes[i])
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
		offsetInSegCells += runewidth.RuneWidth(r)
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

// render renders the editor to the screen.
// render отображает редактор на экране.
func (e *Editor) render() {
	e.screen.Clear()
	display := e.buildDisplayBuffer()
	total := len(display)
	topLine, bottomLine := e.statusBar()
	tRunes := []rune(topLine)
	for x := 0; x < e.contentWidth; x++ {
		var ch rune = ' '
		if x < len(tRunes) {
			ch = tRunes[x]
		}
		e.screen.SetContent(x, 0, ch, nil, tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite))
	}
	contentRows := e.contentHeight - 4
	if contentRows < 0 {
		contentRows = 0
	}
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
		xPos := 0
		tokenStartRuneIdx := 0
		for _, token := range tokens {
			tokenRunes := []rune(token.Text)
			tokenLenRunes := len(tokenRunes)
			tokenEndRuneIdx := tokenStartRuneIdx + tokenLenRunes
			style := token.Style
			if needHighlight {
				style = style.Background(tcell.ColorBlue)
			}
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
					if runeIdxInSeg >= 0 && runeIdxInSeg < len([]rune(row.text)) {
						segRunes := []rune(row.text)
						if runeIdxInSeg < len(segRunes) {
							r := segRunes[runeIdxInSeg]
							rw := 1
							if runeIdxInSeg < len(row.widths) {
								rw = row.widths[runeIdxInSeg]
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
			if needHighlight {
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
	bRunes := []rune(bottomLine)
	for i := 0; i < e.contentWidth; i++ {
		var ch rune = ' '
		if i < len(bRunes) {
			ch = bRunes[i]
		}
		e.screen.SetContent(i, e.contentHeight-1, ch, nil, tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite))
	}
	e.screen.Show()
}

// statusBar generates the top and bottom status bar text.
// statusBar генерирует текст верхней и нижней строки состояния.
func (e *Editor) statusBar() (string, string) {
	left := "EDITOR " + Version
	name := e.filename
	if name == "" {
		name = "[new file]"
	}
	langInfo := ""
	if e.language != LangUnknown {
		langInfo = " [" + string(e.language) + "]"
	}
	center := fmt.Sprintf("%s%s  Ln %d, Col %d", name, langInfo, e.cy+1, e.cx+1)
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
	bottom := "CTRL-L  Ctrl-S Save  Ctrl-O Open  Ctrl-N New  Ctrl-Q Quit  Ctrl-F Find  Ctrl-G GoTo  Ctrl-U OTME Ctrl-Y BEPH Ctrl-K"
	return top, bottom
}

// handleKey handles keyboard input.
// handleKey обрабатывает ввод с клавиатуры.
func (e *Editor) handleKey(ev *tcell.EventKey) {
	if e.prompt != nil {
		e.handlePromptInput(ev)
		return
	}
	switch ev.Key() {
	case tcell.KeyCtrlS:
		_ = e.save()
	case tcell.KeyCtrlQ:
		if e.dirty {
			e.promptShow("Записать перед выходом? (y/n)", func(input string) {
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
		e.promptShow("Поиск", func(input string) {
			e.findAndJump(input)
		})
	case tcell.KeyCtrlL:
		e.promptShow("Указание для LLM", func(input string) {
			e.llmQuery(input)
		})
	case tcell.KeyCtrlG:
		e.promptShow("Переход к строке", func(input string) {
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
		})
	case tcell.KeyCtrlU:
		e.undo()
	case tcell.KeyCtrlY:
		e.redo()
	case tcell.KeyCtrlK:
		e.cutLine()
	case tcell.KeyCtrlO:
		e.promptShow("Open file (path)", func(input string) {
			p := strings.TrimSpace(input)
			if p != "" {
				e.openFile(p)
			}
		})
	case tcell.KeyCtrlN:
		e.newFile()
	case tcell.KeyUp:
		if e.cy > 0 {
			e.cy--
			curRunes := []rune(e.lines[e.cy])
			if e.cx > len(curRunes) {
				e.cx = len(curRunes)
			}
			e.ensureVisible()
		}
	case tcell.KeyDown:
		if e.cy < len(e.lines)-1 {
			e.cy++
			curRunes := []rune(e.lines[e.cy])
			if e.cx > len(curRunes) {
				e.cx = len(curRunes)
			}
			e.ensureVisible()
		}
	case tcell.KeyLeft:
		if e.cx > 0 {
			e.cx--
		} else if e.cy > 0 {
			e.cy--
			prevRunes := []rune(e.lines[e.cy])
			e.cx = len(prevRunes)
			e.ensureVisible()
		}
	case tcell.KeyRight:
		lineRunes := []rune(e.lines[e.cy])
		lineLen := len(lineRunes)
		if e.cx < lineLen {
			e.cx++
		} else if e.cy < len(e.lines)-1 {
			e.cy++
			e.cx = 0
			e.ensureVisible()
		}
	case tcell.KeyHome:
		e.cx = 0
	case tcell.KeyEnd:
		lineRunes := []rune(e.lines[e.cy])
		e.cx = len(lineRunes)
	case tcell.KeyPgUp:
		step := e.height - 1
		e.offsetY -= step
		if e.offsetY < 0 {
			e.offsetY = 0
		}
		e.cy = e.offsetY
		if e.cy > len(e.lines)-1 {
			e.cy = len(e.lines) - 1
		}
	case tcell.KeyPgDn:
		step := e.height - 1
		e.offsetY += step
		if e.offsetY > len(e.lines)-1 {
			e.offsetY = len(e.lines) - 1
		}
		e.cy = e.offsetY
	case tcell.KeyEnter:
		e.newline()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		e.backspace()
	default:
		r := ev.Rune()
		if r != 0 && (ev.Modifiers()&tcell.ModAlt) == 0 {
			e.insertRune(r)
		}
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

// undo reverts the last change.
// undo отменяет последнее изменение.
func (e *Editor) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	current := make([]string, len(e.lines))
	copy(current, e.lines)
	e.redoStack = append(e.redoStack, current)
	last := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.lines = last
	e.cy = len(e.lines) - 1
	if e.cy < 0 {
		e.cy = 0
	}
	lineRunes := []rune(e.lines[e.cy])
	if e.cx > len(lineRunes) {
		e.cx = len(lineRunes)
	}
	e.dirty = true
	e.ensureVisible()
}

// redo reapplies the last undone change.
// redo повторно применяет последнее отмененное изменение.
func (e *Editor) redo() {
	if len(e.redoStack) == 0 {
		return
	}
	current := make([]string, len(e.lines))
	copy(current, e.lines)
	e.undoStack = append(e.undoStack, current)
	next := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.lines = next
	e.cy = len(e.lines) - 1
	if e.cy < 0 {
		e.cy = 0
	}
	lineRunes := []rune(e.lines[e.cy])
	if e.cx > len(lineRunes) {
		e.cx = len(lineRunes)
	}
	e.dirty = true
	e.ensureVisible()
}

// cutLine cuts the current line and copies it to the clipboard.
// cutLine вырезает текущую строку и копирует её в буфер обмена.
func (e *Editor) cutLine() {
	if e.cy >= 0 && e.cy < len(e.lines) {
		e.pushUndo()
		e.clipboard = e.lines[e.cy]
		if err := clipboard.WriteAll(e.clipboard); err != nil {
			e.statusMessage("Ошибка копирования БО: " + err.Error())
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

// llmQuery sends a query to the LLM.
// llmQuery отправляет запрос LLM.
func (e *Editor) llmQuery(instruction string) {
	if strings.TrimSpace(e.llmProvider) == "" {
		e.llmProvider = "pollinations"

	}
	if strings.TrimSpace(e.llmModel) == "" {
		e.llmModel = "openai"

	}
	if strings.TrimSpace(e.llmKey) == "" {
		e.llmKey = ""

	}
	payload := instruction
	if cb, err := clipboard.ReadAll(); err == nil {
		cb = strings.TrimSpace(cb)
		if cb != "" {
			payload = payload + "\nДанные из БО:\n" + cb
		}
	}
	visible := e.getVisibleText()
	if strings.TrimSpace(visible) != "" {
		payload = payload + "\nИмеющийся текст:\n" + visible
	}

	// If you do not have a program for interacting with the LLM called Cogitor,
	// I recommend using Tgpt, which can be installed from the MacOS terminal: brew install tgpt.
	// In this case, after installation, replace "cogitor" with "tgpt" in the code.
	// Если у Вас нет программы для взаимодействия с LLM под названием Cogitor, рекомендую
	// использовать Tgpt, которую можно установить из терминала MacOS: brew install tgpt.
	// В этом случае, после установки, замените в коде "cogitor" на "tgpt"

	cmd := exec.Command("cogitor", "-w", "-q", "--provider", e.llmProvider, "--model", e.llmModel, "--key", e.llmKey, payload)
	out, err := cmd.Output()
	if err != nil {
		e.statusMessage("LLM error: " + err.Error())
		return
	}
	resp := string(out)
	if strings.TrimSpace(resp) == "" {
		e.statusMessage("LLM вернула пустой ответ")
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
		e.statusMessage("Не удается открыть файл: " + err.Error())
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
	if showVersion {
		printVersion()
		return
	}
	if path == "" && flag.NArg() > 0 {
		path = flag.Arg(0)
	}
	editor := NewEditor(path, provider, model)
	if editor == nil {
		return
	}
	if err := editor.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка запуска редактора:", err)
	}
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
