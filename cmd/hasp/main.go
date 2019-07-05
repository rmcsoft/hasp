package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rmcsoft/hasp"

	"github.com/jessevdk/go-flags"
	"github.com/rmcsoft/chanim"
)

const (
	pixFormat = chanim.RGB16
)

type options struct {
	PackedImageDir string `short:"i" long:"image-dir" description:"Packed image directory needed for animation"`
	UseSDL         bool   `short:"s" long:"use-sdl"   description:"Render with sdl"`

	CaptureDevice  string `short:"c" long:"capture-dev" default:"hw:0" description:"Sound capture device name"`
	PlayDevice     string `short:"p" long:"play-dev"    default:"mono" description:"Sound play device name"`
	ModelParamPath string `short:"m" long:"model-param" description:"Path to file containing model parameters"`
	KeywordPath    string `short:"k" long:"keyword"     description:"Path to keyword file"`
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

	if opts.PackedImageDir, err = filepath.Abs(opts.PackedImageDir); err != nil {
		panic(err)
	}

	return opts
}

func makePaintEngine(opts options) chanim.PaintEngine {
	var err error
	var paintEngine chanim.PaintEngine
	if opts.UseSDL {
		paintEngine, err = chanim.NewSDLPaintEngine(1024, 600)
	} else {
		paintEngine, err = chanim.NewKMSDRMPaintEngine(0, pixFormat)
	}
	if err != nil {
		panic(err)
	}

	return paintEngine
}

func makeAnimator(opts options) *chanim.Animator {
	paintEngine := makePaintEngine(opts)
	animator, err := hasp.CreateAnimator(paintEngine, opts.PackedImageDir)
	if err != nil {
		panic(err)
	}
	return animator
}

func makeHotWordDetector(opts options) hasp.EventSource {
	hotWordDetector, err := hasp.NewHotWordDetector(opts.CaptureDevice, opts.ModelParamPath, opts.KeywordPath)
	if err != nil {
		panic(err)
	}
	return hotWordDetector
}

func makeSoundPlayer(opts options) *hasp.SoundPlayer {
	player, err := hasp.NewSoundPlayer(opts.PlayDevice, 16000)
	if err != nil {
		panic(err)
	}
	return player
}

func makeCharacter(opts options) *hasp.Character {
	states := hasp.States{
		"idle": hasp.NewIdleState(
			[]string{"lotus", "reading"},
			time.Duration(2)*time.Minute),
	}

	eventSources := hasp.EventSources{
		makeHotWordDetector(opts),
	}

	animator := makeAnimator(opts)
	soundPlayer := makeSoundPlayer(opts)
	character, err := hasp.NewCharacter("idle", states, nil, eventSources, animator, soundPlayer)
	if err != nil {
		panic(err)
	}

	return character
}

func main() {
	opts := parseCmd()

	character := makeCharacter(opts)
	err := character.Run()
	if err != nil {
		panic(err)
	}
}
