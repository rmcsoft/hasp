package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/rmcsoft/hasp"
)

type options struct {
	CaptureDevice  string `short:"c" long:"capture-dev" default:"hw:0"`
	ModelParamPath string `short:"m" long:"model-param"`
	KeywordPath    string `short:"k" long:"keyword"`
}

func parseCmd() options {
	var opts options
	var cmdParser = flags.NewParser(&opts, flags.Default)
	var err error

	if _, err = cmdParser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			panic(err)
		}
	}

	return opts
}

func main() {
	opts := parseCmd()
	hotWordDetector, err := hasp.NewHotWordDetector(opts.CaptureDevice, opts.ModelParamPath, opts.KeywordPath)
	if err != nil {
		panic(err)
	}

	for event := range hotWordDetector.Events() {
		v, _ := event.GetVoice()
		fmt.Printf("Samples count (voice)=%v\n", len(v))
	}

	fmt.Printf("HotWordDetector was closed\n")
}
