package main

import (
	"fmt"
)


// printUsageExtended prints the extended help information based on OS language
func printUsageExtended() {
	lang := detectSystemLanguage()
	switch lang {
	case "ru":
		printUsageRU()
	default:
		printUsageEN()
	}
}

// printUsageExtended prints the extended help information based on OS language
func printUsageMini() {
	lang := detectSystemLanguage()
	switch lang {
	case "ru":
		printUsageRUMini()
	default:
		printUsageENMini()
	}
}

// printUsageExtended prints the extended help information.
// printUsageExtended выводит расширенную справку.
func printUsageRU() {
	fmt.Println("Editor - расширенная справка")
	fmt.Println("Usage: editor  [provider]/[URL provider] [model] [path]/[URL GitHub project] [sn-...] [ghp_]")
	fmt.Println()
	fmt.Println("provider {default: ollama}, model {default: qwen3:1.7b}")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help         Показать эту справку и использование.")
	fmt.Println("  -v, --version      Показать версию программы.")
	fmt.Println()
	fmt.Println("Особенности:")
	fmt.Println("  - Текстовый редактор с поддержкой многострочного редактирования, курсорной навигации,")
	fmt.Println("    отмены/повтора (undo/redo), вырезания/копирования/вставки, поиска, перехода к строке,")
	fmt.Println("    и опциональной интеграции с LLM.")
	fmt.Println("  - Интеграция LLM: вызывается при настройке provider/model.")
	fmt.Println()
	fmt.Println("Горячие клавиши:")
	fmt.Println("  Ctrl-L  Ввести указание для LLM или генерировать код (по коментарию вверху) если указание пустое")
	fmt.Println("  Ctrl-R  Запускает код программы, при ошибке в коде - ")
	fmt.Println("          Поддерживаемые языки: c, cpp, assembly, fortran, go, \n          python, ruby, kotlin, swift, html, lisp")
	fmt.Println("  Ctrl-S  Сохранить файл")
	fmt.Println("  Ctrl-O  Открыть файл")
	fmt.Println("  Ctrl-N  Новый файл")
	fmt.Println("  Ctrl-Q  Выход из редактора")
	fmt.Println("  Ctrl-F  Поиск текста. Для замены текста используй символ -> .\n          Пример: Print -> Printf")
	fmt.Println("  Ctrl-G  Перейти к строке")
	fmt.Println("  Ctrl-Z  Отменить")
	fmt.Println("  Ctrl-E  Вернуть отменённое")
	fmt.Println("  Ctrl-X  Убрать текущую строку")
	fmt.Println("  Ctrl-A  Выделить все")
	fmt.Println("  Ctrl-B  Сдвиг канваса (листание файлов)")
	fmt.Println("  Ctrl-C  Копировать в буфер обмена")
	fmt.Println("  Ctrl-V  Вставить буфер обмена")
	fmt.Println("  Ctrl-T  Терминал ОС (печать ответа в canvas)")
	fmt.Println("  Ctrl-K  Выставить символ коментария для строки или выделеных строк,\n          убрать символ коментария")
	fmt.Println("  Ctrl-W  Перевод строки или выделеного текста на требуемый иностранный язык.\n          После перевода осуществляется замена. По умолчанию, язык локали.")
	fmt.Println("  Ctrl-Y  Сдвиг строк выделенного кода влево на 4 знака")
	fmt.Println("  Ctrl-U  Сдвиг строк выделенного кода вправо на 4 знака")
	fmt.Println("  Ctrl-P  Отправка проекта на GitHub / Дополнительная клавиша для\n          отправки всех файлов проекта, как данных для LLM")

	fmt.Println("Навигация:")
	fmt.Println("  Стрелки: перемещение курсора, Home/End, PgUp/PgDn — навигация по тексту")
	fmt.Println()
	fmt.Println("Примеры:")
	fmt.Println("  editor pollinations openai /path/to/file.txt")
	fmt.Println("  editor pollinations openai /path/to")
	fmt.Println("  editor llm7 help")
	fmt.Println("  editor pollinations help")
	fmt.Println("  editor https://openai.ai/api/v1/chat/completions gpt-4.1-nano file.txt sn-...")
	fmt.Println("  editor openrouter deepseek/deepseek-r1:free file.txt sn-...")
	fmt.Println("  editor file.txt")
}

