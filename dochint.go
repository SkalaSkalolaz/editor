// dochint.go
// Показывает описание функции при её вызове в коде.

package main

import (
	"regexp"
	"strings"
	"time"
	"unicode"
)

// FunctionDocHint хранит текущее отображаемое описание
type FunctionDocHint struct {
	active    bool
	content   string
	startTime time.Time
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

	doc := e.findFunctionDoc(e.language, name)
	if doc == "" {
		return
	}

	if e.funcDocHint == nil {
		e.funcDocHint = &FunctionDocHint{}
	}
	e.funcDocHint.active = true
	e.funcDocHint.content = doc
	e.funcDocHint.startTime = time.Now()
	e.render()
}

// hideFunctionDoc скрывает подсказку при закрытии ')'
func (e *Editor) hideFunctionDoc() {
	if e.funcDocHint != nil && e.funcDocHint.active {
		e.funcDocHint.active = false
		e.render()
	}
}

// findFunctionDoc ищет определение функции и возвращает её комментарий
func (e *Editor) findFunctionDoc(lang Language, name string) string {
	pattern := ""
	switch lang {
	case LangGo:
		pattern = `(?m)(?P<comment>(?://[^\n]*\n)*)\s*func\s+` + name + `\s*\(`
	case LangC, LangCpp:
		pattern = `(?m)(?P<comment>(?://[^\n]*\n|/\*.*?\*/\s*)*)\s*(?:\w+\s+)+` + name + `\s*\(`
	case LangPython, LangRuby:
		pattern = `(?m)(?P<comment>(?:#.*\n)*)\s*def\s+` + name + `\s*\(`
	case LangKotlin, LangSwift:
		pattern = `(?m)(?P<comment>(?://[^\n]*\n|/\*.*?\*/\s*)*)\s*fun\s+` + name + `\s*\(`
	case LangFortran:
		pattern = `(?m)(?P<comment>(?:!.*\n)*)\s*(?:function|subroutine)\s+` + name
	case LangLisp:
		pattern = `(?m)(?P<comment>(?:;.*\n)*)\s*\(defun\s+` + name
	case LangAssembly:
		pattern = `(?m)(?P<comment>(?:;.*\n)*)\s*` + name + `:`
	case LangHTML:
		pattern = `(?m)(?P<comment><!--.*?-->)\s*<\s*script.*` + name
	default:
		return ""
	}

	re := regexp.MustCompile(pattern)

	for _, canvas := range e.canvases {
		text := strings.Join(canvas.lines, "\n")
		match := re.FindStringSubmatch(text)
		if match != nil {
			for i, name := range re.SubexpNames() {
				if name == "comment" {
					comment := strings.TrimSpace(match[i])
					comment = cleanCommentPrefix(comment, lang)
					return comment
				}
			}
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
	name := strings.TrimSpace(string(runes[i+1:]))
	if len(name) == 0 {
		return ""
	}
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

// renderFunctionDocHint отображает подсказку прямо над строкой вызова функции.
func (e *Editor) renderFunctionDocHint() {
    if e.funcDocHint == nil || !e.funcDocHint.active {
        return
    }

    doc := e.funcDocHint.content
    if strings.TrimSpace(doc) == "" {
        return
    }

    lines := strings.Split(doc, "\n")

    displayRow, _, _ := e.cursorDisplayPosition()

    yAbove := displayRow - e.offsetY

    if yAbove <= len(lines) {
        yAbove = len(lines)
    }

    startY := yAbove - len(lines) + 1
    if startY < 0 {
        startY = 0
    }

    if startY+len(lines) >= e.contentHeight-1 {
        lines = lines[:max(1, e.contentHeight-startY-1)]
    }

    style := styleComment

    for i, line := range lines {
        runes := []rune(line)
        for x, r := range runes {
            if x >= e.contentWidth {
                break
            }
            e.screen.SetContent(x, startY+i, r, nil, style)
        }

        for x := len(runes); x < e.contentWidth; x++ {
            e.screen.SetContent(x, startY+i, ' ', nil, styleComment)
        }
    }
}

// вспомогательная функция для безопасности
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
