package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Language represents the programming language of the file.
// Language представляет язык программирования файла.
type Language string

// Supported languages.
// Поддерживаемые языки.
const (
	LangC        Language = "c"
	LangCpp      Language = "cpp"
	LangAssembly Language = "assembly"
	LangFortran  Language = "fortran"
	LangGo       Language = "go"
	LangPython   Language = "python"
	LangRuby     Language = "ruby"
	LangKotlin   Language = "kotlin"
	LangSwift    Language = "swift"
	LangHTML     Language = "html"
	LangLisp     Language = "lisp"
	LangUnknown  Language = "unknown"
)

// Language statement terminators
var languageTerminators = map[Language]string{
	LangC:        ";",
	LangCpp:      ";",
	LangAssembly: "",
	LangFortran:  "",
	LangGo:       "",
	LangPython:   ":",
	LangRuby:     "",
	LangKotlin:   "",
	LangSwift:    "",
	LangHTML:     "",
	LangLisp:     "",
}

// Keywords for each language
var languageKeywords = map[Language]map[string]bool{
	LangC: map[string]bool{
		"auto": true, "break": true, "case": true, "char": true, "const": true,
		"continue": true, "default": true, "do": true, "double": true, "else": true,
		"enum": true, "extern": true, "float": true, "for": true, "goto": true,
		"if": true, "int": true, "long": true, "register": true, "return": true,
		"short": true, "signed": true, "sizeof": true, "static": true, "struct": true,
		"switch": true, "typedef": true, "union": true, "unsigned": true, "void": true,
		"volatile": true, "while": true,
		"_Bool": true, "_Complex": true, "_Imaginary": true, "inline": true,
		"restrict": true,
		"_Alignas": true, "_Alignof": true, "_Atomic": true, "_Generic": true,
		"_Noreturn": true, "_Static_assert": true, "_Thread_local": true,
		"alignas": true, "alignof": true, "bool": true, "constexpr": true,
		"false": true, "nullptr": true, "static_assert": true, "thread_local": true,
		"true": true, "typeof": true, "typeof_unqual": true,
		"#define": true, "#elif": true, "#else": true, "#endif": true, "#error": true,
		"#if": true, "#ifdef": true, "#ifndef": true, "#include": true, "#line": true,
		"#pragma": true, "#undef": true,
		"__FILE__": true, "__LINE__": true, "__DATE__": true, "__TIME__": true,
		"__STDC__": true, "__STDC_VERSION__": true, "__STDC_HOSTED__": true,
		"__STDC_IEC_559__": true, "__STDC_IEC_559_COMPLEX__": true,
		"__STDC_ISO_10646__": true, "__func__": true,
		"__asm__": true, "__attribute__": true, "__builtin_": true, "__declspec": true,
		"__inline__": true, "__volatile__": true, "__restrict__": true,
		"size_t": true, "ptrdiff_t": true, "int8_t": true, "int16_t": true,
		"int32_t": true, "int64_t": true, "uint8_t": true, "uint16_t": true,
		"uint32_t": true, "uint64_t": true, "intptr_t": true, "uintptr_t": true,
		"intmax_t": true, "uintmax_t": true, "wchar_t": true, "char16_t": true,
		"char32_t": true,
		"NULL":     true, "EOF": true, "stdin": true, "stdout": true, "stderr": true,
		"printf": true, "scanf": true, "malloc": true, "free": true, "calloc": true,
		"realloc": true, "exit": true, "abort": true, "assert": true,
		"_Packed": true, "__packed": true, "__aligned": true, "__section": true,
		"_Nullable": true, "_Nonnull": true, "_Null_unspecified": true,
		"__complex__": true, "__imag__": true, "__real__": true,
	},
	LangCpp: map[string]bool{
		"alignas": true, "alignof": true, "and": true, "and_eq": true, "asm": true,
		"auto": true, "bitand": true, "bitor": true, "bool": true, "break": true,
		"case": true, "catch": true, "char": true, "char8_t": true, "char16_t": true,
		"char32_t": true, "class": true, "compl": true, "concept": true, "const": true,
		"const_cast": true, "consteval": true, "constexpr": true, "constinit": true,
		"continue": true, "co_await": true, "co_return": true, "co_yield": true,
		"decltype": true, "default": true, "delete": true, "do": true, "double": true,
		"dynamic_cast": true, "else": true, "enum": true, "explicit": true,
		"export": true, "extern": true, "false": true, "float": true, "for": true,
		"friend": true, "goto": true, "if": true, "inline": true, "int": true,
		"long": true, "mutable": true, "namespace": true, "new": true, "noexcept": true,
		"not": true, "not_eq": true, "nullptr": true, "operator": true, "or": true,
		"or_eq": true, "private": true, "protected": true, "public": true,
		"register": true, "reinterpret_cast": true, "requires": true, "return": true,
		"short": true, "signed": true, "sizeof": true, "static": true, "static_assert": true,
		"static_cast": true, "struct": true, "switch": true, "template": true,
		"this": true, "thread_local": true, "throw": true, "true": true, "try": true,
		"typedef": true, "typeid": true, "typename": true, "union": true, "unsigned": true,
		"using": true, "virtual": true, "void": true, "volatile": true, "wchar_t": true,
		"while": true, "xor": true, "xor_eq": true,
		"#define": true, "#elif": true, "#else": true, "#endif": true,
		"#error": true, "#if": true, "#ifdef": true, "#ifndef": true,
		"#include": true, "#line": true, "#pragma": true, "#undef": true,
		"#import": true, "#using": true,
		"__cplusplus": true, "__FILE__": true, "__LINE__": true, "__DATE__": true,
		"__TIME__": true, "__STDC_HOSTED__": true, "__STDC__": true,
		"__STDC_VERSION__": true, "__STDC_UTF_16__": true, "__STDC_UTF_32__": true,
		"[[noreturn]]": true, "[[carries_dependency]]": true, "[[deprecated]]": true,
		"[[fallthrough]]": true, "[[nodiscard]]": true, "[[maybe_unused]]": true,
		"[[likely]]": true, "[[unlikely]]": true, "[[no_unique_address]]": true,
		"__asm": true, "__attribute__": true, "__builtin": true, "__declspec": true,
		"__forceinline": true, "__inline": true, "__int64": true, "__ptr64": true,
		"__restrict": true, "__super": true, "__thread": true, "__uuidof": true,
		"__virtual_inheritance": true,
		"size_t":                true, "ptrdiff_t": true, "int8_t": true, "int16_t": true,
		"int32_t": true, "int64_t": true, "uint8_t": true, "uint16_t": true,
		"uint32_t": true, "uint64_t": true, "intptr_t": true, "uintptr_t": true,
		"string": true, "vector": true, "map": true, "set": true, "unordered_map": true,
		"unique_ptr": true, "shared_ptr": true, "weak_ptr": true, "function": true,
		"tuple": true, "optional": true, "variant": true, "any": true,
		"exception": true, "runtime_error": true, "logic_error": true,
		"out_of_range": true, "invalid_argument": true, "bad_alloc": true,
		"__live": true, "__domain": true,
	},
	LangAssembly: map[string]bool{
		"mov": true, "add": true, "sub": true, "mul": true, "div": true, "cmp": true,
		"jmp": true, "je": true, "jne": true, "jg": true, "jl": true, "jge": true, "jle": true,
		"call": true, "ret": true, "push": true, "pop": true, "lea": true, "nop": true,
		"int": true, "cli": true, "sti": true, "hlt": true, "in": true, "out": true,
		"section": true, "global": true, "extern": true, "db": true, "dw": true, "dd": true, "dq": true,
		"times": true, "equ": true,
	},
	LangFortran: map[string]bool{
		"abstract": true, "accept": true, "allocatable": true, "allocate": true,
		"assign": true, "assignment": true, "associate": true, "asynchronous": true,
		"backspace": true, "bind": true, "block": true, "blockdata": true,
		"call": true, "case": true, "change": true, "character": true,
		"class": true, "close": true, "codimension": true, "common": true,
		"complex": true, "concurrent": true, "contains": true, "contiguous": true,
		"continue": true, "critical": true, "cycle": true, "data": true,
		"deallocate": true, "default": true, "deferred": true, "dimension": true,
		"do": true, "double": true, "dowhile": true, "else": true, "elseif": true,
		"elsewhere": true, "end": true, "endassociate": true, "endblock": true,
		"endblockdata": true, "endcritical": true, "enddo": true, "endenum": true,
		"endfile": true, "endforall": true, "endfunction": true, "endif": true,
		"endinterface": true, "endmodule": true, "endprogram": true,
		"endselect": true, "endsubmodule": true, "endsubroutine": true,
		"endtype": true, "endwhere": true, "enum": true, "enumerator": true,
		"equivalence": true, "errmsg": true, "error": true, "event": true,
		"exit": true, "extends": true, "external": true, "final": true,
		"flush": true, "forall": true, "format": true, "function": true,
		"generic": true, "goto": true, "if": true, "implicit": true,
		"import": true, "impure": true, "in": true, "include": true,
		"inout": true, "inquire": true, "integer": true, "intent": true,
		"interface": true, "intrinsic": true, "is": true, "kind": true,
		"len": true, "local": true, "logical": true, "memory": true,
		"module": true, "namelist": true, "non_intrinsic": true, "non_overridable": true,
		"nopass": true, "nullify": true, "only": true, "open": true,
		"operator": true, "optional": true, "out": true, "parameter": true,
		"pass": true, "pause": true, "pointer": true, "precision": true,
		"print": true, "private": true, "procedure": true, "program": true,
		"protected": true, "public": true, "pure": true, "read": true,
		"real": true, "recursive": true, "result": true, "return": true,
		"rewind": true, "save": true, "select": true, "selectcase": true,
		"selectrank": true, "selecttype": true, "sequence": true,
		"shared": true, "source": true, "stat": true, "stop": true,
		"submodule": true, "subroutine": true, "sync": true, "syncall": true,
		"syncimages": true, "target": true, "then": true, "threadsafe": true,
		"type": true, "unformatted": true, "use": true, "value": true,
		"volatile": true, "wait": true, "where": true, "while": true,
		"write": true,
	},
	LangGo: map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
		"bool": true, "byte": true, "complex64": true, "complex128": true, "error": true,
		"float32": true, "float64": true, "int": true, "int8": true, "int16": true,
		"int32": true, "int64": true, "rune": true, "string": true, "uint": true,
		"uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
		"true": true, "false": true, "iota": true, "nil": true,
		"append": true, "cap": true, "close": true, "complex": true, "copy": true,
		"delete": true, "imag": true, "len": true, "make": true, "new": true,
		"panic": true, "print": true, "println": true, "real": true, "recover": true,
		"any": true, "comparable": true,
	},
	LangPython: map[string]bool{
		"False": true, "None": true, "True": true, "and": true, "as": true,
		"assert": true, "async": true, "await": true, "break": true, "class": true,
		"continue": true, "def": true, "del": true, "elif": true, "else": true,
		"except": true, "finally": true, "for": true, "from": true, "global": true,
		"if": true, "import": true, "in": true, "is": true, "lambda": true,
		"nonlocal": true, "not": true, "or": true, "pass": true, "raise": true,
		"return": true, "try": true, "while": true, "with": true, "yield": true,
		"__debug__": true, "Ellipsis": true, "NotImplemented": true,
		"bool": true, "bytearray": true, "bytes": true, "complex": true, "dict": true,
		"float": true, "frozenset": true, "int": true, "list": true, "memoryview": true,
		"object": true, "set": true, "str": true, "tuple": true, "type": true,
		"abs": true, "all": true, "any": true, "ascii": true, "bin": true,
		"callable": true, "chr": true, "classmethod": true, "compile": true,
		"delattr": true, "dir": true, "divmod": true, "enumerate": true,
		"eval": true, "exec": true, "filter": true, "format": true, "getattr": true,
		"globals": true, "hasattr": true, "hash": true, "help": true, "hex": true,
		"id": true, "input": true, "isinstance": true, "issubclass": true,
		"iter": true, "len": true, "locals": true, "map": true, "max": true,
		"min": true, "next": true, "oct": true, "open": true,
		"ord": true, "pow": true, "print": true, "property": true, "range": true,
		"repr": true, "reversed": true, "round": true, "setattr": true,
		"slice": true, "sorted": true, "staticmethod": true, "sum": true,
		"super": true, "vars": true, "zip": true,
		"__import__": true, "__name__": true, "__doc__": true, "__package__": true,
		"__class__": true, "__dict__": true, "__module__": true, "__annotations__": true,
		"os": true, "sys": true, "math": true, "json": true, "re": true,
		"datetime": true, "collections": true, "itertools": true, "functools": true,
		"BaseException": true, "Exception": true, "ArithmeticError": true,
		"LookupError": true, "AssertionError": true, "AttributeError": true,
		"EOFError": true, "FloatingPointError": true, "GeneratorExit": true,
		"ImportError": true, "IndexError": true, "KeyError": true, "KeyboardInterrupt": true,
		"MemoryError": true, "NameError": true, "NotImplementedError": true,
		"OSError": true, "OverflowError": true, "ReferenceError": true,
		"RuntimeError": true, "StopIteration": true, "SyntaxError": true,
		"IndentationError": true, "TabError": true, "SystemError": true,
		"SystemExit": true, "TypeError": true, "UnboundLocalError": true,
		"UnicodeError": true, "UnicodeEncodeError": true, "UnicodeDecodeError": true,
		"UnicodeTranslateError": true, "ValueError": true, "ZeroDivisionError": true,
		"Union": true, "Optional": true, "List": true, "Dict": true, "Tuple": true,
		"Set": true, "Any": true, "Callable": true, "Type": true, "Literal": true,
		"Final": true, "ClassVar": true,
	},
	LangRuby: map[string]bool{
		"alias": true, "and": true, "begin": true, "break": true, "case": true, "class": true,
		"def": true, "defined?": true, "do": true, "else": true, "elsif": true, "end": true,
		"ensure": true, "false": true, "for": true, "if": true, "in": true, "module": true,
		"next": true, "nil": true, "not": true, "or": true, "redo": true, "rescue": true,
		"retry": true, "return": true, "self": true, "super": true, "then": true, "true": true,
		"undef": true, "unless": true, "until": true, "when": true, "while": true, "yield": true,
	},
	LangKotlin: map[string]bool{
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
	},
	LangSwift: map[string]bool{
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
	},
	LangLisp: map[string]bool{
		"defun": true, "defvar": true, "defparameter": true, "defconstant": true,
		"let": true, "let*": true, "setf": true, "setq": true, "if": true,
		"cond": true, "case": true, "when": true, "unless": true, "loop": true,
		"do": true, "dolist": true, "dotimes": true, "lambda": true, "quote": true,
		"function": true, "progn": true, "prog1": true, "prog2": true, "block": true,
		"return": true, "return-from": true, "catch": true, "throw": true,
		"unwind-protect": true, "multiple-value-bind": true, "labels": true,
		"flet": true, "macrolet": true, "eval-when": true,
	},
}

