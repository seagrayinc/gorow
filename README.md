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

| Command | Function | Description |
|---------|----------|-------------|
| `CSAFE_GETSTATUS_CMD` | `pm5.GetStatus()` | Get machine status |
| `CSAFE_RESET_CMD` | `pm5.Reset()` | Reset the machine |
| `CSAFE_GOIDLE_CMD` | `pm5.GoIdle()` | Set machine to idle state |
| `CSAFE_GOHAVEID_CMD` | `pm5.GoHaveID()` | Set machine to have ID state |
| `CSAFE_GOINUSE_CMD` | `pm5.GoInUse()` | Set machine to in-use state |
| `CSAFE_GOFINISHED_CMD` | `pm5.GoFinished()` | Set machine to finished state |
| `CSAFE_GOREADY_CMD` | `pm5.GoReady()` | Set machine to ready state |
| `CSAFE_BADID_CMD` | `pm5.BadID()` | Signal bad ID |
| `CSAFE_GETID_CMD` | `pm5.GetID()` | Get machine ID |
| `CSAFE_GETVERSION_CMD` | `pm5.GetVersion()` | Get firmware version |
| `CSAFE_GETPOWER_CMD` | `pm5.GetPower()` | Get stroke power |
| `CSAFE_PM_GETSTROKESTATE` | `pm5.GetStrokeState()` | Get current stroke state |
| `CSAFE_PM_GETSTROKESTATS` | `pm5.GetStrokeStats()` | Get stroke statistics |
| `CSAFE_PM_GETWORKOUTSTATE` | `pm5.GetWorkoutState()` | Get workout state |

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

