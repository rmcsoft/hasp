package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/rmcsoft/hasp/sound"
)

type options struct {
	CaptureDevice  string `short:"c" long:"capture-dev" default:"hw:0" description:"Sound capture device name"`
	ModelParamPath string `short:"m" long:"model-param" description:"Path to file containing model parameters" required:"true"`
	KeywordPath    string `short:"k" long:"keyword"     description:"Path to keyword file" required:"true"`
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func parseCmd() options {
	var opts options
	var cmdParser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var err error

	if _, err = cmdParser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			fmt.Println(flagsErr)
			os.Exit(0)
		}
		fail(err)
	}

	return opts
}

func main() {
	opts := parseCmd()

	hotWordDetectorParams := sound.HotWordDetectorParams{
		CaptureDeviceName: opts.CaptureDevice,
		ModelPath:         opts.ModelParamPath,
		KeywordPath:       opts.KeywordPath,
	}

	hotWordDetector, err := sound.NewHotWordDetector(hotWordDetectorParams)
	if err != nil {
		fail(err)
	}

	eventSource, err := hotWordDetector.StartDetect()
	if err != nil {
		fail(err)
	}

	var n int
	ticker := time.NewTicker(time.Duration(30) * time.Second)
	for {
		select {
		case event, ok := <-eventSource.Events():
			if ok {
				d, _ := sound.GetHotWordDetectedEventData(event)
				fmt.Printf("samplesCount=%v, sampleRate=%v\n", len(d.Samples), d.SampleRate)
			} else {
				eventSource, err = hotWordDetector.StartDetect()
				if err != nil {
					fail(err)
				}

				n++
				fmt.Printf("n=%v, t=%v: HotWordDetector was closed\n", n, time.Now().Unix())
			}

		case <-ticker.C:
			eventSource.Close()
		}
	}
}
