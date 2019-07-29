package main

import (
	"fmt"
	"log"
	"time"

	"periph.io/x/periph"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"

	atmel "github.com/rmcsoft/hasp/atmel/periph_gpio"
)

func main() {
	// Register your driver in the registry:
	if err := periph.Register(atmel.AtmelGpioDriver{
		Pins: atmel.AtmelGpioPins{
			atmel.AtmelGpioPin{91, "pioC27"},
			atmel.AtmelGpioPin{92, "pioC28"},
		},
	}); err != nil {
		log.Fatal(err)
	}
	// Initialize normally. Your driver will be loaded:
	status, err := host.Init()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(status.Loaded)
	fmt.Println(gpioreg.Aliases())
	fmt.Println(gpioreg.All())

	pin27 := gpioreg.ByName("pioC27")
	pin28 := gpioreg.ByName("pioC28")
	if pin27 == nil {
		log.Fatal("Failed to open pin C27")
	}
	if pin28 == nil {
		log.Fatal("Failed to open pin C28")
	}
	err = pin27.In(gpio.PullNoChange, gpio.NoEdge)
	if err != nil {
		log.Fatal(err)
	}
	err = pin28.In(gpio.PullNoChange, gpio.NoEdge)
	if err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		log.Println("pioC27 == ", pin27.Read())
		log.Println("pioC28 == ", pin28.Read())
		<-ticker.C
	}
}
