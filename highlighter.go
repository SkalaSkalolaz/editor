package main

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// HighlightedToken represents a token with its style.
// HighlightedToken представляет токен с его стилем.
type HighlightedToken struct {
	Text  string
	Style tcell.Style
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
