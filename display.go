package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
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

type TerminalPrompt struct {
	Value    string
	Callback func(string)
}

// ExitManager управляет процессом выхода с проверкой всех канвасов.
type ExitManager struct {
	editor         *Editor
	canvasesToSave []int
	currentPrompt  int
}

func getLineCommentPrefix(lang Language) (string, bool) {
	switch lang {
	case LangGo, LangC, LangCpp, LangKotlin, LangSwift:
		return "// ", true
	case LangFortran:
		return "! ", true
	case LangPython, LangRuby:
		return "# ", true
	case LangLisp, LangAssembly:
		return "; ", true
	default:
		return "", false
	}
}

// refreshSize updates the editor's dimensions.
// refreshSize обновляет размеры редактора.
func (e *Editor) refreshSize() {
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

func (e *Editor) llmPromptWithPrevShow() {
	e.multiLinePrompt = &MultiLinePrompt{
		Label: "Enter your prompt. /Ctrl+L to send, Ctrl+P to send with project context/",
		Value: e.llmPrefill,
		Callback: func(input string) {
			e.llmPrefill = input
			e.llmQuery(input)
		},
	}
	e.prompt = nil
}

// cursorDisplayPosition calculates the display position of the cursor.
// cursorDisplayPosition вычисляет позицию отображения курсора.
func (e *Editor) cursorDisplayPosition() (int, int, int) {
	if e.cy < 0 {
		e.cy = 0
	}
	if e.cy >= len(e.lines) {
		if len(e.lines) > 0 {
			e.cy = len(e.lines) - 1
		} else {
			e.cy = 0
		}
	}
	if e.cx < 0 {
		e.cx = 0
	}

	if len(e.lines) == 0 {
		e.lines = []string{""}
		e.cy = 0
		e.cx = 0
	}

	if e.cy < len(e.lines) {
		lineRunes := []rune(e.lines[e.cy])
		if e.cx > len(lineRunes) {
			e.cx = len(lineRunes)
		}
	} else {
		e.cy = 0
		e.cx = 0
		if len(e.lines) == 0 {
			e.lines = []string{""}
		}
	}

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
	if len(e.lines) == 0 {
		e.lines = []string{""}
		e.cy = 0
		e.cx = 0
	}

	if e.cy < 0 {
		e.cy = 0
	}
	if e.cy >= len(e.lines) {
		e.cy = len(e.lines) - 1
	}
	if e.cx < 0 {
		e.cx = 0
	}
	if e.cy < len(e.lines) {
		lineRunes := []rune(e.lines[e.cy])
		if e.cx > len(lineRunes) {
			e.cx = len(lineRunes)
		}
	}

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

// insertTextAtCursor inserts given text at current cursor position, handling multi-line text.
func (e *Editor) insertTextAtCursor(text string) {
	e.pushUndo()
	parts := strings.Split(text, "\n")
	if len(parts) == 0 {
		return
	}
	if e.cy < 0 {
		e.cy = 0
	}
	for e.cy >= len(e.lines) {
		e.lines = append(e.lines, "")
	}
	lineRunes := []rune(e.lines[e.cy])
	if e.cx < 0 {
		e.cx = 0
	}
	if e.cx > len(lineRunes) {
		e.cx = len(lineRunes)
	}

	if len(parts) == 1 {
		left := string(lineRunes[:e.cx])
		right := string(lineRunes[e.cx:])
		e.lines[e.cy] = left + parts[0] + right
		e.cx += len([]rune(parts[0]))
	} else {
		origCy := e.cy
		left := string(lineRunes[:e.cx])
		right := string(lineRunes[e.cx:])
		e.lines[e.cy] = left + parts[0]
		for i := 1; i < len(parts)-1; i++ {
			insertLine := parts[i]
			e.lines = append(e.lines[:e.cy+1], append([]string{insertLine}, e.lines[e.cy+1:]...)...)
			e.cy++
		}
		last := parts[len(parts)-1] + right
		e.lines = append(e.lines[:e.cy+1], append([]string{last}, e.lines[e.cy+1:]...)...)
		e.cy = origCy + (len(parts) - 1)
		e.cx = len([]rune(parts[len(parts)-1]))
	}

	e.dirty = true
	e.ensureVisible()
}

// indentSelection добавляет отступ (например, 4 пробела) в начало выделенных строк.
// Если выделение построчное или символьное, применяется ко всем затронутым строкам.
func (e *Editor) indentSelection() {
	if !e.selecting {
		return
	}
	e.pushUndo()
	startLine, _, endLine, _ := e.getSelectionRange()
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(e.lines) {
		endLine = len(e.lines) - 1
	}
	if startLine > endLine {
		startLine, endLine = endLine, startLine
	}

	indentText := "    "
	for i := startLine; i <= endLine; i++ {
		e.lines[i] = indentText + e.lines[i]
	}

	e.dirty = true
}

// unindentSelection удаляет отступ (например, 4 пробела или 1 таб) из начала выделенных строк.
// Если выделение построчное или символьное, применяется ко всем затронутым строкам.
func (e *Editor) unindentSelection() {
	if !e.selecting {
		return
	}
	e.pushUndo()

	startLine, _, endLine, _ := e.getSelectionRange()
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(e.lines) {
		endLine = len(e.lines) - 1
	}
	if startLine > endLine {
		startLine, endLine = endLine, startLine
	}

	const tabSize = 4
	spaceIndent := strings.Repeat(" ", tabSize)

	for i := startLine; i <= endLine; i++ {
		line := e.lines[i]
		if strings.HasPrefix(line, spaceIndent) {
			e.lines[i] = line[tabSize:]
		} else if strings.HasPrefix(line, "\t") {
			e.lines[i] = line[1:]
		} else if len(line) > 0 {
			numSpaces := 0
			for numSpaces < len(line) && line[numSpaces] == ' ' && numSpaces < tabSize {
				numSpaces++
			}
			if numSpaces > 0 {
				e.lines[i] = line[numSpaces:]
			}
		}
	}

	e.dirty = true
}

// deleteWordAfterCursor удаляет слово, которое начинается либо после курсора,
// либо является словом, в котором находится курсор (если курсор внутри слова).
func (e *Editor) deleteWordAfterCursor() {
	if e.cy < 0 || e.cy >= len(e.lines) {
		return
	}
	e.pushUndo()

	line := e.lines[e.cy]
	runes := []rune(line)
	if e.cx < 0 {
		e.cx = 0
	}
	if e.cx >= len(runes) {
		if e.cy < len(e.lines)-1 {
			e.lines[e.cy] = line + e.lines[e.cy+1]
			e.lines = append(e.lines[:e.cy+1], e.lines[e.cy+2:]...)
			e.dirty = true
			e.ensureVisible()
		}
		return
	}

	idx := e.cx
	if idx > 0 && !unicode.IsSpace(runes[idx-1]) {
		start := idx - 1
		for start > 0 && !unicode.IsSpace(runes[start-1]) {
			start--
		}
		end := idx
		for end < len(runes) && !unicode.IsSpace(runes[end]) {
			end++
		}
		e.lines[e.cy] = string(append(runes[:start], runes[end:]...))
		e.cx = start
		e.dirty = true
		e.ensureVisible()
		return
	}
	i := idx
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	if i < len(runes) && !unicode.IsSpace(runes[i]) {
		start := i
		for i < len(runes) && !unicode.IsSpace(runes[i]) {
			i++
		}
		e.lines[e.cy] = string(append(runes[:start], runes[i:]...))
		e.cx = start
		e.dirty = true
		e.ensureVisible()
		return
	}

	if e.cy < len(e.lines)-1 {
		e.lines[e.cy] = line + e.lines[e.cy+1]
		e.lines = append(e.lines[:e.cy+1], e.lines[e.cy+2:]...)
		e.cx = len([]rune(line))
		e.dirty = true
		e.ensureVisible()
	}
}

// getUsageText возвращает текст расширенной справки в виде строки.
// getUsageText returns the extended help text as a string.
func (e *Editor) getUsageText() string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	outCh := make(chan string)
	go func() {
		var b bytes.Buffer
		io.Copy(&b, r)
		outCh <- b.String()
	}()

	printUsageMini()
	w.Close()
	os.Stdout = oldStdout
	usageText := <-outCh

	return usageText
}

// toggleCommentSelection комментирует или снимает комментарий у выделённых строк.
// Примечание: выполняется только если есть активное построчное выделение (lineSelecting)
func (e *Editor) toggleCommentSelection() {
	if !e.selecting || !e.lineSelecting {
		return
	}
	prefix, ok := getLineCommentPrefix(e.language)
	if !ok || prefix == "" {
		return
	}
	lo, _, hi, _ := e.getSelectionRange()
	if lo > hi {
		lo, hi = hi, lo
	}
	e.pushUndo()
	allCommented := true
	for i := lo; i <= hi; i++ {
		line := e.lines[i]
		lead := 0
		for lead < len(line) && (line[lead] == ' ' || line[lead] == '\t') {
			lead++
		}
		if lead+len(prefix) > len(line) || line[lead:lead+len(prefix)] != prefix {
			allCommented = false
			break
		}
	}

	if allCommented {
		for i := lo; i <= hi; i++ {
			l := e.lines[i]
			lead := 0
			for lead < len(l) && (l[lead] == ' ' || l[lead] == '\t') {
				lead++
			}
			if lead+len(prefix) <= len(l) && l[lead:lead+len(prefix)] == prefix {
				e.lines[i] = l[:lead] + l[lead+len(prefix):]
			}
		}
	} else {
		for i := lo; i <= hi; i++ {
			l := e.lines[i]
			lead := 0
			for lead < len(l) && (l[lead] == ' ' || l[lead] == '\t') {
				lead++
			}
			if lead+len(prefix) <= len(l) && l[lead:lead+len(prefix)] == prefix {
				continue
			}
			e.lines[i] = l[:lead] + prefix + l[lead:]
		}
	}
	e.dirty = true
	e.ensureVisible()
	e.endSelection()
}

func (e *Editor) toggleCommentLine() {
	lineIdx := e.cy
	if lineIdx < 0 || lineIdx >= len(e.lines) {
		return
	}
	line := e.lines[lineIdx]
	prefix, ok := getLineCommentPrefix(e.language)
	if !ok || prefix == "" {
		return
	}
	lead := 0
	for lead < len(line) && (line[lead] == ' ' || line[lead] == '\t') {
		lead++
	}
	if len(line) >= lead+len(prefix) && line[lead:lead+len(prefix)] == prefix {
		e.lines[lineIdx] = line[:lead] + line[lead+len(prefix):]
	} else {
		e.lines[lineIdx] = line[:lead] + prefix + line[lead:]
	}
	e.dirty = true
	e.ensureVisible()
}

// countSelectedTokens возвращает количество токенов
// в текущем выделении.
// По умолчанию, если нет выделения, возвращает 0.
func (e *Editor) countSelectedTokens() int {
	selected := e.getSelectedText()
	trimmed := strings.TrimSpace(selected)
	if trimmed == "" {
		return 0
	}
	tokens := strings.Fields(trimmed)
	return len(tokens)
}

// NewExitManager создает новый менеджер выхода.
func NewExitManager(editor *Editor) *ExitManager {
	return &ExitManager{
		editor:         editor,
		canvasesToSave: make([]int, 0),
		currentPrompt:  0,
	}
}

// openProjectFile opens a specific file from the project
// openProjectFile открывает конкретный файл из проекта
func (e *Editor) openProjectFile(filename string) {
	if e.filename != "" {
		if info, err := os.Stat(e.filename); err == nil && info.IsDir() {
			fullPath := filepath.Join(e.filename, filename)
			if _, err := os.Stat(fullPath); err == nil {
				e.openOrCreateCanvasForFile(fullPath)
				return
			}
		}
	}

	e.openFile(filename)
}

// openOrCreateCanvasForFile finds existing canvas or creates new one for file
// openOrCreateCanvasForFile находит существующий канвас или создает новый для файла
func (e *Editor) openOrCreateCanvasForFile(fullPath string) {
	for canvasNum, canvas := range e.canvases {
		if canvas.filename == fullPath {
			e.currentCanvas = canvasNum
			e.syncCanvasToEditor()
			e.statusMessage("Switched to canvas " + strconv.Itoa(canvasNum) + ": " + filepath.Base(fullPath))
			return
		}
	}

	if len(e.canvases) >= MaxCanvases {
		e.canvasWarningTime = time.Now()
		e.showError("The maximum number of canvases has been reached (" + strconv.Itoa(MaxCanvases) + ")")
		return
	}

	e.syncEditorToCanvas()
	newCanvasNum := 1
	for i := 1; i <= MaxCanvases; i++ {
		if _, exists := e.canvases[i]; !exists {
			newCanvasNum = i
			break
		}
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		e.showError("Unable to open the file: " + err.Error())
		return
	}

	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n")

	canvas := &Canvas{
		filename: fullPath,
		lines:    strings.Split(content, "\n"),
		cx:       0,
		cy:       0,
		offsetX:  0,
		offsetY:  0,
		dirty:    false,
		language: detectLanguage(fullPath),
	}

	e.canvases[newCanvasNum] = canvas
	e.currentCanvas = newCanvasNum
	e.syncCanvasToEditor()
	e.ensureVisible()

	e.statusMessage("Created canvas " + strconv.Itoa(newCanvasNum) + " for " + filepath.Base(fullPath))
}

// checkAllCanvases проверяет все канвасы на наличие несохраненных изменений.
func (em *ExitManager) checkAllCanvases() bool {
	em.canvasesToSave = make([]int, 0)
	em.editor.syncEditorToCanvas()
	for canvasNum, canvas := range em.editor.canvases {
		if canvas.hasUnsavedChanges() {
			em.canvasesToSave = append(em.canvasesToSave, canvasNum)
		}
	}

	return len(em.canvasesToSave) > 0
}

// promptForCanvasSave запрашивает сохранение для конкретного канваса.
func (em *ExitManager) promptForCanvasSave(canvasNum int) {
	canvas := em.editor.canvases[canvasNum]
	canvasName := canvas.getDisplayName()

	em.editor.promptShow(fmt.Sprintf("Save canvas %d: %s? (y/n/a/c)", canvasNum, canvasName),
		func(input string) {
			switch strings.ToLower(strings.TrimSpace(input)) {
			case "y", "yes", "д", "да":
				em.saveCanvas(canvasNum)
				em.processNextCanvas()
			case "n", "no", "н", "нет":
				em.processNextCanvas()
			case "a", "all", "в", "все":
				em.saveAllRemainingCanvases()
				em.finishExit()
			case "c", "cancel", "о", "отмена":
				em.cancelExit()
			default:
				em.promptForCanvasSave(canvasNum)
			}
		})
}

// saveCanvas сохраняет указанный канвас.
func (em *ExitManager) saveCanvas(canvasNum int) {
	oldCanvas := em.editor.currentCanvas
	em.editor.currentCanvas = canvasNum
	em.editor.syncCanvasToEditor()
	if em.editor.filename == "" {
		em.editor.promptShow("Save as (path)", func(input string) {
			path := strings.TrimSpace(input)
			if path != "" {
				em.editor.filename = path
				if err := em.editor.persist(); err == nil {
					em.editor.syncEditorToCanvas()
					em.editor.statusMessage(fmt.Sprintf("Canvas %d saved", canvasNum))
				}
			}
			em.editor.currentCanvas = oldCanvas
			em.editor.syncCanvasToEditor()
		})
	} else {
		if err := em.editor.persist(); err == nil {
			em.editor.syncEditorToCanvas()
			em.editor.statusMessage(fmt.Sprintf("Canvas %d saved", canvasNum))
		}
		em.editor.currentCanvas = oldCanvas
		em.editor.syncCanvasToEditor()
	}
}

// saveAllRemainingCanvases сохраняет все оставшиеся канвасы.
func (em *ExitManager) saveAllRemainingCanvases() {
	oldCanvas := em.editor.currentCanvas

	for i := em.currentPrompt; i < len(em.canvasesToSave); i++ {
		canvasNum := em.canvasesToSave[i]
		em.editor.currentCanvas = canvasNum
		em.editor.syncCanvasToEditor()

		if em.editor.filename != "" {
			if err := em.editor.persist(); err == nil {
				em.editor.syncEditorToCanvas()
				em.editor.statusMessage(fmt.Sprintf("Canvas %d saved", canvasNum))
			}
		}
	}
	em.editor.currentCanvas = oldCanvas
	em.editor.syncCanvasToEditor()
}

// processNextCanvas обрабатывает следующий канвас, требующий сохранения.
func (em *ExitManager) processNextCanvas() {
	em.currentPrompt++
	if em.currentPrompt >= len(em.canvasesToSave) {
		em.finishExit()
	} else {
		em.promptForCanvasSave(em.canvasesToSave[em.currentPrompt])
	}
}

// finishExit завершает процесс выхода.
func (em *ExitManager) finishExit() {
	em.editor.quit = true
}

// cancelExit отменяет процесс выхода.
func (em *ExitManager) cancelExit() {
	em.editor.prompt = nil
	em.editor.statusMessage("Exit cancelled")
}

// statusBar generates the top and bottom status bar text.
// statusBar генерирует текст верхней и нижней строки состояния.
func (e *Editor) statusBar() (string, string, string) {
	left := "EDITOR " + Version
	canvasInfo := fmt.Sprintf(" [Canvas %d/%d]", e.currentCanvas, len(e.canvases))
	left += canvasInfo

	if e.githubProject != nil {
		githubInfo := fmt.Sprintf(" [GitHub: %s/%s]", e.githubProject.Owner, e.githubProject.Repo)
		left += githubInfo
	}

	name := e.filename
	if name == "" {
		name = "[new file]"
	}
	langInfo := ""
	if e.language != LangUnknown {
		langInfo = " [" + string(e.language) + "]"
	}
	totalLines := len(e.lines)

	selectedTokens := 0
	if e.selecting {
		selectedTokens = e.countSelectedTokens()
	}
	center := fmt.Sprintf("%s%s  Ln %d/%d, Col %d%s", name, langInfo, e.cy+1, totalLines, e.cx+1)
	if selectedTokens > 0 {
		center = fmt.Sprintf("%s%s  Ln %d/%d, Col %d Toc %d", name, langInfo, e.cy+1, totalLines, e.cx+1,
			selectedTokens)
	} else {
		center = fmt.Sprintf("%s%s  Ln %d/%d, Col %d", name, langInfo, e.cy+1, totalLines, e.cx+1)
	}
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

	bottom2 := "^L Prompt    ^R Run code ^N New file ^O Open file ^S Save file ^Q Quit file ^F Find text ^G Go to line ^P Push Git"
	bottom1 := "^J HELP      ^C Copy     ^V Insert   ^B Next      ^A All       ^X Remove    ^Z Cancel    ^E Return     ^K Comment "

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
	case tcell.KeyCtrlV:
		if text, err := clipboard.ReadAll(); err == nil {
			text = strings.ReplaceAll(text, "\r\n", "\n")
			text = strings.ReplaceAll(text, "\r", "\n")
			e.terminalPrompt.Value += text
			e.render()
		} else {
			e.statusMessage("Paste error: " + err.Error())
		}

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

// handlePromptInput handles input for the prompt.
// handlePromptInput обрабатывает ввод для запроса.
func (e *Editor) handlePromptInput(ev *tcell.EventKey) {
	if ev.Key() == tcell.KeyCtrlV {
		if text, err := clipboard.ReadAll(); err == nil {
			text = strings.ReplaceAll(text, "\r\n", "\n")
			text = strings.ReplaceAll(text, "\r", "\n")
			if e.prompt != nil {
				e.prompt.Value = text
				e.render()
			} else if e.multiLinePrompt != nil {
				e.multiLinePrompt.Value += text
				e.render()
			}
		} else {
			e.statusMessage("Paste error: " + err.Error())
		}
		return
	}

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

// openFile открывает файл в текущем канвасе.
func (e *Editor) openFile(path string) {
	if e.filename != "" {
		if info, err := os.Stat(e.filename); err == nil && info.IsDir() {
			if !filepath.IsAbs(path) {
				path = filepath.Join(e.filename, path)
			}
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		e.showError("Unable to open the file: " + err.Error())
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
	e.syncEditorToCanvas()
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

func (e *Editor) promptShowWithInitial(label string, prefill string, cb func(string)) {
	e.prompt = &Prompt{
		Label:    label,
		Value:    prefill,
		Callback: cb,
	}
	e.multiLinePrompt = nil
}

// save сохраняет файл текущего канваса.
func (e *Editor) save() error {
	if e.filename == "" {
		e.promptShow("Save as (path)", func(input string) {
			path := strings.TrimSpace(input)
			if path == "" {
				return
			}
			e.filename = path
			_ = e.persist()
			e.syncEditorToCanvas()
		})
		return nil
	}
	err := e.persist()
	if err == nil {
		e.syncEditorToCanvas()
	}
	return err
}

// persist writes the content to the file with GitHub project support
func (e *Editor) persist() error {
	if e.githubProject != nil && e.filename != "" {
		absPath := e.filename
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(e.githubProject.LocalPath, absPath)
		}

		projectBase := e.githubProject.LocalPath
		if !strings.HasPrefix(absPath, projectBase) {
			return fmt.Errorf("file is outside project directory")
		}

		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		content := strings.Join(e.lines, "\n")
		err := os.WriteFile(absPath, []byte(content), 0644)
		if err != nil {
			e.showError("Unable to save the file: " + err.Error())
			return err
		}
	} else {
		content := strings.Join(e.lines, "\n")
		err := os.WriteFile(e.filename, []byte(content), 0644)
		if err != nil {
			e.showError("Unable to save the file: " + err.Error())
			return err
		}
	}

	e.dirty = false
	return nil
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

// resetEditorStates сбрасывает различные состояния редактора
func (e *Editor) resetEditorStates() {
	e.ctrlAState = false
	e.ctrlLState = false
	e.selectAllBeforeLLM = false
	e.endSelection()
}

// handleMultiLinePromptInput handles input for the multi-line prompt.
// handleMultiLinePromptInput обрабатывает ввод для многострочного запроса.
func (e *Editor) handleMultiLinePromptInput(ev *tcell.EventKey) {
	if ev.Key() == tcell.KeyCtrlV {
		if e.multiLinePrompt != nil {
			if text, err := clipboard.ReadAll(); err == nil {
				text = strings.ReplaceAll(text, "\r\n", "\n")
				text = strings.ReplaceAll(text, "\r", "\n")
				e.multiLinePrompt.Value = text
				e.render()
			}
			return
		}
		if e.prompt != nil && e.prompt.Label == "Search" {
			if text, err := clipboard.ReadAll(); err == nil {
				text = strings.ReplaceAll(text, "\r\n", "\n")
				text = strings.ReplaceAll(text, "\r", "\n")
				e.prompt.Value = text
				e.render()
			}
			return
		}
	}

	switch ev.Key() {
	case tcell.KeyEsc:
		if e.multiLinePrompt != nil && e.multiLinePrompt.Label == "Help (Press Esc to close)" {
			e.multiLinePrompt = nil
			e.render()
			return
		} else if e.multiLinePrompt != nil {
			e.multiLinePrompt = nil
			e.render()
			return
		}
	case tcell.KeyEnter:
		e.multiLinePrompt.Value += "\n"
	case tcell.KeyCtrlL:
		if e.multiLinePrompt != nil {
			if strings.TrimSpace(e.multiLinePrompt.Value) != "" {
				val := e.multiLinePrompt.Value
				cb := e.multiLinePrompt.Callback
				e.multiLinePrompt = nil
				if cb != nil {
					cb(val)
				}
			} else {
				e.ctrlLState = true
				e.sendCommentToLLM()
			}
		}
		e.ctrlLState = false
	case tcell.KeyCtrlP:
		if e.multiLinePrompt != nil && strings.TrimSpace(e.multiLinePrompt.Value) != "" {
			instruction := e.multiLinePrompt.Value
			e.multiLinePrompt = nil
			e.llmQueryWithProjectContext(instruction)
			e.resetEditorStates()
		} else if e.githubProject != nil {
			e.saveToGitHub()
		} else {
			e.showError("Not a GitHub project. Open a GitHub URL to enable this feature.")
		}
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

	if ev.Rune() == '\t' || ev.Key() == tcell.KeyTab {
		completion := e.findKeywordCompletion()
		if completion != "" {
			e.insertTextAtCursor(completion)
			return
		}
		completion = e.findIdentifierCompletion()
		if completion != "" {
			e.insertTextAtCursor(completion)
			return
		}
		if e.isAtEndOfIncompleteStatement() {
			if terminator, exists := languageTerminators[e.language]; exists && terminator != "" {
				e.insertTextAtCursor(terminator)
				return
			}
		}

		// Check if we should add closing brackets
		// This would be for cases like pressing Tab after typing "if ("
		// We'll add this feature later if needed

		e.insertRune('\t')
		return
	}
	if ev.Key() == tcell.KeyCtrlB {
		e.switchToNextCanvas()
		e.ctrlAState = false
		e.ctrlLState = false
		e.endSelection()
		return
	}

	switch ev.Key() {
	case tcell.KeyCtrlN:
		e.createNewCanvas()
		e.ctrlAState = false
		e.ctrlLState = false
		e.endSelection()
	case tcell.KeyCtrlW:
		selectedText := e.getSelectedText()
		hasSelection := strings.TrimSpace(selectedText) != ""
		var sourceText string
		if hasSelection {
			sourceText = selectedText
		} else {
			sourceText = e.lines[e.cy]
		}
		e.promptShow("Translate to language", func(input string) {
			defaultLang := detectSystemLanguage()
			targetLang := strings.TrimSpace(input)
			if targetLang == "" {
				targetLang = defaultLang
			}
			prompt := e.translationPrompt(sourceText, targetLang)

			translation, llmErr := e.llmQueryTranslate(prompt)
			if llmErr != nil {
				e.showError("LLM error: " + llmErr.Error())
				return
			}
			translation = strings.TrimSpace(translation)
			if translation == "" {
				e.statusMessage("Translation is empty")
				return
			}
			if hasSelection {
				e.deleteSelection()
			}
			e.insertTextAtCursor(translation)
		})
		e.ctrlAState = false
		e.ctrlLState = false
	case tcell.KeyDelete:
		e.deleteWordAfterCursor()
		e.ctrlAState = false
		e.ctrlLState = false
		// можно вернуть, если не хотите дальше обрабатывать Delete
		// you can return if you do not want to process Delete further.
		// return

	case tcell.KeyCtrlT:
		e.showTerminalPrompt()
	case tcell.KeyCtrlR:
		e.handleRunCode()
		e.ctrlAState = false
		e.ctrlLState = false
	case tcell.KeyCtrlP:
		if e.multiLinePrompt != nil && strings.TrimSpace(e.multiLinePrompt.Value) != "" {
			instruction := e.multiLinePrompt.Value
			e.multiLinePrompt = nil
			e.llmQueryWithProjectContext(instruction)
			e.resetEditorStates()
		} else if e.githubProject != nil {
			e.saveToGitHub()
		} else {
			e.showError("Not a GitHub project. Open a GitHub URL to enable this feature.")
		}
		e.resetEditorStates()
		e.ctrlAState = false
		e.ctrlLState = false
	case tcell.KeyCtrlJ:
		helpText := e.getUsageText()
		e.multiLinePrompt = &MultiLinePrompt{
			Label: "Help (Press Esc to close)",
			Value: helpText,
			Callback: func(input string) {
				e.multiLinePrompt = nil
				e.render()
			},
		}
		e.prompt = nil
		e.render()
	case tcell.KeyCtrlK:
		if e.selecting && e.lineSelecting {
			e.toggleCommentSelection()
		} else {
			e.toggleCommentLine()
		}
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
		e.ctrlLState = false
	case tcell.KeyCtrlU:
		e.indentSelection()
		e.ctrlAState = false
		e.ctrlLState = false
	case tcell.KeyCtrlY:
		e.unindentSelection()
		e.ctrlAState = false
		e.ctrlLState = false

	case tcell.KeyCtrlQ:

		e.ctrlAState = false
		e.ctrlLState = false
		e.handleExitWithCanvasCheck()
	case tcell.KeyCtrlF:
		e.promptShowWithInitial("Search", e.lastSearch, func(input string) {
			trimmed := strings.TrimSpace(input)
			if trimmed == "" {
				e.lastSearch = input
				return
			}
			if strings.Contains(trimmed, " -> ") {
				parts := strings.SplitN(trimmed, " -> ", 2)
				if len(parts) == 2 {
					old := parts[0]
					newS := parts[1]
					replaced := e.replaceAllOccurrences(old, newS)
					e.statusMessage(fmt.Sprintf("Replaced %d occurrence(s) of %q with %q", replaced, old, newS))
					e.prompt = nil
					return
				}
			}

			e.findAndJump(input)
			if strings.TrimSpace(input) != "" {
				e.lastSearch = input
			}
		})
		e.ctrlAState = false
		e.ctrlLState = false
	case tcell.KeyCtrlL:
		e.llmPromptWithPrevShow()
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
		e.ctrlLState = false
	case tcell.KeyCtrlZ:
		e.undo()
		e.ctrlAState = false
		e.ctrlLState = false
		e.endSelection()
	case tcell.KeyCtrlE:
		e.redo()
		e.ctrlAState = false
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
		e.ctrlLState = false
	case tcell.KeyCtrlO:
		e.promptShow("Open file (path)", func(input string) {
			p := strings.TrimSpace(input)
			if p != "" {
				if e.filename != "" {
					if info, err := os.Stat(e.filename); err == nil && info.IsDir() {
						e.openProjectFile(p)
						return
					}
				}
				e.openFile(p)
			}
		})
		e.ctrlAState = false
		e.ctrlLState = false
		e.endSelection()
	case tcell.KeyCtrlC:
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
		e.ctrlLState = false
	case tcell.KeyCtrlV:
		if e.selecting {
			e.deleteSelection()
		}
		e.pasteFromClipboard()
		e.ctrlAState = false
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
		e.ctrlLState = false
	case tcell.KeyHome:
		if shiftPressed {
			e.startSelection()
		} else if e.selecting {
			e.endSelection()
		}
		e.cx = 0
		e.ctrlAState = false
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
		e.ctrlLState = false
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if e.lastSearch != "" && e.cx == 0 && e.cy == 0 && !e.selecting {
			e.lastSearch = ""
			e.statusMessage("Last search cleared")
			return
		}

		if e.selecting {
			e.deleteSelection()
		} else {
			e.backspace()
		}
		e.ctrlAState = false
		e.ctrlLState = false
	case tcell.KeyEscape:
		e.endSelection()
		e.ctrlAState = false
		e.ctrlLState = false
		e.terminalPrompt = nil
		// Hide tooltip if visible
		// if e.tooltipManager != nil {
		// 	e.tooltipManager.hideTooltip()
		// }

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
			switch r {
			case '(', '[', '{', '"', '\'':
				if e.shouldAutoCloseBracket(r) {
					closing := getClosingBracket(r)
					if closing != 0 {
						e.insertTextAtCursor(string([]rune{r, closing}))
						e.cx--
						return
					}
				}
				fallthrough
			default:
				e.insertRune(r)
			}
			e.ctrlAState = false
		}
		// if !shiftPressed && ev.Key() != tcell.KeyShift && e.selecting {
		//     e.endSelection()
		// }
	}
	e.ensureVisible()
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
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(e.lines) {
		endLine = len(e.lines) - 1
	}
	if startLine > endLine {
		startLine, endLine = endLine, startLine
	}

	if e.lineSelecting {
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
			if startLine >= 0 && startLine < len(e.lines) {
				lineRunes := []rune(e.lines[startLine])
				if startCol < 0 {
					startCol = 0
				}
				if endCol > len(lineRunes) {
					endCol = len(lineRunes)
				}
				if startCol < endCol {
					e.lines[startLine] = string(append(lineRunes[:startCol], lineRunes[endCol:]...))
				}
				e.cx = startCol
				e.cy = startLine
			}
		} else {
			if startLine >= 0 && startLine < len(e.lines) && endLine >= 0 && endLine < len(e.lines) {
				firstLineRunes := []rune(e.lines[startLine])
				lastLineRunes := []rune(e.lines[endLine])
				if startCol < 0 {
					startCol = 0
				}
				if endCol > len(lastLineRunes) {
					endCol = len(lastLineRunes)
				}
				if startCol <= len(firstLineRunes) && endCol <= len(lastLineRunes) {
					merged := string(append(firstLineRunes[:startCol], lastLineRunes[endCol:]...))
					e.lines = append(e.lines[:startLine], append([]string{merged}, e.lines[endLine+1:]...)...)
				}
				e.cy = startLine
				e.cx = startCol
			}
		}
	}

	e.endSelection()
	e.dirty = true
	e.redoStack = nil
	if e.cy < 0 {
		e.cy = 0
	}
	if e.cy >= len(e.lines) {
		e.cy = len(e.lines) - 1
		if e.cy < 0 {
			e.cy = 0
		}
	}
	if e.cx < 0 {
		e.cx = 0
	}
	if e.cy < len(e.lines) {
		lineRunes := []rune(e.lines[e.cy])
		if e.cx > len(lineRunes) {
			e.cx = len(lineRunes)
		}
	}
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

// showError displays an error message in the status bar with red background.
// showError отображает сообщение об ошибке в строке состояния с красным фоном.
func (e *Editor) showError(msg string) {
	e.errorMessage = msg
	e.errorShowTime = time.Now()
	e.render()
	go func() {
		time.Sleep(5 * time.Second)
		if time.Since(e.errorShowTime) >= 5*time.Second {
			e.errorMessage = ""
			e.render()
		}
	}()
}

func (e *Editor) isLLMModeActive() bool {
	return e.multiLinePrompt != nil
}

// render renders the editor to the screen.
// render отображает редактор на экране.
func (e *Editor) render() {
	// Update tooltip based on cursor position
	// if e.tooltipManager != nil {
	// 	e.tooltipManager.updateTooltip()
	// }

	e.screen.Clear()
	display := e.buildDisplayBuffer()
	total := len(display)
	topLine, bottomLine1, bottomLine2 := e.statusBar()
	if e.isLLMModeActive() {
		bottomLine1 = ""
		bottomLine2 = ""
	}

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
		styleSelection := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGray)
		styleSelectionCurrentLine := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGray)
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
				style = style.Background(tcell.ColorBlack)
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
				style = styleDefault.Background(tcell.ColorBlack)
			}
			e.screen.SetContent(x, i+1, ' ', nil, style)
		}
	}

	if e.bracketMatcher != nil {
		matchingPair := e.bracketMatcher.getBracketAtCursor()
		if matchingPair != nil {
			openDisplayRow := 0
			openSegIndex := 0
			openCursorInSeg := 0

			closeDisplayRow := 0
			closeSegIndex := 0
			closeCursorInSeg := 0
			totalBefore := 0
			for i := 0; i < matchingPair.OpenLine; i++ {
				totalBefore += len(e.wrapLine(e.lines[i]))
			}
			segs := e.wrapLine(e.lines[matchingPair.OpenLine])
			segmentStartRune := 0
			for segIndex, seg := range segs {
				segRunes := []rune(seg)
				segEndRune := segmentStartRune + len(segRunes)
				if matchingPair.OpenCol >= segmentStartRune && matchingPair.OpenCol <= segEndRune {
					offsetInSegRunes := matchingPair.OpenCol - segmentStartRune
					offsetInSegCells := 0
					for i := 0; i < offsetInSegRunes; i++ {
						r := segRunes[i]
						if r == '\t' {
							offsetInSegCells += 4 - (offsetInSegCells % 4)
						} else {
							offsetInSegCells += runewidth.RuneWidth(r)
						}
					}
					openDisplayRow = totalBefore + segIndex
					openSegIndex = segIndex
					openCursorInSeg = offsetInSegCells
					break
				}
				segmentStartRune = segEndRune
			}

			totalBefore = 0
			for i := 0; i < matchingPair.CloseLine; i++ {
				totalBefore += len(e.wrapLine(e.lines[i]))
			}
			segs = e.wrapLine(e.lines[matchingPair.CloseLine])
			segmentStartRune = 0
			for segIndex, seg := range segs {
				segRunes := []rune(seg)
				segEndRune := segmentStartRune + len(segRunes)
				if matchingPair.CloseCol >= segmentStartRune && matchingPair.CloseCol <= segEndRune {
					offsetInSegRunes := matchingPair.CloseCol - segmentStartRune
					offsetInSegCells := 0
					for i := 0; i < offsetInSegRunes; i++ {
						r := segRunes[i]
						if r == '\t' {
							offsetInSegCells += 4 - (offsetInSegCells % 4)
						} else {
							offsetInSegCells += runewidth.RuneWidth(r)
						}
					}
					closeDisplayRow = totalBefore + segIndex
					closeSegIndex = segIndex
					closeCursorInSeg = offsetInSegCells
					break
				}
				segmentStartRune = segEndRune
			}
			openY := openDisplayRow - e.offsetY + 1
			closeY := closeDisplayRow - e.offsetY + 1

			if openY >= 1 && openY < e.contentHeight-3 {
				for i := 0; i < contentRows; i++ {
					di := e.offsetY + i
					if di == openDisplayRow {
						row := display[di]
						if openSegIndex == row.segIndex {
							bracketStyle := e.bracketMatcher.getBracketHighlightStyle()
							if matchingPair.OpenLine < len(e.lines) {
								lineRunes := []rune(e.lines[matchingPair.OpenLine])
								if matchingPair.OpenCol < len(lineRunes) {
									e.screen.SetContent(openCursorInSeg, openY, lineRunes[matchingPair.OpenCol], nil, bracketStyle)
								}
							}
						}
						break
					}
				}
			}

			if closeY >= 1 && closeY < e.contentHeight-3 {
				for i := 0; i < contentRows; i++ {
					di := e.offsetY + i
					if di == closeDisplayRow {
						row := display[di]
						if closeSegIndex == row.segIndex {
							bracketStyle := e.bracketMatcher.getBracketHighlightStyle()
							if matchingPair.CloseLine < len(e.lines) {
								lineRunes := []rune(e.lines[matchingPair.CloseLine])
								if matchingPair.CloseCol < len(lineRunes) {
									e.screen.SetContent(closeCursorInSeg, closeY, lineRunes[matchingPair.CloseCol], nil, bracketStyle)
								}
							}
						}
						break
					}
				}
			}
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
				e.screen.SetContent(xPos+cellOffset, promptLine, drawRune, nil, tcell.StyleDefault.Background(tcell.NewRGBColor(211, 211, 211)).Foreground(tcell.ColorBlack))
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
		if numLinesToShow > 25 {
			numLinesToShow = 25
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
	if !e.canvasWarningTime.IsZero() && time.Since(e.canvasWarningTime) < 3*time.Second {
		warningMsg := "Maximum number of canvases: " + strconv.Itoa(MaxCanvases)
		for i := 0; i < e.contentWidth; i++ {
			e.screen.SetContent(i, e.contentHeight-1, ' ', nil,
				tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack))
		}

		runes := []rune(" " + warningMsg)
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
				e.screen.SetContent(xPos+cellOffset, e.contentHeight-1, drawRune, nil,
					tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack))
			}
			xPos += rw
		}
	} else if e.errorMessage != "" && time.Since(e.errorShowTime) < 3*time.Second {

		for i := 0; i < e.contentWidth; i++ {
			e.screen.SetContent(i, e.contentHeight-1, ' ', nil,
				tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite))
		}

		runes := []rune(" " + e.errorMessage)
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
				e.screen.SetContent(xPos+cellOffset, e.contentHeight-1, drawRune, nil,
					tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite))
			}
			xPos += rw
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
