// dochint.go
// Показывает описание функции при её вызове в коде с динамическим обновлением аргументов

package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// FunctionDocHint хранит текущее отображаемое описание
type FunctionDocHint struct {
	active      bool
	content     string
	fullContent string   
	startTime   time.Time
	startCX     int       
	message     string
}

// detectFunctionCall вызывается при вводе '('
// Определяет имя функции и ищет её описание
func (e *Editor) detectFunctionCall() {
	if e.language == LangUnknown {
		return
	}

	line := e.lines[e.cy]
	if e.cx == 0 || e.cx > len([]rune(line)) {
		return
	}
	before := string([]rune(line)[:e.cx])
	name := extractFunctionName(before)
	if name == "" {
		return
	}

	shortDoc, fullDoc := e.findFunctionDocWithFullDescription(e.language, name)
	if e.funcDocHint == nil {
		e.funcDocHint = &FunctionDocHint{}
	}

	e.funcDocHint.startTime = time.Now()
	e.funcDocHint.startCX = e.cx - 1
	e.funcDocHint.active = true

	if shortDoc == "" {
		e.funcDocHint.content = ""
		e.funcDocHint.fullContent = ""
		e.funcDocHint.message = fmt.Sprintf("Function '%s' not found", name)
	} else {
		e.funcDocHint.content = shortDoc
		e.funcDocHint.fullContent = fullDoc
		e.funcDocHint.message = ""
	}

	e.render()
}

// updateFunctionDoc динамически обновляет подсказку, если курсор внутри аргументов функции
func (e *Editor) updateFunctionDoc() {
	if e.funcDocHint == nil || !e.funcDocHint.active {
		return
	}

	line := e.lines[e.cy]
	if e.cx <= e.funcDocHint.startCX || e.cx > len([]rune(line)) {
		e.funcDocHint.active = false
		e.render()
		return
	}

	openParens := 0
	for _, r := range []rune(line)[e.funcDocHint.startCX:] {
		if r == '(' {
			openParens++
		} else if r == ')' {
			openParens--
			if openParens <= 0 {
				e.funcDocHint.active = false
				e.render()
				return
			}
		}
	}

	e.renderFunctionDocHint()
}

// hideFunctionDoc скрывает подсказку
func (e *Editor) hideFunctionDoc() {
	if e.funcDocHint != nil && e.funcDocHint.active {
		e.funcDocHint.active = false
		e.funcDocHint.message = ""
		e.render()
	}
}

// findFunctionDocWithFullDescription ищет документацию и возвращает краткую и полную версии
func (e *Editor) findFunctionDocWithFullDescription(lang Language, name string) (shortDoc, fullDoc string) {
	pattern := ""
	switch lang {
	case LangGo:
		pattern = `(?m)(?P<comment>(?://[^\n]*\n|/\*.*?\*/\s*)*)\s*(?P<signature>func\s*(?:\([^\)]*\)\s*)?` + name + `\s*\([^\)]*\))`
	case LangC, LangCpp:
		pattern = `(?m)(?P<comment>(?://[^\n]*\n|/\*.*?\*/\s*)*)\s*(?P<signature>(?:\w+\s+)+` + name + `\s*\([^\)]*\))`
	case LangPython, LangRuby:
		pattern = `(?m)(?P<comment>(?:#.*\n|""".*?"""\n|'''.*?'''\n)*)\s*(?P<signature>def\s+` + name + `\s*\([^\)]*\))`
	case LangKotlin, LangSwift:
		pattern = `(?m)(?P<comment>(?://[^\n]*\n|/\*.*?\*/\s*)*)\s*(?P<signature>fun\s+` + name + `\s*\([^\)]*\))`
	case LangFortran:
		pattern = `(?m)(?P<comment>(?:!.*\n)*)\s*(?P<signature>(?:function|subroutine)\s+` + name + `[^\n]*)`
	case LangLisp:
		pattern = `(?m)(?P<comment>(?:;.*\n)*)\s*(?P<signature>\(defun\s+` + name + `[^\)]*\))`
	case LangAssembly:
		pattern = `(?m)(?P<comment>(?:;.*\n)*)\s*(?P<signature>` + name + `:)`
	case LangHTML:
		pattern = `(?m)(?P<comment><!--.*?-->)\s*(?P<signature><\s*script.*` + name + `)`
	default:
		return "", ""
	}

	re := regexp.MustCompile(pattern)
	for _, canvas := range e.canvases {
		text := strings.Join(canvas.lines, "\n")
		match := re.FindStringSubmatch(text)
		if match == nil {
			continue
		}

		var comment, signature string
		for i, group := range re.SubexpNames() {
			if group == "comment" {
				comment = strings.TrimSpace(match[i])
				comment = cleanCommentPrefix(comment, lang)
			}
			if group == "signature" {
				signature = strings.TrimSpace(match[i])
			}
		}

		if signature != "" {
			shortDoc = signature
			fullDoc = signature
			if comment != "" {
				fullDoc += "\n" + comment
				shortDoc += " — " + firstLine(comment)
			}
			return shortDoc, fullDoc
		}
	}
	return "", ""
}

