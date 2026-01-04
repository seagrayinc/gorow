package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/seagrayinc/pm5-csafe/pkg/hid"
	"github.com/seagrayinc/pm5-csafe/pkg/pm5"
)

func main() {
	flag.Parse()

	mgr, err := hid.NewManager()
	if err != nil {
		panic(err)
	}

	performanceMonitor, err := pm5.Open(mgr)
	if err != nil {
		panic(err)
	}

	defer performanceMonitor.Close()

	ctx := context.Background()

	id, err := performanceMonitor.GetID(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", id)

	version, err := performanceMonitor.GetVersion(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", version)

	status, err := performanceMonitor.GetStatus(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", status)

	//power, err := performanceMonitor.GetStrokeStats(ctx)
	//fmt.Printf("%+v\n", power)

	//return

	for {
		time.Sleep(100 * time.Millisecond)
		power, err := performanceMonitor.GetWorkoutState(ctx)
		if err != nil {
			//fmt.Println(err)
			continue
		}

		fmt.Printf("%+v\n", power)
	}
	return
}
