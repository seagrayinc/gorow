// Package pm5 implements the PM5 CSAFE Protocol defined at
// https://www.concept2.sg/files/pdf/us/monitors/PM5_CSAFECommunicationDefinition.pdf

package pm5

import (
	"context"
	"time"

	"github.com/seagrayinc/gorow/internal/csafe"
	"github.com/seagrayinc/gorow/internal/hid"
)

const (
	vidConcept2 uint16 = 0x17A4
	pidPM5      uint16 = 0x001E
)

var (
	reportLengths = map[byte]int{
		0x01: 21,
		0x02: 121,
		0x04: 501,
	}
)

type Command = csafe.Command

// PM5 represents a connection to a Concept2 PM5 monitor over USB HID.
type PM5 struct {
	events    chan any
	transport csafe.Transport
}

// Open opens a connection to the PM5 monitor.
func Open(ctx context.Context) (*PM5, error) {
	mgr, err := hid.NewManager()
	if err != nil {
		panic(err)
	}

	dev, err := mgr.OpenVIDPID(vidConcept2, pidPM5)
	if err != nil {
		panic(err)
	}

	p := &PM5{
		events: make(chan any, 100),
		transport: csafe.Transport{
			Device:        dev,
			ReportLengths: reportLengths,
			SendTimeout:   50 * time.Millisecond,
			SendBuffer:    100,
		},
	}
	p.transport.StartSender(ctx)

	reports := dev.PollReports(ctx)
	go func() {
		for f := range p.transport.Poll(ctx, reports) {
			parsed, err := parseResponses(f)
			if err != nil {
				continue
			}

			for _, r := range parsed {
				p.events <- r
			}
		}
	}()

	return p, nil
}

// EventStream returns a channel that emits PM5 events as they are received.
func (p *PM5) EventStream() <-chan any {
	return p.events
}

// Send sends one or more CSAFE commands to the PM5.
func (p *PM5) Send(ctx context.Context, commands ...Command) error {
	return p.transport.Send(ctx, commands...)
}
