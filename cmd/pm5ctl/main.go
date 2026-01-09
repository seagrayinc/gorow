package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seagrayinc/pm5-csafe/pkg/csafe"
	"github.com/seagrayinc/pm5-csafe/pkg/hid"
	"github.com/seagrayinc/pm5-csafe/pkg/pm5"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM, syscall.SIGINT,
	)
	defer stop()

	mgr, err := hid.NewManager()
	if err != nil {
		panic(err)
	}

	dev, err := mgr.OpenVIDPID(pm5.Concept2VID, pm5.PM5PID)
	if err != nil {
		panic(err)
	}

	transport := csafe.Transport{
		Device:        dev,
		ReportLengths: pm5.ReportLengths,
	}

	err = transport.Send(ctx, pm5.GetID())
	if err != nil {
		panic(err)
	}

	reports := dev.PollReports(ctx)
	go func() {
		var prevStrokeState int
		for f := range transport.Poll(ctx, reports) {
			parsed, err := pm5.ParseResponses(f)
			if err != nil {
				fmt.Printf("error parsing response: %+v\n", err)
				continue
			}
			for _, r := range parsed {
				ss, ok := r.(pm5.GetStrokeStateResponse)
				if !ok {
					continue
				}

				if prevStrokeState == 4 && ss.StrokeState == 2 {
					fmt.Println("new stroke!")
				}

				prevStrokeState = ss.StrokeState
			}
		}
	}()

	for {
		if ctx.Err() != nil {
			break
		}

		time.Sleep(75 * time.Millisecond)
		if err := transport.Send(ctx, pm5.GetStrokeState()); err != nil {
			fmt.Printf("%+v\n", err)
		}

	}
	<-ctx.Done()
}
