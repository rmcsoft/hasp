package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/rmcsoft/hasp"
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
	hotWordDetector, err := hasp.NewHotWordDetector(opts.CaptureDevice, opts.ModelParamPath, opts.KeywordPath)
	if err != nil {
		fail(err)
	}

	for event := range hotWordDetector.Events() {
		d, _ := event.GetHotWordDetectedEventData()
		fmt.Printf("samplesCount=%v, sampleRate=%v\n", len(d.Samples), d.SampleRate)
	}

	fmt.Printf("HotWordDetector was closed\n")
}
