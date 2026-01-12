# Copilot Instructions for GoRow

## Implementing a New CSAFE Command

When asked to implement a CSAFE command from the Concept2 PM5 specification, follow these steps:

### 1. Create the Command File

Create a new file in `pkg/pm5/` named `csafe_<commandname>_cmd.go` (lowercase, underscores).

### 2. File Structure

#### For commands with NO response data (N/A):

```go
package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_<COMMANDNAME>_CMD = 0xXX  // Command identifier from spec

func <CommandName>() Command {
	return csafe.ShortCommand(csafe_<COMMANDNAME>_CMD)
}
```

#### For commands WITH response data:

```go
package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_<COMMANDNAME>_CMD = 0xXX  // Command identifier from spec

// Optional: Define constants for response values if applicable
const (
	<ValueName> byte = 0xXX
)

func <CommandName>() Command {
	return csafe.ShortCommand(csafe_<COMMANDNAME>_CMD)
}

type <CommandName>Response struct {
	// Fields matching the response data from the spec
	// Byte 0: first field, Byte 1: second field, etc.
}

func parse<CommandName>Response(b []byte) (<CommandName>Response, error) {
	return <CommandName>Response{
		// Parse bytes from b[] into struct fields
		// Use binary.LittleEndian for multi-byte values
	}, nil
}
```

### 3. Register the Parser (for commands with response data)

Add the parser to `parserMap` in `pkg/pm5/commands.go`:

```go
var (
	parserMap = map[byte]parserFunc{
		// ...existing entries...
		csafe_<COMMANDNAME>_CMD: wrappedParser(parse<CommandName>Response),
	}
)
```

### 4. Update the README

Add the new command to the "Supported Commands" table in `README.md`:

```markdown
| `CSAFE_<COMMANDNAME>_CMD` | `pm5.<CommandName>()` | Description of command |
```

### 5. Build and Test

Run `go build ./...` to verify the implementation compiles correctly.

## Example: Implementing CSAFE_GETUNITS_CMD

Given spec:
```
Command Name: CSAFE_GETUNITS_CMD
Command Identifier: 0x93
Response Data: Byte 0: Units Type
```

1. Create `pkg/pm5/csafe_getunits_cmd.go`:

```go
package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GETUNITS_CMD = 0x93

func GetUnits() Command {
	return csafe.ShortCommand(csafe_GETUNITS_CMD)
}

type GetUnitsResponse struct {
	UnitsType byte
}

func parseGetUnitsResponse(b []byte) (GetUnitsResponse, error) {
	return GetUnitsResponse{
		UnitsType: b[0],
	}, nil
}
```

2. Add to `parserMap` in `commands.go`:
```go
csafe_GETUNITS_CMD: wrappedParser(parseGetUnitsResponse),
```

3. Add to README table:
```markdown
| `CSAFE_GETUNITS_CMD` | `pm5.GetUnits()` | Get units type |
```

## Naming Conventions

- Command constant: `csafe_<COMMANDNAME>_CMD` (lowercase prefix, uppercase name)
- Function name: `<CommandName>()` (PascalCase, e.g., `GetStatus`, `GoIdle`, `BadID`)
- Response type: `<CommandName>Response`
- Parser function: `parse<CommandName>Response` (unexported)
- File name: `csafe_<commandname>_cmd.go` (all lowercase with underscores)

## Common Response Data Parsing

- Single byte: `b[0]`
- Two bytes (uint16, little-endian): `binary.LittleEndian.Uint16(b[:2])`
- Four bytes (uint32, little-endian): `binary.LittleEndian.Uint32(b[:4])`

