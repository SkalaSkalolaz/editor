package main

import (
	"github.com/gdamore/tcell/v2"
)


// BracketPair представляет пару совпадающих скобок и их позиций.
// BracketPair represents a pair of matching brackets and their positions
type BracketPair struct {
	OpenLine  int
	OpenCol   int
	CloseLine int
	CloseCol  int
}

// BracketMatcher отвечает за поиск соответствующих скобок.
// BracketMatcher is responsible for finding matching brackets
type BracketMatcher struct {
	editor *Editor
}

// NewBracketMatcher creates a new bracket matcher
func NewBracketMatcher(editor *Editor) *BracketMatcher {
	return &BracketMatcher{
		editor: editor,
	}
}

// findMatchingBracket finds the matching bracket for the character at the given position
func (bm *BracketMatcher) findMatchingBracket(lineIdx, colIdx int) *BracketPair {
	if lineIdx < 0 || lineIdx >= len(bm.editor.lines) {
		return nil
	}

	line := bm.editor.lines[lineIdx]
	if colIdx < 0 || colIdx >= len([]rune(line)) {
		return nil
	}

	runes := []rune(line)
	char := runes[colIdx]

	openBrackets := map[rune]rune{
		'(': ')',
		'[': ']',
		'{': '}',
	}

	closeBrackets := map[rune]rune{
		')': '(',
		']': '[',
		'}': '{',
	}

	if closing, isOpen := openBrackets[char]; isOpen {
		return bm.findClosingBracket(lineIdx, colIdx, char, closing)
	}

	if opening, isClose := closeBrackets[char]; isClose {
		return bm.findOpeningBracket(lineIdx, colIdx, opening, char)
	}

	return nil
}

// findClosingBracket searches forward for a matching closing bracket
func (bm *BracketMatcher) findClosingBracket(startLine, startCol int, opening, closing rune) *BracketPair {
	if startLine < 0 || startLine >= len(bm.editor.lines) {
		return nil
	}
	
	line := bm.editor.lines[startLine]
	runes := []rune(line)
	
	if startCol < 0 || startCol >= len(runes) {
		return nil
	}

	nesting := 1
	
	lineIdx := startLine
	colIdx := startCol + 1
	
	for lineIdx < len(bm.editor.lines) {
		if lineIdx < 0 || lineIdx >= len(bm.editor.lines) {
			return nil
		}
		
		line := bm.editor.lines[lineIdx]
		runes := []rune(line)
		
		for colIdx < len(runes) {
			if colIdx < 0 || colIdx >= len(runes) {
				break
			}
			
			char := runes[colIdx]
			
			if char == opening {
				nesting++
			} else if char == closing {
				nesting--
				if nesting == 0 {
					return &BracketPair{
						OpenLine:  startLine,
						OpenCol:   startCol,
						CloseLine: lineIdx,
						CloseCol:  colIdx,
					}
				}
			}
			
			colIdx++
		}
		
		lineIdx++
		colIdx = 0
	}
	
	return nil
}

// findOpeningBracket searches backward for a matching opening bracket
func (bm *BracketMatcher) findOpeningBracket(startLine, startCol int, opening, closing rune) *BracketPair {
	if startLine < 0 || startLine >= len(bm.editor.lines) {
		return nil
	}
	
	line := bm.editor.lines[startLine]
	runes := []rune(line)
	
	if startCol < 0 || startCol >= len(runes) {
		return nil
	}

	nesting := 1
	
	lineIdx := startLine
	colIdx := startCol - 1
	
	for lineIdx >= 0 {
		if lineIdx < 0 || lineIdx >= len(bm.editor.lines) {
			return nil
		}
		
		line := bm.editor.lines[lineIdx]
		runes := []rune(line)
		
		if colIdx < 0 {
			lineIdx--
			if lineIdx >= 0 && lineIdx < len(bm.editor.lines) {
				colIdx = len([]rune(bm.editor.lines[lineIdx])) - 1
			}
			continue
		}
		
		for colIdx >= 0 {
			if colIdx < 0 || colIdx >= len(runes) {
				break
			}
			
			char := runes[colIdx]
			
			if char == closing {
				nesting++
			} else if char == opening {
				nesting--
				if nesting == 0 {
					return &BracketPair{
						OpenLine:  lineIdx,
						OpenCol:   colIdx,
						CloseLine: startLine,
						CloseCol:  startCol,
					}
				}
			}
			
			colIdx--
			
			if colIdx < 0 && lineIdx > 0 {
				lineIdx--
				if lineIdx >= 0 && lineIdx < len(bm.editor.lines) {
					colIdx = len([]rune(bm.editor.lines[lineIdx])) - 1
				} else {
					break
				}
			}
		}
		
		lineIdx--
		if lineIdx >= 0 && lineIdx < len(bm.editor.lines) {
			colIdx = len([]rune(bm.editor.lines[lineIdx])) - 1
		}
	}
	
	return nil
}

// getBracketAtCursor returns the bracket pair at the current cursor position
func (bm *BracketMatcher) getBracketAtCursor() *BracketPair {
	if bm.editor.cy < 0 || bm.editor.cy >= len(bm.editor.lines) {
		return nil
	}
	
	line := bm.editor.lines[bm.editor.cy]
	runes := []rune(line)
	
	if bm.editor.cx >= 0 && bm.editor.cx < len(runes) {
		pair := bm.findMatchingBracket(bm.editor.cy, bm.editor.cx)
		if pair != nil {
			return pair
		}
	}
	
	if bm.editor.cx > 0 && bm.editor.cx <= len(runes) {
		pair := bm.findMatchingBracket(bm.editor.cy, bm.editor.cx-1)
		if pair != nil {
			return pair
		}
	}
	
	return nil
}

// getBracketHighlightStyle returns the style for highlighting matched brackets
func (bm *BracketMatcher) getBracketHighlightStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlue)
}