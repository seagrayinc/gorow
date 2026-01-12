package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seagrayinc/gorow/pkg/pm5"
)

type Workout struct {
	LastWorkoutState pm5.GetWorkoutStateResponse
	LastStrokeStats  pm5.GetStrokeStatsResponse
	LastStrokeState  pm5.GetStrokeStateResponse
}

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM, syscall.SIGINT,
	)
	defer stop()

	p, err := pm5.Open(ctx)
	if err != nil {
		panic(err)
	}

	err = p.Send(ctx, pm5.GetStatus())
	if err != nil {
		panic(err)
	}

	go func() {
		var workout Workout
		for r := range p.EventStream() {
			fmt.Printf("%T %+v\n", r, r)
			switch resp := r.(type) {
			case pm5.GetIDResponse:
				fmt.Printf("PM ID: %s%s%s%s%s\n", string(resp.ASCIIDigit0), string(resp.ASCIIDigit1), string(resp.ASCIIDigit2), string(resp.ASCIIDigit3), string(resp.ASCIIDigit4))
			case pm5.GetVersionResponse:
			case pm5.GetPowerResponse:
				fmt.Printf("Power: %d W\n", resp.StrokeWatts)
			case pm5.GetStrokeStatsResponse:
				fmt.Printf("%+v\n", resp)
			case pm5.GetWorkoutStateResponse:
				if workout.LastWorkoutState.WorkoutState != resp.WorkoutState {
					workout.LastWorkoutState = resp
					fmt.Println("workout state changed: ", resp.WorkoutStateString)
				}
			case pm5.GetStrokeStateResponse:
				if workout.LastStrokeState.StrokeState > 2 && resp.StrokeState == 2 {
					fmt.Println("new stroke!")
					_ = p.Send(ctx, pm5.GetStrokeStats(), pm5.GetPower())
				}
				workout.LastStrokeState = resp
			}

		}
	}()

	for {
		if ctx.Err() != nil {
			break
		}

		time.Sleep(1000 * time.Millisecond)
		fmt.Println("-----")
		if err := p.Send(ctx, pm5.GetStrokeState(), pm5.GetWorkoutState()); err != nil {
			fmt.Printf("%+v\n", err)
		}

	}
	<-ctx.Done()
}
