# Code Editor GO: An advanced text editor with backlight and LLM integration

[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)  
[![Go Version](https://img.shields.io/badge/go-1.25.1-blue.svg)](https://golang.org/dl/)

Version: **1.0.14**
![Editor's screenshot](Editor.png)

## Project Description
Code Editor GO is a console text editor for professional development with support for a variety of languages, easy-to-use navigation, editing commands, and built-in integration with LLM (Large Language Models) for code generation, analysis, and translation. Supports working with individual files, project directories (with categorized file overview), and GitHub repositories.

## Key features
- Multi-language syntax highlighting and auto-detection: C, C++, Assembler, Fortran, Go, Python, Ruby, Kotlin, Swift, HTML, Lisp, PlantUML, etc.
- Integration with LLM providers: Pollinations, OpenRouter, Ollama, LLM7, as well as any API URL.
- Auto-completion of keywords and identifiers, auto-closing of brackets.
- AI-powered code completion mode (flag `-a`) with ghost-text suggestions from LLM.
- Built-in terminal (Ctrl-T) for executing OS commands and inserting output into the editor.
- Run and debug code in different languages with error analysis via LLM (Ctrl-R).
- Undo/Redo (Ctrl-Z, Ctrl-E), cut/copy/paste, multi-line selection.
- Search (single-line and multi-line), mass replacement, switching to a line, commenting on a block/line.
- Search normalization: whitespace-insensitive matching (tabs, multiple spaces, trailing whitespace).
- Project-wide search across all canvases with `/all` modifier.
- Multiple "canvases" (working buffers) within the same session (up to 100).
- Support for working with GitHub projects: ZIP cloning, structure overview, commit/push (Ctrl-P).
- Translation of text or selected code into any language with replacement (Ctrl-W).
- Code autocompletion (context of 30 lines before the cursor and 10 after) is triggered by pressing Tab after a word.
- Popup hint with documentation for the function (triggered on typing `(` after function name).
- Project Overview with categorized files: Source, Text/Data, Database, Empty, Configuration, Documentation.
- File selection in Project Overview (Tab key) for targeted LLM processing.
- Mouse support: click positioning, drag selection, scroll wheel, right-click context actions.
- Structure panel (minimap/scrollbar) and line numbers toggle (Ctrl-D).
- Bracket matching and highlighting (Tab when panels enabled, right-click to jump to matching bracket).
- Stream mode (`-s`): process LLM queries via stdin/stdout without launching the editor.
- Input files/directories as RAG context for LLM (`-i` flag).
- Clipboard data as additional LLM input (`-d` flag in stream mode, Ctrl+C in prompt mode).
- Exit manager with per-canvas save prompts (y/n/a/c).

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

## Command-line flags

| Flag | Description |
|------|-------------|
| `-s`, `--stream` | Stream mode: read from stdin, write to stdout |
| `-d`, `--data` | Use clipboard data as input in stream mode |
| `-i`, `--input <path>` | Use file or directory content as input for LLM |
| `-a`, `--auto` | Enable AI-powered auto-complete mode |
| `-v`, `--version` | Show version |
| `-h`, `--help` | Show help |

## Keyboard shortcuts

| Key | Action |
|--------|------------------------------------------------------------------|
| Ctrl-L | Enter an LLM query / auto-generate code based on a comment |
| Ctrl-P | Send the project to GitHub / In LLM prompt: send with project context |
| Ctrl-R | Run the code / in case of error — recommendations from LLM |
| Ctrl-S | Save File |
| Ctrl-O | Open the file / Project file |
| Ctrl-N | New file / canvas |
| Ctrl-Q | Exit (with canvas save check) |
| Ctrl-F | Find text (supports `/all` for project-wide search) |
| Ctrl-G | Go to the line |
| Ctrl-Z | Undo |
| Ctrl-E | Redo |
| Ctrl-X | Cut a line / block |
| Ctrl-C | Copy line / block / In LLM prompt: send with clipboard data |
| Ctrl-V | Paste from Buffer |
| Ctrl-T | Open the OS terminal (output inserted into editor) |
| Ctrl-K | Comment / uncomment a line or selection |
| Ctrl-W | Translate text or highlighted block, replace in document |
| Ctrl-B | Switch canvas / In project mode: return to Project Overview |
| Ctrl-A | Select All |
| Ctrl-Y | Indent selected lines to the left |
| Ctrl-U | Indent selected lines to the right |
| Ctrl-J | Help |
| Ctrl-D | Toggle line numbers + structure panel |
| Tab | Autocomplete / In Project Overview: select file / Bracket highlight (with panels on) |
| Shift+←↑→↓ | Text selection |
| Home/End, PgUp/PgDn | Text navigation |
| Mouse wheel | Scroll up/down |
| Left click + drag | Select text |
| Right click | Context action (copy/jump to bracket/show doc/select line) |
| Esc | End selection / Close panels / Clear search |

## Project Overview Navigation

When opening a directory or GitHub project, the editor shows a categorized Project Overview:

| Key | Action |
|--------|------------------------------------------------------------------|
| ↑ / ↓ | Navigate between files |
| Enter | Open the selected file |
| Tab | Toggle file selection (for LLM processing) |
| Esc | Clear file selection |
| Ctrl-B | Switch to next canvas / Return to overview |
| Ctrl-O | Open file by name |
| Ctrl-N | New file |

Selected files can be sent to LLM using Ctrl-L → type instruction → Ctrl-P.

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
```
- Using a directory or file as RAG for LLM
```
./editor -i ./src ollama gemma3:4b
```

- Translation of the selected text (by default, into the system language):

  Ctrl-W → enter the language code → Enter

- Code launch and analysis:
  
  Ctrl-R  The LLM response will automatically substitute the recommendations or the results of the implementation.

## Supported Languages for Code Execution (Ctrl-R)

| Language | Compiler/Interpreter | Extension |
|----------|---------------------|-----------|
| C | gcc | .c, .h |
| C++ | g++ | .cpp, .cc, .cxx, .hpp, .hh |
| Assembly | nasm + ld | .s, .asm |
| Fortran | gfortran | .f, .for, .f90, .f95, .f03 |
| Go | go run | .go |
| Python | python3 | .py |
| Ruby | ruby | .rb |
| Kotlin | kotlinc + java | .kt, .kts |
| Swift | swiftc | .swift |
| HTML | browser (open) | .html, .htm |
| Lisp | sbcl | .lisp, .lsp, .cl, .el |
| PlantUML | java + plantuml.jar | .uml |

## Project File Categories

When opening a directory, files are categorized in the Project Overview:

- **Source files**: .c, .h, .cpp, .cc, .go, .py, .rb, .kt, .swift, .html, .lisp, .asm, .f90, etc.
- **Text and data files**: .txt, .log, .csv, .json, .md, .xml, .yaml, .ini, .cfg, .env, .rtf, .doc, .pdf, etc.
- **Database files**: .sql, .db, .sqlite, .sqlite3, .mdb, .accdb, .dbf, .jsonl, .parquet, .avro, .orc, etc.
- **Configuration files**: Makefile, Dockerfile, go.mod, package.json, requirements.txt, Cargo.toml, pom.xml, CMakeLists.txt, docker-compose.yml, etc.
- **Documentation**: README, LICENSE, COPYING, CREDITS, CHANGELOG, etc.

## LLM Integration

### Supported Providers
| Provider | Default Model | Notes |
|----------|--------------|-------|
| ollama | gemma3:4b | Local models |
| pollinations | qwen3:1.7b | Free API |
| openrouter | — | Requires API key |
| llm7 | — | Free API |
| Custom URL | — | Any OpenAI-compatible endpoint |

### LLM Features
- **Code generation** (Ctrl-L): Generate code from natural language description
- **Comment-based generation** (Ctrl-L with empty prompt): Reads comment above cursor
- **Code analysis and fix** (Ctrl-R): Runs code, on error asks LLM for fix suggestions
- **Translation** (Ctrl-W): Translate text/selection to any language
- **Project context** (Ctrl-L → Ctrl-P): Send all or selected project files as context
- **Clipboard context** (Ctrl-L → Ctrl-C): Include clipboard data in LLM request
- **Auto-complete** (`-a` flag): AI-powered code completion with ghost text
- **Stream mode** (`-s`): Use LLM via pipes without editor UI

### Environment Variables
| Variable | Description |
|----------|-------------|
| `LLM_PROVIDER` | Default LLM provider |
| `LLM_MODEL` | Default LLM model |

## GitHub Integration

- Open a GitHub project: `./editor https://github.com/owner/repo [token]`
- Downloads repository as ZIP archive
- Creates Project Overview with all files
- Push changes back: Ctrl-P → enter commit message
- Supports per-file push and full project push
- Uses `git init` / `git add` / `git commit` / `git push` workflow
- Force push fallback on rejection
- Token authentication via `ghp_` or `github_pat_` prefixed tokens

> **ATTENTION:** When connecting to your project, overwriting files on a local PC from the GitHub server deletes the data on the server. It is NECESSARY to resend all files to the server via Ctrl+P. When you open a project without overwriting it, the data remains on the server.

## Bracket Matching and Highlighting

- Auto-closing brackets: typing `(`, `[`, `{`, `"`, `'` automatically inserts the closing pair
- Bracket pair highlighting: Press Tab when both line numbers and structure panel are enabled (Ctrl-D)
- Jump to matching bracket: Right-click on a bracket to jump to its pair
- Visual indication: bracket lines highlighted in line number panel

## Mouse Support

| Action | Effect |
|--------|--------|
| Left click | Position cursor |
| Left click + drag | Select text |
| Scroll wheel | Scroll up/down (3 lines per step) |
| Right click (on selection) | Copy selected text to clipboard |
| Right click (on bracket) | Jump to matching bracket |
| Right click (on function name) | Show function documentation |
| Right click (elsewhere) | Select entire line |

## License

The project is distributed under the **BSD-3-Clause** license. The details are in the file. LICENSE.txt .

## Participation in development

We welcome your edits, fixes and ideas!  
Create a pull request or issue in the repository.

## Dependencies

- [gdamore/tcell/v2](https://github.com/gdamore/tcell) — Terminal cell-based screen library (Apache-2.0)
- [atotto/clipboard](https://github.com/atotto/clipboard) — Clipboard support (BSD-3-Clause)
- [mattn/go-runewidth](https://github.com/mattn/go-runewidth) — Unicode character width calculation (MIT)
- [SkalaSkalolaz/llmclient](https://github.com/SkalaSkalolaz/llmclient) — Unified LLM client library

---

Thanks for using Code Editor GO!  
We wish you productive and comfortable work.