# editor
# Текстовый редактор

[![License](https://img.shields.io/badge/license-BSD%203--Clause-blue.svg)](LICENSE)

Это текстовый редактор для работы в терминале, написанный на языке Go, с использованием библиотеки `tcell/v2` для работы с терминалом. Он поддерживает основные функции редактирования, синтаксическую подсветку для множества языков программирования и интеграцию с LLM (Large Language Models) через внешнюю программу `cogitor`.

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
./editor [путь_к_файлу] [флаги]
```

### Флаги

*   `-provider string`: Провайдер LLM (по умолчанию берется из переменной окружения `LLM_PROVIDER`).
*   `-model string`: Модель LLM (по умолчанию берется из переменной окружения `LLM_MODEL`).
*   `-path string`: Путь к файлу для открытия.
*   `-v, -version`: Показать версию редактора.
*   `-h, --help`: Показать расширенную справку.

Если путь к файлу не указан в флагах, его можно передать как первый аргумент командной строки.

### Горячие клавиши

*   **Ctrl-S:** Сохранить файл.
*   **Ctrl-Q:** Выход из редактора (с запросом на сохранение, если файл изменен).
*   **Ctrl-F:** Поиск текста.
*   **Ctrl-G:** Перейти к строке.
*   **Ctrl-U:** Отменить (Undo).
*   **Ctrl-Y:** Повторить (Redo).
*   **Ctrl-K:** Вырезать текущую строку.
*   **Ctrl-O:** Открыть файл.
*   **Ctrl-N:** Создать новый файл.
*   **Ctrl-L:** Отправить инструкцию LLM (через `tgpt`).
*   **Стрелки:** Перемещение курсора.
*   **Home/End:** Перемещение в начало/конец строки.
*   **PgUp/PgDn:** Перемещение на страницу вверх/вниз.
*   **Enter:** Вставить новую строку.
*   **Backspace:** Удалить символ перед курсором.

## Версия

Текущая версия: 1.1.2

## Зависимости

*   `github.com/atotto/clipboard`
*   `github.com/gdamore/tcell/v2`
*   `github.com/mattn/go-runewidth`

## Лицензия

Этот проект лицензирован по лицензии BSD 3-Clause - подробности см. в файле [LICENSE](LICENSE).

## Примечания

*   Код содержит комментарии на русском языке.
*   Для работы с LLM используется внешняя команда `tgpt` (brew install tgpt) https://github.com/aandrew-me/tgpt. Если она недоступна, можно попробовать заменить её на иную (см. комментарии в функции `llmQuery`).
*   Редактор имеет фиксированный размер окна (115x34), определяемый в коде (`contentWidth`, `contentHeight`).
*   Проверка кода программы была произведена на ОС macOS 15.6 

## Контактная информация

Если есть вопросы или предложения обращайтесь по адресу: [skala.skalolaz.1970@gmail.com]

Смотрите [CREDITS.md](CREDITS.md) — благодарности и информация о зависимостях.

# Editor
# Text Editor

[![License](https://img.shields.io/badge/license-BSD%203--Clause-blue.svg)](LICENSE)

This is a terminal-based text editor written in Go, utilizing the `tcell/v2` library for terminal interactions. It supports basic editing functions, syntax highlighting for many programming languages, and integration with LLMs (Large Language Models) via an external program `cogitor`.

## Features

*   **Text Editing:** Multiline editing, navigation via keys (arrows, Home, End, PgUp, PgDn), inserting, deleting, creating, and opening files.
*   **Syntax Highlighting:** Supports syntax highlighting for the following languages: C, C++, Assembly, Fortran, Go, Python, Ruby, Kotlin, Swift, HTML, Lisp.
*   **Search:** Search through the file text.
*   **Navigation:** Jump to a specific line.
*   **Undo/Redo:** Support for undo (`Undo`) and redo (`Redo`) operations.
*   **Clipboard:** Cut, copy, and paste text (uses system clipboard via `atotto/clipboard` library).
*   **LLM Integration:** Ability to send instructions and text from the editor to an external `tgpt` program for interaction with LLMs. Data from the clipboard and the visible part of the text can be automatically added to the request.
*   **Status Bar:** Displays file name, language, line and column number, as well as hotkey hints.

## Installation

1.  Ensure Go is installed (https://golang.org/dl/).
2.  Install necessary dependencies:
    ```bash
    go mod init <module_name> # If creating a new module
    go get github.com/atotto/clipboard
    go get github.com/gdamore/tcell/v2
    go get github.com/mattn/go-runewidth
    ```
    (Or just use `go mod tidy` if `go.mod` already exists).
3.  Compile the program:
    ```bash
    go build -o editor main.go # Replace main.go with your code file if different
    ```
4.  To work with LLMs, install and configure the external `tgpt` program (or modify the code to use another program as noted in comments).

## Usage

Run the compiled file:

```bash
./editor [file_path] [flags]
```

### Flags

*   `-provider string`: LLM provider (defaults from environment variable `LLM_PROVIDER`).
*   `-model string`: LLM model (defaults from environment variable `LLM_MODEL`).
*   `-path string`: Path to the file to open.
*   `-v, -version`: Show editor version.
*   `-h, --help`: Show extended help.

If the file path isn't specified in flags, it can be provided as the first command-line argument.

### Hotkeys

*   **Ctrl-S:** Save the file.
*   **Ctrl-Q:** Exit the editor (prompt to save if the file has been modified).
*   **Ctrl-F:** Search text.
*   **Ctrl-G:** Go to line.
*   **Ctrl-U:** Undo.
*   **Ctrl-Y:** Redo.
*   **Ctrl-K:** Cut the current line.
*   **Ctrl-O:** Open file.
*   **Ctrl-N:** Create a new file.
*   **Ctrl-L:** Send instruction to LLM (via `tgpt`).
*   **Arrow keys:** Move cursor.
*   **Home/End:** Move to start/end of line.
*   **PgUp/PgDn:** Scroll up/down a page.
*   **Enter:** Insert new line.
*   **Backspace:** Delete character before cursor.

## Version

Current version: 1.1.2

## Dependencies

*   `github.com/atotto/clipboard`
*   `github.com/gdamore/tcell/v2`
*   `github.com/mattn/go-runewidth`

## License

This project is licensed under the BSD 3-Clause License — see the [LICENSE](LICENSE) file for details.

## Notes

*   The code contains comments in Russian.
*   The LLM interaction uses an external command `tgpt` (install via `brew install tgpt`) https://github.com/aandrew-me/tgpt. If unavailable, it can be replaced with another as noted in the `llmQuery` function comments.
*   The editor window size is fixed (115x34), defined in the code (`contentWidth`, `contentHeight`).
*   The code has been tested on macOS 15.6.

## Contact Information

For questions or suggestions, contact: [skala.skalolaz.1970@gmail.com]

See also [CREDITS.md](CREDITS.md) — acknowledgments and information on dependencies.
