package main

import "strings"

// EditorState представляет состояние редактора для undo/redo.
type EditorState struct {
	Lines []string
	Cx    int
	Cy    int
}

// replaceAllOccurrences заменяет во всём документе все вхождения old на new.
// Возвращает число сделанных замен.
func (e *Editor) replaceAllOccurrences(old, new string) int {
	if old == "" {
		return 0
	}
	e.pushUndo()
	count := 0
	for i := range e.lines {
		count += strings.Count(e.lines[i], old)
		e.lines[i] = strings.ReplaceAll(e.lines[i], old, new)
	}
	e.dirty = true
	e.ensureVisible()
	return count
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
			e.lastSearch = strings.TrimSpace(query)
			return
		}
	}
	e.statusMessage("Not found: " + q)
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

// pushUndo pushes the current state onto the undo stack.
// pushUndo помещает текущее состояние в стек отмены.
func (e *Editor) pushUndo() {
	state := EditorState{
		Lines: make([]string, len(e.lines)),
		Cx:    e.cx,
		Cy:    e.cy,
	}
	copy(state.Lines, e.lines)
	e.undoStack = append(e.undoStack, state)
	e.redoStack = nil
	e.dirty = true
}

// undo reverts the last change.
// undo отменяет последнее изменение.
func (e *Editor) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	currentState := EditorState{
		Lines: make([]string, len(e.lines)),
		Cx:    e.cx,
		Cy:    e.cy,
	}
	copy(currentState.Lines, e.lines)
	e.redoStack = append(e.redoStack, currentState)
	lastState := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]

	e.lines = lastState.Lines
	e.cx = lastState.Cx
	e.cy = lastState.Cy
	e.ensureVisible()
}

// redo reapplies the last undone change.
// redo повторно применяет последнее отмененное изменение.
func (e *Editor) redo() {
	if len(e.redoStack) == 0 {
		return
	}
	currentState := EditorState{
		Lines: make([]string, len(e.lines)),
		Cx:    e.cx,
		Cy:    e.cy,
	}
	copy(currentState.Lines, e.lines)
	e.undoStack = append(e.undoStack, currentState)
	nextState := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.lines = nextState.Lines
	e.cx = nextState.Cx
	e.cy = nextState.Cy
	e.ensureVisible()
}
