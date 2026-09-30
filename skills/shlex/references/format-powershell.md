# PowerShell Lexical Format

Lexical rules a command-line lexer needs for PowerShell. PowerShell uses backtick (`` ` ``) as the escape character (not backslash), `''`/`""` doubled-quote escaping, here-strings (`@'...'@` / `@"..."@`), the `--%` stop-parsing token, and `|`/`;`/`&&`/`||` operators.

> **Source of truth**: PowerShell docs ([About Quoting Rules](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_quoting_rules), [About Special Characters](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_special_characters)). For broader PowerShell internals, use the **powershell** skill. For cross-shell comparison, see [comparison.md](comparison.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` — backtick escapes |
| non-escaping quote | `'` | `QUOTING_STATE` — `''` → `'` |
| escape | `` ` `` (backtick) | `ESCAPING_STATE` |
| comment | `#` | `COMMENT_STATE` |

**Key difference from POSIX**: the escape character is backtick (`` ` ``), not backslash (`\`). Backslash is a literal character in PowerShell.

## Quotes

### Single quotes `'...'`

Verbatim — no substitution, no escape processing. The backtick is literal inside single quotes. **`''` (doubled single quote) produces one literal `'`** — same as elvish and zsh's `RC_QUOTES`.

```powershell
'$HOME'                 # $HOME (literal)
'don''t'                # don't ('' → ')
'hello world'           # hello world
```

For the lexer: `QUOTING_STATE` with `''` peek-ahead (consume both, emit one `'`, stay in state).

### Double quotes `"..."`

Expandable — variable substitution (`$variable`) and subexpression evaluation (`$(...)`) occur. The backtick is the escape character. `""` (doubled double quote) produces one literal `"`.

```powershell
"$HOME"                 # /home/user (expanded)
"say `"hello`""         # say "hello (`" escapes the quote)
"The value is $(2+3)"   # The value is 5
"don""t"                # don"t ("" → ")
```

Backtick escape sequences inside double quotes:

| Sequence | Meaning |
|----------|---------|
| `` `$ `` | literal `$` (prevent expansion) |
| `` `" `` | literal `"` |
| `` `` `` | literal backtick |
| `` `0 `` `` `a `` `` `b `` `` `e `` `` `f `` `` `n `` `` `r `` `` `t `` `` `v `` | control chars |
| `` `u{xxxx} `` | Unicode (PS 6+) |

For the lexer: inside `"..."`, backtick enters `ESCAPING_QUOTED_STATE` — the next rune is literal, then back to `QUOTING_ESCAPING_STATE`. `""` is also a valid escape (doubled quote) — peek-ahead on `"` in `QUOTING_ESCAPING_STATE`.

### Here-strings `@'...'@` / `@"..."@`

Multi-line strings. Opening `@"` or `@'` must be followed by a newline. Closing `"@` or `'@` must be at the start of a line.

```powershell
@"
Line 1: $HOME
Line 2: $(Get-Date)
"@

@'
Line 1: $HOME
'@
```

For the lexer: here-strings require a multi-line scan. The `@"` / `@'` opener is a two-rune sequence at a word boundary. The closer (`"@` / `'@`) must be at the beginning of a line. This is the most complex string type to tokenize — it may need a format-specific scan routine rather than the single-rune state machine.

## Escape Character

Backtick `` ` ``:

- **Outside quotes**: escapes the next character (literal). Primarily used for line continuation (`` ` `` + newline) and escaping metacharacters.
- **Inside double quotes**: the escape sequences above.
- **Inside single quotes**: **literal** (not an escape).

```powershell
echo `$HOME             # $HOME
echo ``                 # ` (literal backtick)
```

## Word Breaks and Operators

PowerShell metacharacters (special at token start or throughout):

```
<space>  '  "  `  ,  ;  (  )  {  }  |  &  <  >  @  #
```

Operators:

| Operator | Meaning | Type |
|----------|---------|------|
| `\|` | pipe | pipeline delimiter |
| `;` | command separator | pipeline delimiter |
| `>` `>>` | redirects | redirect |
| `2>` `2>&1` etc. | stream redirects | redirect |
| `&&` `\|\|` | logical and/or (PS 7+) | list operator |
| `&` | background/call operator | (call, not separator) |

**Note**: `&` in PowerShell is the *call operator* (runs a command), not a command separator like in cmd. `;` is the statement separator. `&&`/`||` are conditional operators added in PowerShell 7.

## Comments

`#` to end of line. PowerShell also supports block comments `<# ... #>` (multi-line).

```powershell
echo hello # a comment
<#
  multi-line
  comment
#>
```

For the lexer: `#` starts `COMMENT_STATE` (terminated by `\n`). Block comments `<# ... #>` need format-specific handling — the `<#` opener and `#>` closer, with `\n` not terminating the comment.

## The `--%` Stop-Parsing Token

PowerShell 3.0+ `--%` stops PowerShell from interpreting subsequent input. After `--%`, only `%variable%` environment variable expansion occurs; everything else is literal until newline or `|`.

```powershell
icacls X:\VMS --% /grant Dom\HVAdmin:(CI)(OI)F
```

For the lexer: `--%` is a special word that, once seen, switches the remainder of the line (until `\n` or `|`) to a raw/literal lexing mode. This is a format-specific state.

## Argument Passing Modes

`$PSNativeCommandArgumentPassing` (PS 7.3+) affects how PowerShell passes args to native commands:

| Mode | Default | Behavior |
|------|---------|----------|
| `Legacy` | — | quotes stripped, empty strings lost |
| `Standard` | non-Windows | quotes preserved, empty strings preserved |
| `Windows` | Windows | Standard, but Legacy for .bat/.cmd/etc. |

For the lexer this affects whether quote characters reach the completion function. Carapace's PowerShell integration uses the AST (`CommandAst`) which provides already-parsed arguments, so the lexer is used for the raw `commandAst` text rather than individual args.

## Edge Cases

- **Backtick vs backslash**: the single biggest departure from POSIX. The classifier must map `` ` `` to `escapeRuneClass`, not `\`.
- **`''` and `""` doubled quotes**: both quote types support doubled-quote escaping, unlike POSIX where only `"..."` has escapes.
- **Here-strings**: multi-line, line-start closer — needs special scan.
- **`<# #>` block comments**: multi-line comment.
- **`--%` stop-parsing**: format-specific raw mode.
- **`@` at token start**: `@` is a metacharacter (here-string opener, splat operator) — not a wordbreak in the bash sense.

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model
- [format-elvish.md](format-elvish.md) — shared `''` doubled-quote behavior

## Related Skills

- **powershell** skill — PowerShell completion, PSReadLine, styling, language, startup
- **carapace-dev** skill → `references/shell-powershell.md` — carapace's PowerShell output formatting