// Popular identifiers for each language (limited set)
var languageIdentifiers = map[Language][]string{
	LangC: []string{
		"argc", "argv", "main", "printf", "scanf", "fprintf", "fscanf", "sprintf", "sscanf",
		"malloc", "calloc", "realloc", "free", "sizeof", "strlen", "strcpy", "strncpy",
		"strcmp", "strncmp", "strcat", "strncat", "strstr", "strchr", "strrchr",
		"memcpy", "memmove", "memcmp", "memset", "errno", "perror", "strerror",
		"exit", "atexit", "abort", "assert", "rand", "srand", "time", "clock",
		"fopen", "fclose", "fread", "fwrite", "fseek", "ftell", "rewind", "fflush",
		"getc", "putc", "fgets", "fputs", "getchar", "putchar", "gets", "puts",
		"buffer", "data", "size", "count", "index", "length", "result", "status",
		"value", "temp", "flag", "state", "error", "message", "input", "output",
		"file", "name", "path", "mode", "type", "info", "config", "options",
		"char", "int", "float", "double", "short", "long", "signed", "unsigned",
		"void", "struct", "union", "enum", "typedef", "const", "volatile", "static",
		"extern", "register", "auto", "goto", "switch", "case", "default", "break",
		"continue", "return", "if", "else", "for", "while", "do", "NULL", "true",
		"false", "bool", "stdin", "stdout", "stderr", "FILE", "EOF", "WEOF",
		"va_list", "va_start", "va_arg", "va_end", "offsetof", "container_of",
		"inline", "restrict", "_Alignas", "_Alignof", "_Atomic", "_Generic",
		"_Noreturn", "_Static_assert", "_Thread_local", "setjmp", "longjmp",
		"signal", "raise", "atexit", "at_quick_exit", "quick_exit", "abort",
		"getenv", "system", "qsort", "bsearch", "abs", "labs", "llabs", "div",
		"ldiv", "lldiv", "ceil", "floor", "fmod", "pow", "sqrt", "exp", "log",
		"log10", "sin", "cos", "tan", "asin", "acos", "atan", "atan2", "sinh",
		"cosh", "tanh", "fabs", "ldexp", "frexp", "modf", "isnan", "isinf",
		"isfinite", "fpclassify", "signbit", "copysign", "nextafter", "nan",
		"isalpha", "isdigit", "isalnum", "isxdigit", "isspace", "ispunct",
		"isprint", "isgraph", "iscntrl", "isupper", "islower", "toupper", "tolower",
		"strtod", "strtol", "strtoul", "atoi", "atol", "atof", "itoa", "ltoa",
		"ultoa", "gcvt", "fcvt", "ecvt", "setlocale", "localeconv",
	},
	LangCpp: []string{
		"argc", "argv", "main", "cout", "cin", "cerr", "endl", "string", "vector", "map",
		"list", "set", "queue", "stack", "pair", "make_pair", "begin", "end", "size",
		"push_back", "pop_back", "front", "back", "empty", "clear", "find", "insert",
		"erase", "at", "capacity", "reserve", "resize", "sort", "reverse", "max", "min",
		"swap", "copy", "fill", "transform", "for_each", "bind", "function", "shared_ptr",
		"unique_ptr", "weak_ptr", "make_shared", "thread", "mutex", "lock_guard",
		"unique_lock", "condition_variable", "future", "promise", "async", "buffer",
		"data", "count", "index", "length", "result", "status", "value", "temp", "flag",
		"state", "error", "message", "input", "output", "file", "name", "path", "mode",
		"class", "struct", "enum", "union", "template", "typename", "namespace", "using",
		"public", "private", "protected", "virtual", "override", "final", "explicit",
		"static", "const", "volatile", "mutable", "friend", "operator", "new", "delete",
		"this", "nullptr", "true", "false", "bool", "int", "float", "double", "char",
		"auto", "decltype", "typeid", "sizeof", "alignof", "dynamic_cast", "static_cast",
		"const_cast", "reinterpret_cast", "try", "catch", "throw", "noexcept", "exception",
		"std", "algorithm", "iterator", "memory", "utility", "functional", "chrono",
		"ratio", "random", "tuple", "array", "deque", "forward_list", "unordered_map",
		"unordered_set", "multimap", "multiset", "priority_queue", "bitset", "valarray",
		"complex", "regex", "atomic", "ref", "cref", "move", "forward", "emplace",
		"emplace_back", "push_front", "pop_front", "splice", "merge", "remove",
		"remove_if", "unique", "partition", "stable_sort", "partial_sort", "nth_element",
		"lower_bound", "upper_bound", "equal_range", "binary_search", "accumulate",
		"inner_product", "adjacent_difference", "partial_sum", "iota", "fill_n",
		"generate", "generate_n", "replace", "replace_if", "reverse_copy", "rotate",
		"rotate_copy", "shuffle", "sample", "is_sorted", "is_heap", "clamp", "gcd",
		"lcm", "hypot", "lerp", "midpoint", "format", "vformat", "make_format_args",
	},
	LangAssembly: []string{
		"eax", "ebx", "ecx", "edx", "esi", "edi", "ebp", "esp", "rax", "rbx", "rcx", "rdx",
		"rsi", "rdi", "rbp", "rsp", "ax", "bx", "cx", "dx", "ah", "al", "bh", "bl",
		"ch", "cl", "dh", "dl", "r8", "r9", "r10", "r11", "r12", "r13", "r14", "r15",
		"mm0", "mm1", "mm2", "mm3", "mm4", "mm5", "mm6", "mm7", "xmm0", "xmm1", "xmm2",
		"xmm3", "xmm4", "xmm5", "xmm6", "xmm7", "ymm0", "ymm1", "ymm2", "ymm3", "ymm4",
		"ymm5", "ymm6", "ymm7", "cr0", "cr1", "cr2", "cr3", "dr0", "dr1", "dr2", "dr3",
		"buffer", "data", "size", "count", "index", "length", "result", "status", "value",
		"temp", "flag", "state", "error", "message", "input", "output", "file", "name",
		"mov", "add", "sub", "mul", "div", "inc", "dec", "and", "or", "xor", "not",
		"neg", "shl", "shr", "sal", "sar", "rol", "ror", "rcl", "rcr", "cmp", "test",
		"jmp", "je", "jne", "jz", "jnz", "jg", "jge", "jl", "jle", "ja", "jae", "jb",
		"jbe", "call", "ret", "push", "pop", "pusha", "popa", "pushf", "popf", "lea",
		"lds", "les", "lfs", "lgs", "lss", "enter", "leave", "nop", "hlt", "wait",
		"lock", "rep", "repe", "repne", "cmps", "scas", "lods", "stos", "movs",
		"ins", "outs", "bswap", "xchg", "cmpxchg", "xadd", "bsf", "bsr", "bt", "btc",
		"btr", "bts", "sete", "setne", "setz", "setnz", "setg", "setge", "setl", "setle",
		"seta", "setae", "setb", "setbe", "sets", "setns", "seto", "setno", "setpe",
		"setpo", "shld", "shrd", "emms", "fld", "fst", "fstp", "fadd", "fsub", "fmul",
		"fdiv", "fcom", "fcomp", "fcompp", "fxch", "finit", "fldcw", "fstcw", "fldenv",
		"fstenv", "fsave", "frstor", "fincstp", "fdecstp", "ffree", "fabs", "fchs",
		"frndint", "fsqrt", "fscale", "fprem", "fptan", "fpatan", "f2xm1", "fyl2x",
		"fyl2xp1", "fld1", "fldl2t", "fldl2e", "fldpi", "fldlg2", "fldln2", "fldz",
		"fnop", "fwait", "fclex", "fdisi", "feni", "fsetpm", "fcos", "fsin", "fsincos",
		"ftst", "fxam", "fldenv", "fstenv", "fldcw", "fstcw", "fldpi", "fldz", "fld1",
	},
	LangFortran: []string{
		"program", "end", "implicit", "none", "integer", "real", "double", "precision",
		"complex", "character", "logical", "parameter", "dimension", "allocatable",
		"allocate", "deallocate", "pointer", "target", "if", "then", "else", "elseif",
		"endif", "do", "while", "enddo", "forall", "endforall", "select", "case",
		"endselect", "where", "elsewhere", "endwhere", "continue", "stop", "pause",
		"write", "read", "print", "open", "close", "inquire", "backspace", "endfile",
		"rewind", "format", "buffer", "data", "size", "count", "index", "length",
		"result", "status", "value", "temp", "flag", "state", "error", "message",
		"function", "subroutine", "module", "use", "contains", "return", "call",
		"intent", "optional", "public", "private", "save", "kind", "len", "allocated",
		"associated", "present", "null", "sizeof", "shape", "lbound", "ubound", "size",
		"min", "max", "sum", "product", "dot_product", "matmul", "transpose", "reshape",
		"pack", "unpack", "spread", "cshift", "eoshift", "merge", "mask", "count",
		"any", "all", "minval", "maxval", "sumval", "productval", "findloc", "minloc",
		"maxloc", "norm2", "parity", "leadz", "trailz", "popcnt", "poppar", "bge",
		"bgt", "ble", "blt", "dshiftl", "dshiftr", "maskl", "maskr", "merge_bits",
		"shifta", "shiftl", "shiftr", "storage_size", "btest", "iand", "ibclr",
		"ibits", "ibset", "ieor", "ior", "ishft", "ishftc", "mvbits", "not", "abs",
		"aimag", "aint", "anint", "ceiling", "conjg", "dim", "dprod", "floor", "mod",
		"modulo", "nint", "sign", "acos", "asin", "atan", "atan2", "cos", "cosh",
		"exp", "log", "log10", "sin", "sinh", "sqrt", "tan", "tanh", "epsilon", "huge",
		"tiny", "exponent", "fraction", "nearest", "rrspacing", "scale", "set_exponent",
		"spacing", "maxexponent", "minexponent", "precision", "radix", "range",
		"digits", "selected_int_kind", "selected_real_kind", "adjustl", "adjustr",
		"index", "scan", "verify", "len_trim", "repeat", "trim", "achar", "iachar",
		"char", "ichar", "new_line", "selected_char_kind", "lge", "lgt", "lle", "llt",
	},
	LangGo: []string{
		"main", "fmt", "Println", "Printf", "Scan", "Scanln", "Scanf", "Error", "String",
		"Int", "Bool", "Float64", "Len", "Cap", "Make", "New", "Append", "Copy", "Delete",
		"Close", "Len", "Cap", "panic", "recover", "defer", "goroutine", "channel",
		"select", "interface", "struct", "map", "slice", "array", "buffer", "data",
		"size", "count", "index", "length", "result", "status", "value", "temp", "flag",
		"state", "error", "message", "input", "output", "file", "name", "path", "mode",
		"package", "import", "const", "var", "type", "func", "range", "if", "else",
		"for", "switch", "case", "default", "break", "continue", "return", "goto",
		"fallthrough", "true", "false", "nil", "iota", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64",
		"complex64", "complex128", "byte", "rune", "string", "bool", "error", "comparable",
		"any", "append", "cap", "close", "complex", "copy", "delete", "imag", "len",
		"make", "new", "panic", "print", "println", "real", "recover", "chan", "go",
		"map", "struct", "interface", "func", "select", "defer", "goto", "const", "var",
		"type", "import", "package", "break", "case", "continue", "default", "else",
		"fallthrough", "for", "if", "range", "return", "switch", "embed", "any", "comparable",
		"iota", "nil", "true", "false", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64",
		"complex64", "complex128", "byte", "rune", "string", "bool", "error", "rune",
		"time", "Duration", "Time", "Now", "Sleep", "After", "Tick", "Since", "Until",
		"Parse", "Format", "Unix", "UnixNano", "Year", "Month", "Day", "Hour", "Minute",
		"Second", "Nanosecond", "UTC", "Local", "Date", "ParseDuration", "NewTicker",
		"NewTimer", "AfterFunc", "Ticker", "Timer", "Stop", "Reset", "C", "context",
		"Background", "WithValue", "WithCancel", "WithTimeout", "WithDeadline", "Done",
		"Err", "Value", "CancelFunc", "Deadline", "strings", "Contains", "HasPrefix",
		"HasSuffix", "Index", "Join", "Split", "ToLower", "ToUpper", "Trim", "Replace",
		"Repeat", "Compare", "Count", "Fields", "strconv", "Atoi", "Itoa", "ParseBool",
		"ParseFloat", "ParseInt", "FormatBool", "FormatFloat", "FormatInt", "Quote",
		"Unquote", "CanBackquote", "os", "Args", "Getenv", "Setenv", "Exit", "Stdin",
		"Stdout", "Stderr", "Open", "Create", "ReadFile", "WriteFile", "Remove", "Rename",
		"Mkdir", "MkdirAll", "RemoveAll", "Getwd", "Chdir", "Stat", "IsNotExist",
	},
	LangPython: []string{
		"def", "class", "import", "from", "as", "if", "elif", "else", "for", "while",
		"try", "except", "finally", "with", "lambda", "return", "yield", "break",
		"continue", "pass", "raise", "assert", "global", "nonlocal", "print", "len",
		"str", "int", "float", "bool", "list", "dict", "set", "tuple", "range", "enumerate",
		"zip", "map", "filter", "sorted", "reversed", "sum", "max", "min", "abs",
		"round", "pow", "divmod", "all", "any", "isinstance", "issubclass", "hasattr",
		"getattr", "setattr", "delattr", "buffer", "data", "size", "count", "index",
		"and", "or", "not", "is", "in", "None", "True", "False", "self", "super",
		"__init__", "__name__", "__main__", "__file__", "__doc__", "__package__",
		"__class__", "__module__", "__dict__", "__slots__", "__weakref__", "__str__",
		"__repr__", "__len__", "__getitem__", "__setitem__", "__delitem__", "__iter__",
		"__next__", "__contains__", "__call__", "__enter__", "__exit__", "__new__",
		"__del__", "__getattr__", "__setattr__", "__getattribute__", "__dir__",
		"property", "staticmethod", "classmethod", "abc", "abstractmethod", "ABC",
		"async", "await", "asyncdef", "asyncio", "sleep", "gather", "wait", "create_task",
		"run", "Queue", "Lock", "Event", "Semaphore", "Condition", "Barrier", "Thread",
		"Process", "Pool", "Manager", "Value", "Array", "Queue", "Pipe", "Lock", "RLock",
		"Event", "Condition", "Semaphore", "BoundedSemaphore", "Timer", "local", "current_thread",
		"active_count", "enumerate", "main_thread", "get_ident", "stack_size", "setprofile",
		"settrace", "excepthook", "except", "sys", "argv", "exit", "stdin", "stdout",
		"stderr", "path", "modules", "version", "platform", "executable", "byteorder",
		"maxsize", "maxunicode", "api_version", "version_info", "hexversion", "implementation",
		"thread_info", "copyright", "flags", "float_info", "int_info", "hash_info",
		"getsizeof", "getrefcount", "getrecursionlimit", "setrecursionlimit", "getswitchinterval",
		"setswitchinterval", "_current_frames", "_current_exceptions", "exc_info", "last_type",
		"last_value", "last_traceback", "exc_clear", "exception", "setprofile", "settrace",
		"breakpointhook", "displayhook", "excepthook", "unraisablehook", "version",
		"warnoptions", "winver", "write", "flush", "read", "readline", "seek", "tell",
		"truncate", "fileno", "isatty", "close", "closed", "mode", "name", "encoding",
		"errors", "newlines", "buffer", "detach", "peek", "readable", "writable",
		"seekable", "raw", "reconfigure", "line_buffering", "write_through", "TextIOWrapper",
		"BufferedReader", "BufferedWriter", "BufferedRandom", "BytesIO", "StringIO",
	},
	LangRuby: []string{
		"def", "class", "module", "include", "extend", "require", "load", "if", "elsif",
		"else", "unless", "case", "when", "while", "until", "for", "begin", "rescue",
		"ensure", "end", "do", "break", "next", "redo", "retry", "return", "yield",
		"self", "super", "nil", "true", "false", "and", "or", "not", "puts", "print",
		"p", "gets", "chomp", "strip", "split", "join", "length", "size", "empty",
		"include", "extend", "attr_accessor", "attr_reader", "attr_writer", "initialize",
		"new", "to_s", "to_i", "to_f", "to_a", "to_h", "each", "map", "select", "reject",
		"find", "filter", "reduce", "inject", "sort", "sort_by", "reverse", "shuffle",
		"alias", "undef", "defined", "BEGIN", "END", "__FILE__", "__LINE__", "__ENCODING__",
		"proc", "lambda", "block_given", "yield_self", "then", "tap", "public", "private",
		"protected", "module_function", "refine", "using", "prepend", "alias_method",
		"remove_method", "undef_method", "const_get", "const_set", "const_defined",
		"class_variable_get", "class_variable_set", "instance_variable_get",
		"instance_variable_set", "local_variables", "global_variables", "method",
		"define_method", "method_missing", "respond_to", "send", "public_send",
		"object_id", "display", "clone", "dup", "freeze", "frozen", "taint", "untaint",
		"tainted", "untrust", "trust", "untrusted", "hash", "class", "singleton_class",
		"inspect", "methods", "public_methods", "private_methods", "protected_methods",
		"instance_variables", "instance_of", "kind_of", "is_a", "nil", "equal", "eql",
		"===", "<=>", "=~", "!~", "==", "!=", ">", ">=", "<", "<=", "===", "|", "^", "&",
		"<=>", ">>", "<<", "+", "-", "*", "/", "%", "**", "~", "+@", "-@", "[]", "[]=",
		"`", "!", "!=", "!~", "!==", "unary", "nonzero", "zero", "positive", "negative",
		"floor", "ceil", "round", "truncate", "step", "times", "upto", "downto", "next",
		"pred", "chr", "ord", "to_int", "to_sym", "intern", "id2name", "downcase",
		"upcase", "capitalize", "swapcase", "reverse", "center", "ljust", "rjust",
		"chop", "chomp", "strip", "lstrip", "rstrip", "sub", "gsub", "scan", "index",
		"rindex", "match", "partition", "rpartition", "squeeze", "count", "delete",
		"tr", "tr_s", "encode", "encoding", "force_encoding", "b", "valid_encoding",
		"ascii_only", "sum", "hex", "oct", "crypt", "unpack", "unpack1", "bytes",
		"chars", "codepoints", "lines", "bytesize", "empty", "clear", "insert",
		"concat", "<<", "prepend", "replace", "slice", "slice!", "byteslice", "swapcase!",
		"upcase!", "downcase!", "capitalize!", "reverse!", "strip!", "lstrip!", "rstrip!",
	},
	LangKotlin: []string{
		"fun", "val", "var", "class", "interface", "object", "companion", "data", "sealed",
		"enum", "abstract", "final", "open", "internal", "protected", "private", "public",
		"inline", "infix", "operator", "tailrec", "external", "suspend", "const", "lateinit",
		"if", "else", "when", "for", "while", "do", "try", "catch", "finally", "throw",
		"return", "break", "continue", "this", "super", "is", "as", "in", "out", "by",
		"get", "set", "init", "constructor", "delegate", "apply", "also", "let", "run",
		"with", "takeIf", "takeUnless", "print", "println", "readLine", "toInt", "toDouble",
		"toFloat", "toLong", "toShort", "toByte", "toString", "toList", "toSet", "toMap",
		"package", "import", "typealias", "where", "dynamic", "actual", "expect",
		"override", "annotation", "reified", "crossinline", "noinline", "vararg",
		"Boolean", "Byte", "Short", "Int", "Long", "Float", "Double", "Char", "String",
		"Array", "List", "Set", "Map", "MutableList", "MutableSet", "MutableMap",
		"Sequence", "Flow", "Channel", "Job", "Deferred", "CoroutineScope", "Dispatcher",
		"withContext", "launch", "async", "await", "delay", "yield", "coroutineScope",
		"supervisorScope", "flow", "collect", "map", "filter", "transform", "take",
		"drop", "debounce", "distinctUntilChanged", "buffer", "conflate", "combine",
		"zip", "merge", "flatMap", "catch", "onCompletion", "onStart", "onEmpty",
		"onEach", "launchIn", "asFlow", "asLiveData", "observeOn", "subscribeOn",
		"Any", "Unit", "Nothing", "Throwable", "Exception", "Error", "RuntimeException",
		"NullPointerException", "IllegalArgumentException", "IllegalStateException",
		"IndexOutOfBoundsException", "UnsupportedOperationException", "NumberFormatException",
		"assert", "check", "require", "error", "TODO", "runCatching", "getOrNull",
		"getOrElse", "getOrThrow", "onSuccess", "onFailure", "recover", "mapCatching",
		"also", "let", "run", "with", "apply", "takeIf", "takeUnless", "repeat",
		"forEach", "filter", "map", "flatMap", "distinct", "sorted", "reversed",
		"groupBy", "associate", "associateWith", "associateBy", "partition", "zip",
		"unzip", "fold", "reduce", "sum", "average", "min", "max", "count", "any",
		"all", "none", "find", "first", "last", "single", "elementAt", "indexOf",
		"lastIndexOf", "contains", "isEmpty", "isNotEmpty", "plus", "minus", "times",
		"div", "rem", "rangeTo", "downTo", "until", "step", "in", "!in", "is", "!is",
		"as", "as?", "?:", "!!", "==", "!=", "===", "!==", ">", "<", ">=", "<=",
	},
	LangSwift: []string{
		"func", "var", "let", "class", "struct", "enum", "protocol", "extension", "import",
		"if", "else", "switch", "case", "default", "for", "while", "repeat", "while",
		"break", "continue", "return", "throw", "do", "try", "catch", "finally", "defer",
		"guard", "where", "in", "is", "as", "nil", "true", "false", "self", "Self", "super",
		"init", "deinit", "subscript", "convenience", "dynamic", "final", "infix",
		"internal", "lazy", "mutating", "nonmutating", "optional", "override", "postfix",
		"prefix", "required", "static", "unowned", "weak", "willSet", "didSet", "print",
		"println", "readLine", "toInt", "toDouble", "toFloat", "toLong", "toShort",
		"associatedtype", "indirect", "precedencegroup", "some", "any", "actor", "async",
		"await", "throws", "rethrows", "try", "try?", "try!", "catch", "defer", "fallthrough",
		"inout", "operator", "prefix", "postfix", "left", "right", "none", "assignment",
		"precedence", "higherThan", "lowerThan", "associativity", "Bool", "Int", "Int8",
		"Int16", "Int32", "Int64", "UInt", "UInt8", "UInt16", "UInt32", "UInt64", "Float",
		"Double", "String", "Character", "Optional", "Array", "Dictionary", "Set",
		"Range", "ClosedRange", "PartialRange", "Result", "Error", "Never", "Void",
		"Any", "AnyObject", "AnyClass", "Self", "Protocol", "Type", "sizeof", "strideof",
		"alignof", "unsafeAddress", "unsafeMutableAddress", "withUnsafePointer",
		"withUnsafeMutablePointer", "withUnsafeBytes", "withUnsafeMutableBytes",
		"unsafeBitCast", "unsafeDowncast", "transparent", "available", "discardableResult",
		"frozen", "unchecked", "inlinable", "usableFromInline", "main", "DispatchQueue",
		"async", "sync", "after", "once", "main", "global", "label", "qos", "attributes",
		"autoreleasepool", "objc", "nonobjc", "NSObject", "NSNumber", "NSString",
		"NSArray", "NSDictionary", "NSSet", "URL", "URLRequest", "URLSession",
		"Data", "JSONEncoder", "JSONDecoder", "PropertyListEncoder", "PropertyListDecoder",
		"FileManager", "FileHandle", "Notification", "NotificationCenter", "UserDefaults",
		"Bundle", "ProcessInfo", "Thread", "Operation", "OperationQueue", "BlockOperation",
		"Timer", "RunLoop", "Condition", "ConditionLock", "RecursiveLock", "NSCondition",
		"NSLock", "NSConditionLock", "NSRecursiveLock", "Semaphore", "DispatchGroup",
		"DispatchWorkItem", "DispatchSource", "DispatchTime", "DispatchWallTime",
		"DispatchQueue", "DispatchSemaphore", "OSLog", "Logger", "signpost", "point",
		"event", "begin", "end", "Codable", "Encodable", "Decodable", "CodingKey",
		"Encoder", "Decoder", "KeyedEncodingContainer", "KeyedDecodingContainer",
		"SingleValueEncodingContainer", "SingleValueDecodingContainer", "propertyWrapper",
		"wrappedValue", "projectedValue", "resultBuilder", "buildArray", "buildBlock",
		"buildEither", "buildExpression", "buildFinalResult", "buildLimitedAvailability",
		"buildOptional", "buildPartialBlock", "CaseIterable", "Comparable", "Equatable",
		"Hashable", "Identifiable", "RawRepresentable", "Sequence", "Collection",
		"BidirectionalCollection", "RandomAccessCollection", "RangeReplaceableCollection",
		"MutableCollection", "StringProtocol", "BinaryInteger", "FloatingPoint",
		"SignedNumeric", "Strideable", "CustomStringConvertible", "CustomDebugStringConvertible",
	},
	LangLisp: []string{
		"defun", "defvar", "defparameter", "defconstant", "let", "let*", "setf", "setq",
		"if", "cond", "case", "when", "unless", "loop", "do", "dolist", "dotimes", "lambda",
		"quote", "function", "progn", "prog1", "prog2", "block", "return", "return-from",
		"catch", "throw", "unwind-protect", "multiple-value-bind", "labels", "flet",
		"macrolet", "eval-when", "car", "cdr", "cons", "list", "append", "reverse",
		"length", "nth", "first", "second", "third", "rest", "last", "butlast", "member",
		"assoc", "subst", "sublis", "nsubst", "nsublis", "mapcar", "mapc", "mapcan",
		"some", "every", "notany", "notevery", "reduce", "sort", "merge", "remove",
		"delete", "substitute", "nsubstitute", "find", "position", "count", "mismatch",
		"search", "concatenate", "coerce", "fill", "replace", "rotate", "shuffle",
		"and", "or", "not", "eq", "eql", "equal", "equalp", "=", "/=", "<", ">", "<=", ">=",
		"+", "-", "*", "/", "mod", "rem", "incf", "decf", "1+", "1-", "abs", "floor",
		"ceiling", "truncate", "round", "max", "min", "sin", "cos", "tan", "asin", "acos",
		"atan", "sinh", "cosh", "tanh", "exp", "expt", "log", "sqrt", "isqrt", "random",
		"make-random-state", "numberp", "integerp", "floatp", "complexp", "zerop",
		"plusp", "minusp", "oddp", "evenp", "characterp", "stringp", "symbolp", "listp",
		"consp", "atom", "null", "endp", "arrayp", "vectorp", "simple-vector-p",
		"bit-vector-p", "simple-bit-vector-p", "stringp", "simple-string-p", "packagep",
		"functionp", "compiled-function-p", "commonp", "typep", "subtypep", "boundp",
		"fboundp", "special-form-p", "macro-function", "compiler-macro-function",
		"values", "values-list", "multiple-value-call", "multiple-value-list",
		"multiple-value-prog1", "multiple-value-setq", "nth-value", "prog", "prog*",
		"progv", "unwind-protect", "catch", "throw", "handler-case", "handler-bind",
		"ignore-errors", "define-condition", "make-condition", "signal", "error",
		"cerror", "warn", "break", "invoke-debugger", "restart-case", "restart-bind",
		"with-simple-restart", "find-restart", "invoke-restart", "compute-restarts",
		"abort", "continue", "muffle-warning", "store-value", "use-value", "interactive",
		"eval", "apply", "funcall", "complement", "constantly", "identity", "compose",
		"disjoin", "conjoin", "curry", "rcurry", "always", "never", "ensure", "maybe",
		"defpackage", "in-package", "export", "unexport", "import", "shadow", "shadowing-import",
		"use-package", "unuse-package", "find-package", "find-symbol", "intern",
		"unintern", "package-name", "package-nicknames", "package-use-list",
		"package-used-by-list", "package-shadowing-symbols", "list-all-packages",
		"with-open-file", "with-input-from-string", "with-output-to-string",
		"with-standard-io-syntax", "read", "read-preserving-whitespace", "read-delimited-list",
		"read-line", "read-char", "unread-char", "peek-char", "listen", "clear-input",
		"read-char-no-hang", "write", "prin1", "print", "pprint", "princ", "write-to-string",
		"prin1-to-string", "princ-to-string", "format", "parse-integer", "parse-namestring",
		"make-string-input-stream", "make-string-output-stream", "get-output-stream-string",
		"with-open-stream", "copy-stream", "stream-element-type", "streamp", "input-stream-p",
		"output-stream-p", "interactive-stream-p", "open-stream-p", "stream-external-format",
	},
}

