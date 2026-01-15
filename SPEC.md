# BML Specification

**Version:** 1.0

BML (Binary Markup Language) is a hierarchical, indentation-based markup
language designed for configuration files. It prioritizes human readability
and minimal syntax.

## Document Structure

A BML document consists of zero or more nodes. Each node has:

- A **name** (required)
- A **value** (optional)
- Zero or more **child nodes**
- Zero or more **attributes** (inline child nodes)

## Syntax

### Node Names

Node names consist of one or more valid characters:

- Uppercase letters: `A-Z`
- Lowercase letters: `a-z`
- Digits: `0-9`
- Hyphen: `-`
- Period: `.`

```text
ValidName
Another-Name
Node.Name.123
```

### Values

Values are associated with nodes using one of three formats:

#### Colon Format

```text
Name: value
```

- A colon followed by optional whitespace and the value
- Value extends to end of line (trailing whitespace trimmed)
- Leading space after colon is trimmed

#### Equals Format (Unquoted)

```text
Name=value
```

- An equals sign followed immediately by the value
- Value cannot contain spaces or quotes
- Terminates at first space or end of line

#### Equals Format (Quoted)

```text
Name="value with spaces"
```

- An equals sign followed by a double-quoted string
- Value can contain spaces
- Terminated by closing double quote
- No escape sequences defined

### Hierarchy

Hierarchy is expressed through indentation using spaces or tabs:

```text
Parent
  Child1
  Child2
    Grandchild
```

- Root nodes have zero indentation
- Child nodes have greater indentation than their parent
- Sibling nodes have equal indentation
- Mixed tabs and spaces are allowed but discouraged

### Attributes

Attributes are inline child nodes on the same line as the parent:

```text
Node attr1=value1 attr2: value2
```

- Separated by spaces
- Use the same value formats as regular nodes
- Become children of the node

### Comments

Comments begin with `//` and extend to end of line:

```text
// This is a comment
Node: value  // Inline comment
```

- Full-line comments are skipped entirely
- Inline comments terminate value parsing

### Multiline Values

Values can span multiple lines using continuation syntax:

```text
Description
  : First line
  : Second line
  : Third line
```

- Continuation lines start with `:` at deeper indentation
- Lines are joined with newline characters
- Leading space after `:` is trimmed

A node can have both an initial value and continuations:

```text
Description: Initial line
  : Continuation line
```

### Empty Values

Nodes may have no value:

```text
EmptyNode
Parent
  AlsoEmpty
```

### Whitespace

- **Line endings:** `\n`, `\r\n`, and `\r` are all recognized
- **Empty lines:** Ignored during parsing
- **Indentation:** Spaces and tabs (each counts as one level)
- **Trailing whitespace:** Trimmed from values

## Grammar

```ebnf
document     = { node } ;
node         = indent, name, [value], {attribute}, [comment], newline, {child};
child        = node ;  (* with greater indentation *)
attribute    = name , [ value ] ;
name         = name_char , { name_char } ;
name_char    = "A"-"Z" | "a"-"z" | "0"-"9" | "-" | "." ;
value        = colon_value | equals_value | quoted_value ;
colon_value  = ":" , [ " " ] , text ;
equals_value = "=" , unquoted_text ;
quoted_value = "=" , '"' , text , '"' ;
comment      = "//" , text ;
indent       = { " " | "\t" } ;
newline      = "\n" | "\r\n" | "\r" ;
text         = { any_char - newline } ;
unquoted_text = { any_char - " " - '"' - newline } ;
```

## Example

```text
// Application settings
Video
  Driver: Metal
  Multiplier: 2
  Luminance: 1.0
  ColorBleed: false
  Quality: HD

Audio
  Driver: SDL
  Device: Default
  Volume: 0.8
  Mute: false

Paths
  Home
  Saves: /Users/example/.local/share/app/saves
  Firmware
    BIOS.US
    BIOS.Japan
    BIOS.Europe

Hotkeys
  Save: 0x1/0/2
  Load: 0x1/0/3
  Fullscreen: F11

Description
  : This is a multiline
  : description that spans
  : several lines.
```

## Data Types

BML is untyped at the format level. All values are strings. Type
interpretation is application-defined:

| Type    | Convention                             |
| ------- | -------------------------------------- |
| String  | Raw value                              |
| Boolean | `true` or `false`                      |
| Integer | Decimal digits, optional leading `-`   |
| Float   | Decimal with `.`, optional leading `-` |

## Encoding

- BML files should be UTF-8 encoded
- No BOM (Byte Order Mark) required

## File Extension

- `.bml` is the conventional extension

## References

- Original implementation: [ares emulator][ares] `nall/string/markup/bml.hpp`

[ares]: https://github.com/ares-emulator/ares
