package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/seagrayinc/pm5-csafe/internal/hid"
	"github.com/seagrayinc/pm5-csafe/pm5csafe"
)

func main() {
	flag.Parse()

	mgr, err := hid.NewManager()
	if err != nil {
		panic(err)
	}

	dev, err := mgr.OpenVIDPID(pm5csafe.Concept2VID, pm5csafe.PM5PID)
	if err != nil {
		log.Fatal("PM5 not found by VID")
	}

	defer dev.Close()

	if adv, ok := dev.(hid.Device); ok {
		// Send GETID command
		frame := pm5csafe.NewHIDReport(pm5csafe.NewExtendedFrame(pm5csafe.CSAFEGetVersion()))
		if _, err := adv.Write(frame); err != nil {
			log.Fatalf("Write failed: %v", err)
		}

		resp := make([]byte, len(frame))
		_, err := adv.Read(resp)
		if err != nil {
			log.Fatalf("ReadInput failed: %v", err)
		}

		if len(resp) > 0 {
			fmt.Println(hex.EncodeToString(resp))
		} else {
			log.Fatal("no data received")
		}

		response, err := pm5csafe.ParseExtendedHIDResponse(resp)
		if err != nil {
			panic(err)
		}

		fmt.Println(hex.EncodeToString([]byte{response.FrameToggle()}))
		fmt.Println(hex.EncodeToString([]byte{response.PreviousFrameStatus()}))
		fmt.Println(hex.EncodeToString([]byte{response.StateMachineState()}))
		responses, err := response.CommandResponses()
		if err != nil {
			panic(err)
		}
		fmt.Println(hex.EncodeToString(responses[0].Data))
		parsed, err := pm5csafe.ParseGetVersionResponse(responses[0].Data)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v", parsed)
	} else {
		log.Fatal("device doesn't support Advanced interface")
	}
	return
}
