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
*   **Интеграция с LLM:** Возможность отправки инструкций и текста из редактора во внешнюю программу `cogitor` для взаимодействия с LLM. Данные из буфера обмена и видимая часть текста могут автоматически добавляться к запросу.
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
4.  Для работы с LLM необходимо установить и настроить внешнюю программу `cogitor` (или изменить код для использования другой программы, например, `tgpt`, как указано в комментариях).

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
*   **Ctrl-L:** Отправить инструкцию LLM (через `cogitor`).
*   **Стрелки:** Перемещение курсора.
*   **Home/End:** Перемещение в начало/конец строки.
*   **PgUp/PgDn:** Перемещение на страницу вверх/вниз.
*   **Enter:** Вставить новую строку.
*   **Backspace:** Удалить символ перед курсором.

## Версия

Текущая версия: 1.1.1

## Зависимости

*   `github.com/atotto/clipboard`
*   `github.com/gdamore/tcell/v2`
*   `github.com/mattn/go-runewidth`

## Лицензия

Этот проект лицензирован по лицензии BSD 3-Clause - подробности см. в файле [LICENSE](LICENSE).

## Примечания

*   Код содержит комментарии на русском языке.
*   Для работы с LLM используется внешняя команда `cogitor`. Если она недоступна, можно попробовать заменить её на `tgpt` (см. комментарии в функции `llmQuery`).
*   Редактор имеет фиксированный размер окна (115x34), определяемый в коде (`contentWidth`, `contentHeight`).
*   Проверка кода программы была произведена на ОС macOS 15.6 

## Контактная информация

Если есть вопросы или предложения обращайтесь по адресу: [skala.skalolaz.1970@gmail.com]

Смотрите [CREDITS.md](CREDITS.md) — благодарности и информация о зависимостях.

# Text editor

[![License](https://img.shields.io/badge/license-BSD%203--Clause-blue.svg)](LICENSE)

This is a terminal text editor written in Go, utilizing the `tcell/v2` library for terminal interaction. It supports basic editing features, syntax highlighting for numerous programming languages, and integration with LLM (Large Language Models) via an external program `cogitor`.

## Features

*   **Text editing:** Multi-line editing, navigation with keys (arrows, Home, End, PgUp, PgDn), insertion, deletion, file creation and opening.
*   **Syntax highlighting:** Supports syntax highlighting for the following languages: C, C++, Assembly, Fortran, Go, Python, Ruby, Kotlin, Swift, HTML, Lisp.
*   **Search:** Search text within a file.
*   **Navigation:** Jump to a specific line.
*   **Undo/Redo:** Support for undo (`Undo`) and redo (`Redo`) operations.
*   **Clipboard:** Cut, copy, and paste text (using the system clipboard via the `atotto/clipboard` library).
*   **LLM integration:** Ability to send instructions and text from the editor to the external `cogitor` program for interaction with LLM. Buffer contents and visible text may be automatically included in the request.
*   **Status line:** Shows filename, language, line and column numbers, and hotkey hints.

## Installation

1.  Ensure you have Go installed (https://golang.org/dl/).
2.  Install necessary dependencies:
    ```bash
    go mod init <module_name> # If creating a new module
    go get github.com/atotto/clipboard
    go get github.com/gdamore/tcell/v2
    go get github.com/mattn/go-runewidth
    ```
    (Or just run `go mod tidy` if `go.mod` already exists).
3.  Compile the program:
    ```bash
    go build -o editor main.go # Replace main.go with the path to your code file if different
    ```
4.  For LLM interaction, install and configure the external `cogitor` program (or modify the code to use another program like `tgpt`, as mentioned in comments).

## Usage

Run the compiled file:

```bash
./editor [file_path] [flags]
```

### Flags

*   `-provider string`: LLM provider (defaults to environment variable `LLM_PROVIDER`).
*   `-model string`: LLM model (defaults to environment variable `LLM_MODEL`).
*   `-path string`: Path to the file to open.
*   `-v, -version`: Show editor version.
*   `-h, --help`: Show extended help.

If the file path isn't specified via flags, it can be passed as the first command-line argument.

### Hotkeys

*   **Ctrl-S:** Save file.
*   **Ctrl-Q:** Exit editor (with save prompt if file is modified).
*   **Ctrl-F:** Search text.
*   **Ctrl-G:** Go to line.
*   **Ctrl-U:** Undo.
*   **Ctrl-Y:** Redo.
*   **Ctrl-K:** Cut current line.
*   **Ctrl-O:** Open file.
*   **Ctrl-N:** Create new file.
*   **Ctrl-L:** Send LLM instruction (via `cogitor`).
*   **Arrows:** Move cursor.
*   **Home/End:** Move to start/end of line.
*   **PgUp/PgDn:** Scroll up/down.
*   **Enter:** Insert new line.
*   **Backspace:** Delete character before cursor.

## Version

Current version: 1.1.1

## Dependencies

*   `github.com/atotto/clipboard`
*   `github.com/gdamore/tcell/v2`
*   `github.com/mattn/go-runewidth`

## License

This project is licensed under the BSD 3-Clause License - see the [LICENSE](LICENSE) file for details.

## Notes

*   The code contains comments in Russian.
*   LLM interaction is handled via the external `cogitor` command. If unavailable, it can be replaced with `tgpt` (see comments in `llmQuery` function).
*   The editor window size is fixed (115x34), specified in the code (`contentWidth`, `contentHeight`).
*   The program code check was performed on macOS 15.6.

## Contact information

If you have any questions or suggestions, please contact us at: [skala.skaloalz.1970@gmail.com].

Look [CREDITS.md](CREDITS.md) — thanks and information about dependencies.
