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

	time.Sleep(5 * time.Second)
	for {
		power, err := performanceMonitor.GetPower(ctx)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("%+v\n", power)
		time.Sleep(250 * time.Millisecond)
	}
	return
}
