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

	err = transport.Send(ctx, pm5.GetID{})
	if err != nil {
		panic(err)
	}

	reports := dev.PollReports(ctx)
	go func() {
		for f := range transport.Poll(ctx, reports) {
			fmt.Printf("%+v\n", f)
		}
	}()

	for {
		if ctx.Err() != nil {
			break
		}

		time.Sleep(5 * time.Second)
		fmt.Printf("sending!")
		if err := transport.Send(ctx, pm5.GetID{}); err != nil {
			fmt.Printf("%+v\n", err)
		}

	}
	<-ctx.Done()
}