func printUsageEN() {
	fmt.Println("Editor - extended help")
	fmt.Println("Usage: editor  [provider]/[URL provider] [model] [path]/[URL GitHub project] [sn-...] [ghp_]")
	fmt.Println()
	fmt.Println("provider {default: ollama}, model {default: qwen3:1.7b}")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help         Show this help and usage.")
	fmt.Println("  -v, --version      Show program version.")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  - Text editor with support for multiline editing, cursor navigation,")
	fmt.Println("    undo/redo, cut/copy/paste, find, go to line,")
	fmt.Println("    and optional integration with LLM.")
	fmt.Println("  - LLM integration: invoked during provider/model setup.")
	fmt.Println()
	fmt.Println("Hotkeys:")
	fmt.Println("  Ctrl-L  Enter prompt for LLM or generate code if prompt is empty")
	fmt.Println("  Ctrl-R  Run code, and on error - recommendations to fix")
	fmt.Println("  Supported languages: c, cpp, assembly, fortran, go, \n          python, ruby, kotlin, swift, html, lisp.")
	fmt.Println("  Ctrl-S  Save file")
	fmt.Println("  Ctrl-O  Open file")
	fmt.Println("  Ctrl-N  New file")
	fmt.Println("  Ctrl-Q  Quit editor")
	fmt.Println("  Ctrl-F  Find text. To replace the text, use the symbol -> .\n          Example: Print -> Printf.")
	fmt.Println("  Ctrl-G  Go to line")
	fmt.Println("  Ctrl-Z  Undo")
	fmt.Println("  Ctrl-E  Redo")
	fmt.Println("  Ctrl-X  Remove current line")
	fmt.Println("  Ctrl-A  Select all")
	fmt.Println("  Ctrl-B  Shift of the canvas (scrolling files)")
	fmt.Println("  Ctrl-C  Copy to clipboard")
	fmt.Println("  Ctrl-V  Paste clipboard")
	fmt.Println("  Ctrl-T  OS terminal (print LLM answer on canvas)")
	fmt.Println("  Ctrl-K  Set a comment symbol for the line or selected lines,\n            remove the comment symbol.")
	fmt.Println("  Ctrl-Y  Shift the selected code lines to the left by 4 characters")
	fmt.Println("  Ctrl-U  Shift the selected code lines to the right by 4 characters")
	fmt.Println("  Ctrl-W  Translating a line or selected text into the required foreign language.\n          After translation, replacement is carried out. By default, the locale language.")
	fmt.Println("  Ctrl-P  Sending the project to GitHub / Additional key for\n            sending all project files as LLM data")
	fmt.Println("Navigation:")
	fmt.Println("  Arrows: cursor movement, Home/End, PgUp/PgDn — navigation in text")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  editor pollinations openai /path/to/file.txt")
	fmt.Println("  editor pollinations openai /path/to")
	fmt.Println("  editor llm7 help")
	fmt.Println("  editor pollinations help")
	fmt.Println("  editor https://openai.ai/api/v1/chat/completions gpt-4.1-nano file.txt sn-...")
	fmt.Println("  editor openrouter deepseek/deepseek-r1:free file.txt sn-...")
	fmt.Println("  editor file.txt")
}

func printUsageRUMini() {
	fmt.Println("  ")
	fmt.Println("     Ctrl-L  Ввести указание для LLM или генерировать код (по коментарию вверху) если указание пустое")
	fmt.Println("     Ctrl-R  Запускает код программы, при ошибке в коде - рекомендации по их исправлению")
	fmt.Println("     Ctrl-S  Сохранить файл")
	fmt.Println("     Ctrl-O  Открыть файл")
	fmt.Println("     Ctrl-N  Новый файл")
	fmt.Println("     Ctrl-Q  Выход из редактора")
	fmt.Println("     Ctrl-F  Поиск текста. Для замены текста используй символ -> . Пример: Print -> Printf")
	fmt.Println("     Ctrl-G  Перейти к строке")
	fmt.Println("     Ctrl-Z  Отменить")
	fmt.Println("     Ctrl-E  Вернуть отменённое")
	fmt.Println("     Ctrl-X  Убрать текущую строку")
	fmt.Println("     Ctrl-A  Выделить все")
	fmt.Println("     Ctrl-B  Сдвиг канваса (листание файлов)")
	fmt.Println("     Ctrl-C  Копировать в буфер обмена")
	fmt.Println("     Ctrl-V  Вставить буфер обмена")
	fmt.Println("     Ctrl-T  Терминал ОС (печать ответа в canvas)")
	fmt.Println("     Ctrl-K  Выставить символ коментария для строки или выделеных строк, убрать символ коментария")
	fmt.Println("     Ctrl-W  Перевод строки или выделеного текста на требуемый иностранный язык.")
	fmt.Println("     Ctrl-Y  Сдвиг строк выделенного кода влево на 4 знака")
	fmt.Println("     Ctrl-U  Сдвиг строк выделенного кода вправо на 4 знака")
	fmt.Println("     Ctrl-P  Отправка проекта на GitHub / Дополнительная клавиша для\n             отправки всех файлов проекта, как данных для LLM")
}

func printUsageENMini() {
	fmt.Println("   ")
	fmt.Println("  Ctrl-L  Enter prompt for LLM or generate code if prompt is empty")
	fmt.Println("  Ctrl-R  Run code, and on error - recommendations to fix")
	fmt.Println("  Ctrl-S  Save file")
	fmt.Println("  Ctrl-O  Open file")
	fmt.Println("  Ctrl-N  New file")
	fmt.Println("  Ctrl-Q  Quit editor")
	fmt.Println("  Ctrl-F  Find text. To replace the text, use the symbol -> . Example: Print -> Printf.")
	fmt.Println("  Ctrl-G  Go to line")
	fmt.Println("  Ctrl-Z  Undo")
	fmt.Println("  Ctrl-E  Redo")
	fmt.Println("  Ctrl-X  Remove current line")
	fmt.Println("  Ctrl-A  Select all")
	fmt.Println("  Ctrl-B  Select by line (from cursor)")
	fmt.Println("  Ctrl-C  Copy to clipboard")
	fmt.Println("  Ctrl-V  Paste clipboard")
	fmt.Println("  Ctrl-T  OS terminal (print LLM answer on canvas)")
	fmt.Println("  Ctrl-K  Set a comment symbol for the line or selected lines, remove the comment symbol.")
	fmt.Println("  Ctrl-Y  Shift the selected code lines to the left by 4 characters")
	fmt.Println("  Ctrl-U  Shift the selected code lines to the right by 4 characters")
	fmt.Println("  Ctrl-W  Translating a line or selected text into the required foreign language.")
	fmt.Println("  Ctrl-P  Sending the project to GitHub / Additional key for\n               sending all project files as LLM data")
}
