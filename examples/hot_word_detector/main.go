package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rmcsoft/hasp"
)

func main() {
	capDev := os.Args[1]
	modelPath := os.Args[2]
	keywordPath := os.Args[3]

	hotWordDetector, err := hasp.NewHotWordDetector(capDev, modelPath, keywordPath)
	if err != nil {
		panic(err)
	}

	go func() {
		time.Sleep(5 * time.Minute)
		hotWordDetector.Close()
	}()

	for event := range hotWordDetector.Events() {
		v, _ := event.GetVoice()
		fmt.Printf("Samples count (voice)=%v\n", len(v))
	}

	fmt.Printf("HotWordDetector was closed\n")
}