func splitArgs(raw string) []string {
	var args []string
	var cur []rune
	inDouble := false
	inSingle := false
	escaped := false

	for _, r := range raw {
		switch {
		case escaped:
			cur = append(cur, r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == ' ' && !inDouble && !inSingle:
			if len(cur) > 0 {
				args = append(args, string(cur))
				cur = nil
			}
		default:
			cur = append(cur, r)
		}
	}
	if len(cur) > 0 {
		args = append(args, string(cur))
	}
	return args
}

// This wrapper converts the args to a string and delegates to the internal runner.
func (e *Editor) runCodeViaRuncode(code string, lang string, flags string, runArgs string) (string, string, error) {
	args := strings.TrimSpace(runArgs)
	return e.runCodeInternally(code, lang, flags, args)
}

func (e *Editor) runCodeInternally(code string, lang string, flags string, runArgs string) (string, string, error) {
	l := strings.ToLower(strings.TrimSpace(lang))
	if l == "" {
		if strings.Contains(code, "package main") && strings.Contains(code, "func main") {
			l = "go"
		} else if strings.Contains(code, "#include <stdio.h>") {
			l = "c"
		} else {
			l = "go"
		}
	}

	switch l {
	case "cpp", "c++":
		return runCppInternal(code, flags, runArgs)
	case "c":
		return runCInternal(code, flags, runArgs)
	case "assembly", "asm":
		return runAssemblyInternal(code, flags, runArgs)
	case "fortran":
		return runFortranInternal(code, flags, runArgs)
	case "go":
		return runGoInternal(code, flags, runArgs)
	case "python", "py":
		return runPythonInternal(code, flags, runArgs)
	case "ruby", "rb":
		return runRubyInternal(code, flags, runArgs)
	case "kotlin", "kt":
		return runKotlinInternal(code, flags, runArgs)
	case "swift":
		return runSwiftInternal(code, flags, runArgs)
	case "html":
		return "", "", openHTMLInBrowser(code)
	case "lisp", "common lisp":
		return runLispInternal(code, flags, runArgs)
	default:
		return "", "", fmt.Errorf("unsupported language: %s", lang)
	}
}

func (e *Editor) llmQueryWithoutInsert(instruction string) (string, error) {
	if strings.TrimSpace(e.llmProvider) == "" {
		e.llmProvider = "ollama"
	}
	if strings.TrimSpace(e.llmModel) == "" {
		e.llmModel = "qwen3:1.7b"
	}
	e.statusMessage("Sending analysis request to LLM...")
	out, err := SendMessageToLLM(instruction, e.llmProvider, e.llmModel, e.llmKey)
	if err != nil {
		e.showError("Analysis error: " + err.Error())
		return "", err
	}
	e.statusMessage("Analysis completed successfully")
	return string(out), nil
}

func (e *Editor) handleRunCode() {
	e.statusMessage("Analyzing code with LLM...")
	code := strings.Join(e.lines, "\n")
	analysisQuery := "Analyze this code and return a JSON object with fields: language, flags, and args to run the" +
		"code (if the argument of the program is necessary for its correct launch, then come up with it yourself, based on" +
		"the requirements of the code, but do not specify the flags of the program name, for example: -o multiplication)." +
		" The JSON must be exactly in the format: {\"language\":\"<lang>\",\"flags\":\"<flags>\",\"args\":\"<args>\"}." +
		" Provide no extra text. Code:\n" + code

	analysis, err := e.llmQueryWithoutInsert(analysisQuery)
	if err != nil {
		e.showError("Parsing error: " + err.Error())
		return
	}

	type llmResponse struct {
		Language string `json:"language"`
		Flags    string `json:"flags"`
		Args     string `json:"args"`
	}
	var resp llmResponse

	if err := json.Unmarshal([]byte(analysis), &resp); err != nil {
		s := strings.TrimSpace(analysis)
		start := strings.IndexByte(s, '{')
		end := strings.LastIndexByte(s, '}')
		if start != -1 && end != -1 && end >= start {
			if err2 := json.Unmarshal([]byte(s[start:end+1]), &resp); err2 != nil {
				e.showError("Invalid JSON from LLM: " + err2.Error())
				return
			}
		} else {
			e.showError("Invalid JSON from LLM: " + err.Error())
			return
		}
	}

	lang := strings.TrimSpace(resp.Language)
	flags := strings.TrimSpace(resp.Flags)
	args := strings.TrimSpace(resp.Args)

	if lang == "" {
		lang = "go"
	}
	e.statusMessage("Running code with " + lang + "...")
	stdout, stderr, runErr := e.runCodeViaRuncode(code, lang, flags, args)
	if runErr != nil {
		e.showError("Execution error: " + runErr.Error())
		errorQuery := "Analyze the error and suggest corrections for the code: Error: " + runErr.Error() +
			"\nStderr: " + stderr + "\nCode:\n" + code
		e.statusMessage("Requesting error analysis from LLM...")
		fixes, err2 := e.llmQueryWithoutInsert(errorQuery)
		if err2 != nil {
			e.showError("Error obtaining corrections:" + err2.Error())
			return
		}
		e.insertLLMResponse("\n\n// Recommendations for correction:\n" + fixes)
		e.statusMessage("Error analysis completed - fixes suggested")
	} else {
		e.insertLLMResponse("\n\n// No errors were detected in the code. \n// Execution result.\n" + stdout)
		e.statusMessage("Code executed successfully")
	}
}

// New internal helpers: per-language implementations that compile/run locally
// and capture stdout/stderr. These do not depend on the external "run" binary.
func runCppInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.cpp")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	exe := tmp.Name() + ".out"

	compileArgs := []string{tmp.Name(), "-o", exe}
	if strings.TrimSpace(compileFlags) != "" {
		compileArgs = append(compileArgs, strings.Fields(compileFlags)...)
	}
	compile := exec.Command("g++", compileArgs...)
	var cbuf bytes.Buffer
	compile.Stdout = &cbuf
	compile.Stderr = &cbuf
	if err := compile.Run(); err != nil {
		return "", cbuf.String(), fmt.Errorf("compile error: %w", err)
	}

	run := exec.Command(exe)
	if strings.TrimSpace(runArgs) != "" {
		run.Args = append([]string{exe}, splitArgs(runArgs)...)
	} else {
		run.Args = []string{exe}
	}
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runCInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.c")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()

	exe := tmp.Name() + ".out"

	compileArgs := []string{tmp.Name(), "-o", exe}
	if strings.TrimSpace(compileFlags) != "" {
		compileArgs = append(compileArgs, strings.Fields(compileFlags)...)
	}
	compile := exec.Command("gcc", compileArgs...)
	var cbuf bytes.Buffer
	compile.Stdout = &cbuf
	compile.Stderr = &cbuf
	if err := compile.Run(); err != nil {
		return "", cbuf.String(), fmt.Errorf("compile error: %w", err)
	}

	run := exec.Command(exe)
	if strings.TrimSpace(runArgs) != "" {
		run.Args = append([]string{exe}, splitArgs(runArgs)...)
	} else {
		run.Args = []string{exe}
	}
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runAssemblyInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.asm")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()

	obj := tmp.Name() + ".o"
	exe := tmp.Name() + ".out"

	var nasmFormat string
	switch runtime.GOOS {
	case "linux", "darwin":
		nasmFormat = "elf64"
	case "windows":
		nasmFormat = "win64"
	default:
		nasmFormat = "elf64"
	}

	nasmArgs := []string{"-f", nasmFormat, tmp.Name(), "-o", obj}

	if strings.TrimSpace(compileFlags) != "" {
		nasmArgs = append(nasmArgs, splitArgs(compileFlags)...)
	}

	asm := exec.Command("nasm", nasmArgs...)
	asmOut, err := asm.CombinedOutput()
	if err != nil {
		return "", string(asmOut), fmt.Errorf("nasm error: %w", err)
	}
	defer os.Remove(obj)
	var ld *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		ld = exec.Command("ld", obj, "-o", exe)
	case "darwin":
		ld = exec.Command("cc", obj, "-o", exe)
	case "windows":
		ld = exec.Command("ld", obj, "-o", exe)
	default:
		ld = exec.Command("ld", obj, "-o", exe)
	}

	ldOut, err := ld.CombinedOutput()
	if err != nil {
		return "", string(ldOut), fmt.Errorf("link error: %w", err)
	}
	defer os.Remove(exe)
	run := exec.Command(exe)

	if strings.TrimSpace(runArgs) != "" {

		run.Args = append(run.Args, splitArgs(runArgs)...)
	}

	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr

	err = run.Run()

	return stdout.String(), stderr.String(), err
}

func runFortranInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.f90")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	exe := tmp.Name() + ".out"

	args := []string{tmp.Name(), "-o", exe}
	if compileFlags != "" {
		args = append(args, strings.Fields(compileFlags)...)
	}
	compile := exec.Command("gfortran", args...)
	compileOut, err := compile.CombinedOutput()
	if err != nil {
		return "", string(compileOut), fmt.Errorf("compile error: %w", err)
	}
	run := exec.Command(exe)
	if strings.TrimSpace(runArgs) != "" {
		run.Args = append([]string{exe}, splitArgs(runArgs)...)
	} else {
		run.Args = []string{exe}
	}
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runGoInternal(code string, compileFlags string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.go")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()

	args := []string{tmp.Name()}
	if strings.TrimSpace(runArgs) != "" {
		extra := splitArgs(runArgs)
		if len(extra) > 0 {
			args = append(args, extra...)
		}
	}
	cmd := exec.Command("go", append([]string{"run"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func runPythonInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.py")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	args := []string{tmp.Name()}
	if strings.TrimSpace(runArgs) != "" {
		args = append(args, splitArgs(runArgs)...)
	}
	cmd := exec.Command("python3", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func runRubyInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.rb")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	args := []string{tmp.Name()}
	if strings.TrimSpace(runArgs) != "" {
		args = append(args, splitArgs(runArgs)...)
	}
	cmd := exec.Command("ruby", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func runKotlinInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.kt")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	jar := tmp.Name() + ".jar"
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()

	compile := exec.Command("kotlinc", tmp.Name(), "-include-runtime", "-d", jar)
	if out, err := compile.CombinedOutput(); err != nil {
		return "", string(out), fmt.Errorf("kotlinc error: %w", err)
	}
	var run *exec.Cmd
	args := []string{"-jar", jar}
	if strings.TrimSpace(runArgs) != "" {
		args = append(args, splitArgs(runArgs)...)
	}
	run = exec.Command("java", args...)
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runSwiftInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.swift")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	exe := tmp.Name() + ".out"
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	compile := exec.Command("swiftc", tmp.Name(), "-o", exe)
	if out, err := compile.CombinedOutput(); err != nil {
		return "", string(out), fmt.Errorf("swiftc error: %w", err)
	}
	defer os.Remove(exe)
	run := exec.Command(exe)
	if strings.TrimSpace(runArgs) != "" {
		run.Args = append([]string{exe}, splitArgs(runArgs)...)
	} else {
		run.Args = []string{exe}
	}
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

func runLispInternal(code string, _ string, runArgs string) (string, string, error) {
	tmp, err := ioutil.TempFile("", "runner_*.lisp")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(code); err != nil {
		tmp.Close()
		return "", "", err
	}
	tmp.Close()
	cmd := exec.Command("sbcl", "--script", tmp.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if strings.TrimSpace(runArgs) != "" {
		cmd.Args = append(cmd.Args, splitArgs(runArgs)...)
	}
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func openHTMLInBrowser(content string) error {
	tmp, err := ioutil.TempFile("", "runner_*.html")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open -a Safari ", tmp.Name())
	case "windows":
		cmd = exec.Command("start msedge ", tmp.Name())
	default:
		cmd = exec.Command("firefox ", tmp.Name())
	}
	return cmd.Start()
}

// insertContextualLLMResponse inserts the LLM response contextually based on the current mode
func (e *Editor) insertContextualLLMResponse(resp string, isIncomplete bool) {
	if strings.TrimSpace(resp) == "" {
		return
	}

	resp = strings.ReplaceAll(resp, "\r\n", "\n")
	respLines := strings.Split(resp, "\n")

	if len(respLines) == 0 {
		return
	}

	if e.cy < 0 {
		e.cy = 0
	}

	for e.cy >= len(e.lines) {
		e.lines = append(e.lines, "")
	}

	if isIncomplete && len(respLines) > 0 {
		currentLine := e.lines[e.cy]
		if len(currentLine) > 0 {
			e.lines[e.cy] = currentLine + respLines[0]

			insertIndex := e.cy + 1
			for i := 1; i < len(respLines); i++ {
				if insertIndex >= len(e.lines) {
					e.lines = append(e.lines, respLines[i])
				} else {
					e.lines = append(e.lines[:insertIndex], append([]string{respLines[i]}, e.lines[insertIndex:]...)...)
				}
				insertIndex++
			}

			e.cy = insertIndex - 1
			if e.cy >= 0 && e.cy < len(e.lines) {
				e.cx = len([]rune(e.lines[e.cy]))
			}
		} else {
			// Insert all lines after the current line
			insertIndex := e.cy + 1
			for i := 0; i < len(respLines); i++ {
				if insertIndex >= len(e.lines) {
					e.lines = append(e.lines, respLines[i])
				} else {
					e.lines = append(e.lines[:insertIndex], append([]string{respLines[i]}, e.lines[insertIndex:]...)...)
				}
				insertIndex++
			}

			e.cy = insertIndex - 1
			if e.cy >= 0 && e.cy < len(e.lines) {
				e.cx = len([]rune(e.lines[e.cy]))
			}
		}
	} else {
		insertIndex := e.cy + 1
		for i := 0; i < len(respLines); i++ {
			if insertIndex >= len(e.lines) {
				e.lines = append(e.lines, respLines[i])
			} else {
				e.lines = append(e.lines[:insertIndex], append([]string{respLines[i]}, e.lines[insertIndex:]...)...)
			}
			insertIndex++
		}

		e.cy = insertIndex - 1
		if e.cy >= 0 && e.cy < len(e.lines) {
			e.cx = len([]rune(e.lines[e.cy]))
		}
	}

	e.dirty = true
	e.ensureVisible()
}

// isAtEndOfIncompleteStatement checks if the cursor is at the end of an incomplete statement
func (e *Editor) isAtEndOfIncompleteStatement() bool {
	if e.cy < 0 || e.cy >= len(e.lines) {
		return false
	}

	line := e.lines[e.cy]
	lineRunes := []rune(line)

	if e.cx != len(lineRunes) {
		return false
	}

	if len(lineRunes) == 0 {
		return false
	}

	terminator, exists := languageTerminators[e.language]
	if !exists || terminator == "" {
		return false
	}

	trimmed := strings.TrimRight(line, " \t")
	if len(trimmed) == 0 {
		return false
	}

	if terminator == ";" {
		if !strings.HasSuffix(trimmed, ";") {
			trimmedContent := strings.TrimSpace(trimmed)
			if len(trimmedContent) > 0 && !strings.HasPrefix(trimmedContent, "//") && !strings.HasPrefix(trimmedContent, "/*") &&
				!strings.HasPrefix(trimmedContent, "#") && !strings.HasPrefix(trimmedContent, ";") {
				if !strings.HasSuffix(trimmedContent, "{") && !strings.HasSuffix(trimmedContent, ":") {
					if !strings.HasPrefix(trimmedContent, "#") {
						return true
					}
				}
			}
		}
	}

	if terminator == ":" {
		trimmedContent := strings.TrimSpace(trimmed)
		pythonStatements := []string{"if", "else", "elif", "for", "while", "def", "class", "try", "except", "finally", "with"}
		for _, stmt := range pythonStatements {
			if strings.HasSuffix(trimmedContent, stmt) {
				return true
			}
			if strings.Contains(trimmedContent, stmt+" ") && !strings.HasSuffix(trimmedContent, ":") {
				return true
			}
		}
	}

	return false
}

// getContextAroundCursor gets the context around the cursor for code completion
// Returns the context lines and whether the cursor is at the end of an incomplete statement
func (e *Editor) getContextAroundCursor() (string, bool) {
	linesBefore := 5
	linesAfter := 2

	startLine := e.cy - linesBefore
	if startLine < 0 {
		startLine = 0
	}

	endLine := e.cy + linesAfter
	if endLine >= len(e.lines) {
		endLine = len(e.lines) - 1
	}

	contextLines := make([]string, 0)
	for i := startLine; i <= endLine; i++ {
		contextLines = append(contextLines, e.lines[i])
	}

	isIncomplete := e.isAtEndOfIncompleteStatement()

	return strings.Join(contextLines, "\n"), isIncomplete
}

// findKeywordCompletion checks if the word before cursor is a partial keyword and returns the full keyword
func (e *Editor) findKeywordCompletion() string {
	if e.cy < 0 || e.cy >= len(e.lines) {
		return ""
	}

	line := e.lines[e.cy]
	runes := []rune(line)

	if e.cx != len(runes) {
		return ""
	}

	keywords, exists := languageKeywords[e.language]
	if !exists {
		return ""
	}

	if len(runes) == 0 {
		return ""
	}

	start := len(runes)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == ' ' || runes[i] == '\t' {
			start = i + 1
			break
		}
		if i == 0 {
			start = 0
		}
	}

	// If we're at the beginning of the line or after a space
	if start >= len(runes) {
		return ""
	}

	lastWordRunes := runes[start:]
	lastWord := string(lastWordRunes)

	if len(lastWord) == 0 {
		return ""
	}

	for keyword := range keywords {
		if len(keyword) > len(lastWord) && strings.HasPrefix(strings.ToLower(keyword), strings.ToLower(lastWord)) {
			return keyword[len(lastWord):]
		}
	}

	return ""
}

// findIdentifierCompletion checks if the word before cursor is a partial identifier and returns the full identifier
func (e *Editor) findIdentifierCompletion() string {
	if e.cy < 0 || e.cy >= len(e.lines) {
		return ""
	}

	line := e.lines[e.cy]
	runes := []rune(line)

	if e.cx != len(runes) {
		return ""
	}

	identifiers, exists := languageIdentifiers[e.language]
	if !exists {
		return ""
	}

	if len(runes) == 0 {
		return ""
	}

	start := len(runes)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == ' ' || runes[i] == '\t' {
			start = i + 1
			break
		}
		if i == 0 {
			start = 0
		}
	}

	if start >= len(runes) {
		return ""
	}

	lastWordRunes := runes[start:]
	lastWord := string(lastWordRunes)

	if len(lastWord) == 0 {
		return ""
	}

	for _, identifier := range identifiers {
		if len(identifier) > len(lastWord) && strings.HasPrefix(strings.ToLower(identifier), strings.ToLower(lastWord)) {
			return identifier[len(lastWord):]
		}
	}

	return ""
}

// shouldAutoCloseBracket checks if we should automatically close a bracket
func (e *Editor) shouldAutoCloseBracket(openBracket rune) bool {
	if e.cy < 0 || e.cy >= len(e.lines) {
		return false
	}

	line := e.lines[e.cy]
	runes := []rune(line)

	if e.cx >= len(runes) {
		return true
	}

	nextChar := runes[e.cx]
	return nextChar == ' ' || nextChar == '\t' ||
		nextChar == ')' || nextChar == ']' || nextChar == '}' ||
		nextChar == ';' || nextChar == ',' || nextChar == ':'
}

// getClosingBracket returns the matching closing bracket for an opening one
func getClosingBracket(openBracket rune) rune {
	switch openBracket {
	case '(':
		return ')'
	case '[':
		return ']'
	case '{':
		return '}'
	case '"':
		return '"'
	case '\'':
		return '\''
	default:
		return 0
	}
}
