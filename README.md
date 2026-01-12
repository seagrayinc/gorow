# GoRow

A pure Go library for communicating with Concept2 Performance Monitors.

> ⚠️ **Alpha Software**: This library is currently in alpha. The API may change without notice.

## Features

- Pure Go implementation with no CGO dependencies
- USB HID communication with Concept2 PM5 monitors
- Event-driven architecture for real-time workout data
- Support for CSAFE (Communication Specification for Fitness Equipment) protocol

## Requirements

- **Operating System**: Windows 11 only (currently)
- **Hardware**: Single Concept2 PM5 connected via USB

## Installation

```bash
go get github.com/seagrayinc/gorow
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/seagrayinc/gorow/pkg/pm5"
)

func main() {
    ctx, stop := signal.NotifyContext(
        context.Background(),
        os.Interrupt,
        syscall.SIGTERM, syscall.SIGINT,
    )
    defer stop()

    // Open connection to PM5
    p, err := pm5.Open(ctx)
    if err != nil {
        panic(err)
    }

    // Send a command
    err = p.Send(ctx, pm5.GetStatus())
    if err != nil {
        panic(err)
    }

    // Listen for events
    for r := range p.EventStream() {
        fmt.Printf("%T %+v\n", r, r)
    }
}
```

## Supported Commands

- `CSAFE_GETSTATUS_CMD`
- `CSAFE_GETID_CMD`
- `CSAFE_GETVERSION_CMD`
- `CSAFE_GETPOWER_CMD`
- `CSAFE_PM_GETSTROKESTATE`
- `CSAFE_PM_GETSTROKESTATS`
- `CSAFE_PM_GETWORKOUTSTATE`

## Examples

See the [examples](./examples) subdirectory for complete working examples.

## Documentation

This implementation is based on the Concept2 PM5 CSAFE Communication Definition:
https://www.concept2.sg/files/pdf/us/monitors/PM5_CSAFECommunicationDefinition.pdf

## Roadmap

Future additions may include:
- Support for additional operating systems (macOS, Linux)
- Bluetooth connectivity
- Additional PM commands
- Support for multiple connected devices

## License

MIT License - see [LICENSE.txt](./LICENSE.txt) for details.

## Disclaimer

This project is not affiliated with, endorsed by, or sponsored by Concept2, Inc.