// firstLine возвращает первую непустую строку комментария
func firstLine(comment string) string {
	lines := strings.Split(comment, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			return l
		}
	}
	return ""
}

// extractFunctionName достает имя функции перед '('
func extractFunctionName(before string) string {
	runes := []rune(before)
	i := len(runes) - 1
	for i >= 0 && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
		i--
	}
	name := string(runes[i+1:])
	return name
}

// cleanCommentPrefix очищает префиксы комментариев
func cleanCommentPrefix(comment string, lang Language) string {
	lines := strings.Split(comment, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
		switch lang {
		case LangGo, LangC, LangCpp, LangKotlin, LangSwift:
			lines[i] = strings.TrimPrefix(lines[i], "//")
		case LangPython, LangRuby:
			lines[i] = strings.TrimPrefix(lines[i], "#")
			lines[i] = strings.Trim(lines[i], `"'`)
		case LangLisp, LangAssembly:
			lines[i] = strings.TrimPrefix(lines[i], ";")
		case LangFortran:
			lines[i] = strings.TrimPrefix(lines[i], "!")
		case LangHTML:
			lines[i] = strings.TrimPrefix(lines[i], "<!--")
			lines[i] = strings.TrimSuffix(lines[i], "-->")
		}
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}

// renderFunctionDocHint отображает подсказку прямо над строкой вызова функции с подсветкой
func (e *Editor) renderFunctionDocHint() {
	if e.funcDocHint == nil || !e.funcDocHint.active {
		return
	}

	var lines []string
	if e.funcDocHint.message != "" {
		lines = []string{e.funcDocHint.message}
	} else {
		lines = strings.Split(e.funcDocHint.fullContent, "\n")
	}

	displayRow, _, _ := e.cursorDisplayPosition()
	yAbove := displayRow - e.offsetY
	if yAbove <= len(lines) {
		yAbove = len(lines)
	}

	startY := yAbove - len(lines) - 1
	if startY < 0 {
		startY = 0
	}
	if startY+len(lines) >= e.contentHeight-1 {
		lines = lines[:max(1, e.contentHeight-startY-1)]
	}

	for i, line := range lines {
		styled := highlightSignature(line)
		for x, r := range styled {
			if x >= e.contentWidth {
				break
			}
			e.screen.SetContent(x, startY+i, r.r, nil, r.style)
		}
		for x := len([]rune(line)); x < e.contentWidth; x++ {
			e.screen.SetContent(x, startY+i, ' ', nil, styleComment)
		}
	}
}

// структура для подсветки символов
type styledRune struct {
	r     rune
	style tcell.Style
}

// highlightSignature подсвечивает ключевые слова и типы
func highlightSignature(line string) []styledRune {
	keywords := []string{"func", "def", "function", "subroutine", "fun", "return"}
	types := []string{"int", "float", "float64", "string", "bool", "char", "double", "var", "let", "Integer", "REAL"}

	runes := []rune(line)
	result := make([]styledRune, len(runes))
	for i, r := range runes {
		result[i] = styledRune{r, styleComment}
	}

	wordStart := -1
	for i, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			if wordStart == -1 {
				wordStart = i
			}
		} else {
			if wordStart != -1 {
				word := string(runes[wordStart:i])
				if contains(keywords, word) {
					for j := wordStart; j < i; j++ {
						result[j].style = styleKeyword
					}
				} else if contains(types, word) {
					for j := wordStart; j < i; j++ {
						result[j].style = styleType
					}
				}
				wordStart = -1
			}
		}
	}
	if wordStart != -1 {
		word := string(runes[wordStart:])
		if contains(keywords, word) {
			for j := wordStart; j < len(runes); j++ {
				result[j].style = styleKeyword
			}
		} else if contains(types, word) {
			for j := wordStart; j < len(runes); j++ {
				result[j].style = styleType
			}
		}
	}
	return result
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}