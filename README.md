# editor
# Text Editor

[![License](https://img.shields.io/badge/license-BSD%203--Clause-blue.svg)](LICENSE)
[![Go version](https://img.shields.io/badge/go-go1.24.6-brightgreen?style=flat-square&logo=go&logoColor=white)](
https://golang.org/dl/)

This is a terminal-based text editor written in Go, utilizing the `tcell/v2` library for terminal interaction. It supports basic editing functions, syntax highlighting for numerous programming languages, and integration with LLM (Large Language Models) via an external program called `tgpt`.

## Features

*   **Text Editing:** Multiline editing, navigation with keys (arrow keys, Home, End, PgUp, PgDn), insertion, deletion, file creation, and opening.
*   **Syntax Highlighting:** Supports syntax highlighting for the following languages: C, C++, Assembly, Fortran, Go, Python, Ruby, Kotlin, Swift, HTML, Lisp.
*   **Search:** Text search within a file.
*   **Navigation:** Jump to a specific line.
*   **Undo/Redo:** Supports undo and redo operations.
*   **Clipboard:** Cut, copy, and paste (uses system clipboard via `atotto/clipboard` library).
*   **LLM Integration:** Allows sending instructions and text from the editor to the external `tgpt` program for interacting with LLMs. Buffer contents and visible text can be automatically included in requests.
*   **Status Bar:** Displays filename, language, cursor position (line and column), and hotkey hints.

## Installation

1.  Ensure you have Go installed (https://golang.org/dl/).
2.  Install necessary dependencies:
    ```bash
    go mod init <module_name> # If creating a new module
    go get github.com/atotto/clipboard
    go get github.com/gdamore/tcell/v2
    go get github.com/mattn/go-runewidth
    ```
    (Or simply run `go mod tidy` if `go.mod` already exists).
3.  Compile the program:
    ```bash
    go build -o editor main.go # Replace main.go with the path to your code file if different
    ```
4.  To enable LLM functionality, install and configure the external `tgpt` program (or modify the code to use a different program as described in the comments).

## Usage

Run the compiled file:

```bash
./editor  [flags] [file_path]
```

### Flags

*   `-provider string`: LLM provider (defaults to value from environment variable `LLM_PROVIDER`).
*   `-model string`: LLM model (defaults to value from environment variable `LLM_MODEL`).
*   `-path string`: Path to open a specific file.
*   `-v, -version`: Show editor version.
*   `-h, --help`: Show extended help.

If no file path is specified in the flags, it can be passed as the first positional argument.

### Hotkeys

- Arrow keys: move cursor
- Home/End, PgUp/PgDn: text navigation
- Ctrl-A: select all (and other selection options)
- Ctrl-F: search text
- Ctrl-G: go to line
- Ctrl-S: save file
- Ctrl-O: open file
- Ctrl-N: new file
- Ctrl-Q: quit
- Ctrl-X: cut current line
- Ctrl-C: copy to clipboard
- Ctrl-V: paste from clipboard
- Ctrl-P: generate text/code based on description
- Ctrl-L: send request to LLM (and insert response)


## Version

Current version: 1.2.0

## Dependencies

*   `github.com/atotto/clipboard`
*   `github.com/gdamore/tcell/v2`
*   `github.com/mattn/go-runewidth`

## License

This project is licensed under the BSD 3-Clause License — see details in the [LICENSE](LICENSE) file.

## Notes

*   The code contains comments in Russian.
*   It uses the external command `tgpt` for LLM interaction (install via `brew install tgpt`, https://github.com/aandrew-me/tgpt). If unavailable, you can try replacing it with another.
*   The editor has a fixed window size (115x34), defined in the code (`contentWidth`, `contentHeight`).
*   The program has been tested on macOS 15.6.

## Contact Information

For questions or suggestions, contact: [skala.skalolaz.1970@gmail.com]

See [CREDITS.md](CREDITS.md) — acknowledgements and dependency information.


# editor
# Текстовый редактор

Это текстовый редактор для работы в терминале, написанный на языке Go, с использованием библиотеки `tcell/v2` для работы с терминалом. Он поддерживает основные функции редактирования, синтаксическую подсветку для множества языков программирования и интеграцию с LLM (Large Language Models) через внешнюю программу `tgpt`.

## Возможности

*   **Редактирование текста:** Многострочное редактирование, навигация с помощью клавиш (стрелки, Home, End, PgUp, PgDn), вставка, удаление, создание и открытие файлов.
*   **Синтаксическая подсветка:** Поддержка подсветки синтаксиса для следующих языков: C, C++, Assembly, Fortran, Go, Python, Ruby, Kotlin, Swift, HTML, Lisp.
*   **Поиск:** Поиск текста по файлу.
*   **Навигация:** Переход к определенной строке.
*   **Отмена/повтор:** Поддержка операций отмены (`Undo`) и повтора (`Redo`) действий.
*   **Буфер обмена:** Вырезание, копирование и вставка текста (используется системный буфер обмена через библиотеку `atotto/clipboard`).
*   **Интеграция с LLM:** Возможность отправки инструкций и текста из редактора во внешнюю программу `tgpt` для взаимодействия с LLM. Данные из буфера обмена и видимая часть текста могут автоматически добавляться к запросу.
*   **Статусная строка:** Отображение имени файла, языка, номера строки и столбца, а также подсказок по горячим клавишам.

## Установка

1.  Убедитесь, что у вас установлен Go (https://golang.org/dl/).
2.  Установите необходимые зависимости:
    ```bash
    go mod init <название_модуля> # Если вы создаете новый модуль
    go get github.com/atotto/clipboard
    go get github.com/gdamore/tcell/v2
    go get github.com/mattn/go-runewidth
    ```
    (Или просто используйте `go mod tidy` если `go.mod` уже существует).
3.  Скомпилируйте программу:
    ```bash
    go build -o editor main.go # Замените main.go на путь к файлу с кодом, если он другой
    ```
4.  Для работы с LLM необходимо установить и настроить внешнюю программу `tgpt` (или изменить код для использования другой программы, как указано в комментариях).

## Использование

Запустите скомпилированный файл:

```bash
./editor  [флаги] [путь_к_файлу]
```

### Флаги

*   `-provider string`: Провайдер LLM (по умолчанию берется из переменной окружения `LLM_PROVIDER`).
*   `-model string`: Модель LLM (по умолчанию берется из переменной окружения `LLM_MODEL`).
*   `-path string`: Путь к файлу для открытия.
*   `-v, -version`: Показать версию редактора.
*   `-h, --help`: Показать расширенную справку.

Если путь к файлу не указан в флагах, его можно передать как первый аргумент командной строки.

### Горячие клавиши

- Стрелки: перемещение курсора
- Home/End, PgUp/PgDn: навигация по тексту
- Ctrl-A: выделение всего (и другие варианты выделения)
- Ctrl-F: поиск текста
- Ctrl-G: перейти к строке
- Ctrl-S: сохранить файл
- Ctrl-O: открыть файл
- Ctrl-N: новый файл
- Ctrl-Q: выход
- Ctrl-X: вырезать текущую строку
- Ctrl-C: копировать в буфер обмена
- Ctrl-V: вставить из буфера обмена
- Ctrl-P: сгенерировать текст/код на основе описания
- Ctrl-L: отправить запрос к LLM (и вставить ответ)


## Версия

Текущая версия: 1.2.0

## Зависимости

*   `github.com/atotto/clipboard`
*   `github.com/gdamore/tcell/v2`
*   `github.com/mattn/go-runewidth`

## Лицензия

Этот проект лицензирован по лицензии BSD 3-Clause - подробности см. в файле [LICENSE](LICENSE).

## Примечания

*   Код содержит комментарии на русском языке.
*   Для работы с LLM используется внешняя команда `tgpt` (brew install tgpt) https://github.com/aandrew-me/tgpt. Если она недоступна, можно попробовать заменить её на иную.
*   Редактор имеет фиксированный размер окна (115x34), определяемый в коде (`contentWidth`, `contentHeight`).
*   Проверка кода программы была произведена на ОС macOS 15.6 

## Контактная информация

Если есть вопросы или предложения обращайтесь по адресу: [skala.skalolaz.1970@gmail.com]

Смотрите [CREDITS.md](CREDITS.md) — благодарности и информация о зависимостях.
