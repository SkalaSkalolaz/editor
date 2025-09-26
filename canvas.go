package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Максимальное количество канвасов
const MaxCanvases = 100

// Canvas представляет отдельный канвас редактора.
// Canvas represents a separate editor canvas.
type Canvas struct {
	filename      string
	lines         []string
	cx, cy        int
	offsetX       int
	offsetY       int
	dirty         bool
	language      Language
	undoStack     []EditorState
	redoStack     []EditorState
	githubProject *GitHubProject
}

// switchToNextCanvas переключается на следующий канвас по кругу.
func (e *Editor) switchToNextCanvas() {
	e.syncEditorToCanvas()
	nextCanvas := e.currentCanvas + 1
	if nextCanvas > MaxCanvases {
		nextCanvas = 1
	}
	for i := 0; i < MaxCanvases; i++ {
		if _, exists := e.canvases[nextCanvas]; exists {
			break
		}
		nextCanvas++
		if nextCanvas > MaxCanvases {
			nextCanvas = 1
		}
	}
	if _, exists := e.canvases[nextCanvas]; exists {
		e.currentCanvas = nextCanvas
		e.syncCanvasToEditor()
		e.ensureVisible()
		e.statusMessage("Canvas " + strconv.Itoa(nextCanvas))
	} else {
		e.statusMessage("There are no other canvases")
	}
}

// syncCanvasToEditor синхронизирует текущий канвас с редактором.
func (e *Editor) syncCanvasToEditor() {
	canvas, exists := e.canvases[e.currentCanvas]
	if !exists {
		return
	}

	e.filename = canvas.filename
	e.lines = canvas.lines
	e.cx = canvas.cx
	e.cy = canvas.cy
	e.offsetX = canvas.offsetX
	e.offsetY = canvas.offsetY
	e.dirty = canvas.dirty
	e.language = canvas.language
	e.undoStack = canvas.undoStack
	e.redoStack = canvas.redoStack
	if canvas.githubProject != nil {
		e.githubProject = canvas.githubProject
	}
}

// syncEditorToCanvas синхронизирует редактор с текущим канвасом.
func (e *Editor) syncEditorToCanvas() {
	canvas, exists := e.canvases[e.currentCanvas]
	if !exists {
		return
	}

	canvas.filename = e.filename
	canvas.lines = e.lines
	canvas.cx = e.cx
	canvas.cy = e.cy
	canvas.offsetX = e.offsetX
	canvas.offsetY = e.offsetY
	canvas.dirty = e.dirty
	canvas.language = e.language
	canvas.undoStack = e.undoStack
	canvas.redoStack = e.redoStack
	if e.githubProject != nil {
		canvas.githubProject = e.githubProject
	}
}

// createNewCanvas создает новый канвас.
func (e *Editor) createNewCanvas() {
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
	e.canvases[newCanvasNum] = &Canvas{
		filename: "",
		lines:    []string{""},
		cx:       0,
		cy:       0,
		offsetX:  0,
		offsetY:  0,
		dirty:    false,
		language: LangUnknown,
	}
	e.currentCanvas = newCanvasNum
	e.syncCanvasToEditor()
	e.ensureVisible()

	e.statusMessage("Canvas created " + strconv.Itoa(newCanvasNum))
}

// hasUnsavedChanges проверяет, есть ли несохраненные изменения в канвасе.
func (c *Canvas) hasUnsavedChanges() bool {
	return c.dirty
}

// saveAllCanvases сохраняет все канвасы с несохраненными изменениями
func (e *Editor) saveAllCanvases() error {
	currentCanvas := e.currentCanvas
	defer func() {
		e.currentCanvas = currentCanvas
		e.syncCanvasToEditor()
	}()

	for canvasNum, canvas := range e.canvases {
		if canvas.hasUnsavedChanges() {
			e.currentCanvas = canvasNum
			e.syncCanvasToEditor()

			if e.filename == "" {
				continue
			}

			if err := e.persist(); err != nil {
				return fmt.Errorf("failed to save canvas %d: %w", canvasNum, err)
			}

			e.syncEditorToCanvas()
		}
	}

	return nil
}

// getDisplayName возвращает отображаемое имя канваса.
func (c *Canvas) getDisplayName() string {
	if c.filename == "" {
		return "[new file]"
	}
	return c.filename
}

// canvas.go - добавляем функцию для получения списка файлов проекта
// getProjectFiles возвращает map всех файлов проекта из всех канвасов
func (e *Editor) getProjectFiles() map[string]string {
	files := make(map[string]string)

	for _, canvas := range e.canvases {
		if canvas.filename != "" && len(canvas.lines) > 0 {
			content := strings.Join(canvas.lines, "\n")
			files[canvas.filename] = content
		}
	}

	return files
}

// isProjectMode проверяет, работает ли редактор в режиме проекта (несколько файлов)
func (e *Editor) isProjectMode() bool {
	if e.githubProject != nil {
		return true
	}
	fileCount := 0
	for _, canvas := range e.canvases {
		if canvas.filename != "" {
			fileCount++
			if fileCount > 1 {
				return true
			}
		}
	}

	return false
}
