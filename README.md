# Code Editor GO: An advanced text editor with backlight and LLM integration

[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)  
[![Go Version](https://img.shields.io/badge/go-1.25.1-blue.svg)](https://golang.org/dl/)

Version: **0.9.11**
![Editor's screenshot](EditorGO_0_9_8.png)

## Project Description

Code Editor GO is a console text editor for professional development with support for a variety of languages
easy-to-use navigation, editing commands, and built-in integration with LLM (Large Language Models) for code generation, analysis, and translation.

## Key features

- Multi-language syntax highlighting and auto-detection: C, C++, Assembler, Fortran, Go, Python, Ruby, Kotlin, Swift, HTML, Lisp, etc.
- Integration with LLM providers: Pollinations, OpenRouter, Ollama, LLM7, as well as any API URL.
- Auto-completion of keywords and identifiers, auto-closing of brackets.
- Built-in terminal (Ctrl-T) for executing OS commands and inserting output into the editor.
- Run and debug code in different languages with error analysis via LLM (Ctrl-R).
- Undo/Redo (Ctrl-Z, Ctrl-E), cut/copy/paste, multi-line selection.
- Search, mass replacement, switching to a line, commenting on a block/line.
- Multiple "canvases" (working buffers) within the same session.
- Support for working with GitHub projects: ZIP cloning, structure overview, commit/push (Ctrl-P).
- Translation of text or selected code into any language with replacement (Ctrl-W).

## Installation
```
git clone https://github.com/SkalaSkalolaz/editor.git
cd editor
go build -o editor .
```

## Quick start

```
go run .
```


# Running the editor
```
./editor [provider]/[URL provider] [model] [path to file]/[directory]/[GitHub URL] [API key] [GitHub key]
```

If the path points to the project directory, the editor will automatically upload an overview of the files and create canvases for each source.

# ATTENTION: 
		when connecting to your project, overwriting files on a local PC from the GitHub server deletes the data on the server. It is NECESSARY to resend all files to the server via Ctrl +P. When you open a project without overwriting it, the data remains on the server.

## Keyboard shortcuts

| Key 	 | Action 															|
|--------|------------------------------------------------------------------|
| Ctrl-L | Enter an LLM query / auto-generate code based on a comment 		|
| Ctrl-P | Send the project (or file) to GitHub 							|
| Ctrl-R | Run the code / in case of error — recommendations from LLM 		|
|        | Additional key for sending all project files as LLM data         |
| Ctrl-S | Save File 														|
| Ctrl-O | Open the file / Project file 									|
| Ctrl-N | New file / canvas 												|
| Ctrl-Q | Exit 															|
| Ctrl-F | Search / Replace (old -> new)                               		|
| Ctrl-G | Go to the line 													|
| Ctrl-Z | Undo 	                                                        |
| Ctrl-E | Redo		                                                        |
| Ctrl-X | Cut a line / block 												|
| Ctrl-C | Copy line / block 												|
| Ctrl-V | Paste from Buffer 												|
| Ctrl-T | Open the OS terminal 											|
| Ctrl-K | Comment / uncomment a line or selection                          |
| Ctrl-W | Translate text or highlighted block, replace in document         |
| Ctrl-B | Switching between canvases                                       |
| Ctrl-A | Select All                                                       |
| Ctrl-Y | Indent selected lines to the left                                |
| Ctrl-U | Indent selected lines to the right                               |
| Ctrl-J | Help                                                             |
| ←↑→↓, Home/End, PgUp/PgDn / Text navigation                               |

## Examples

- Open the file:
```
./editor /path/to/file.go
```
- Generation of Go code from the description:
```
./editor pollinations qwen3:1.7b /path/to/file.go
```  
- Processing LLM requests via standard streams
```
echo 'Analyze data' | ./editor -s --data  --input ./src ollama gemma3:4b"
```
- Text generation using an LLM based on the Openrouter provider with an access key
```
./editor openrouter deepseek/deepseek-chat-v3.1:free file.txt sn-...
```
- Work with code from a project that is located on the GitHub server
```
./editor pollinations qwen3:1.7b https://github.com/SkalaSkalolaz/editor ghp_
``
- Translation of the selected text (by default, into the system language):

  Ctrl-W → enter the language code → Enter

- Code launch and analysis:
  
  Ctrl-R  The LLM response will automatically substitute the recommendations or the results of the implementation.

## License

The project is distributed under the **BSD-3-Clause** license. The details are in the file. LICENSE.txt .

## Participation in development

We welcome your edits, fixes and ideas!  
Create a pull request or issue in the repository.

---

Thanks for using Code Editor GO!  
We wish you productive and comfortable work.