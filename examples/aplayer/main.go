package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rmcsoft/hasp"

	"github.com/jessevdk/go-flags"
	"github.com/rmcsoft/chanim"
)

const (
	pixFormat = chanim.RGB16
)

type options struct {
	PackedImageDir string `short:"i" long:"image-dir" description:"Packed image directory needed for animation" required:"true"`
	UseSDL         bool   `short:"s" long:"use-sdl"   description:"Render with sdl"`
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
			os.Exit(0)
		}
		fail(err)
	}

	if opts.PackedImageDir, err = filepath.Abs(opts.PackedImageDir); err != nil {
		fail(err)
	}

	return opts
}

func makePaintEngine(opts options) chanim.PaintEngine {
	var err error
	var paintEngine chanim.PaintEngine
	if opts.UseSDL {
		paintEngine, err = chanim.NewSDLPaintEngine(600, 1024)
	} else {
		paintEngine, err = chanim.NewKMSDRMPaintEngine(0, pixFormat)
	}
	if err != nil {
		fail(err)
	}

	return paintEngine
}

func makeAnimator(opts options) *chanim.Animator {
	paintEngine := makePaintEngine(opts)
	animator, err := hasp.CreateAnimator(paintEngine, opts.PackedImageDir)
	if err != nil {
		fail(err)
	}

	return animator
}

func main() {
	opts := parseCmd()
	animator := makeAnimator(opts)
	animationNames := animator.GetAnimationNames()

	err := animator.Start(animationNames[0])
	if err != nil {
		fail(err)
	}
	for {
		fmt.Printf("Available animations:\n")
		for i, animationName := range animationNames {
			fmt.Printf("\t%v %s:\n", i, animationName)
		}

		var numAnimation int
		fmt.Printf("Enter the number of the next animation -> ")
		fmt.Scanf("%v", &numAnimation)
		if numAnimation < 0 || numAnimation >= len(animationNames) {
			fmt.Printf("Invalid animation number\n")
			continue
		}

		nextAnimation := animationNames[numAnimation]
		err = animator.ChangeAnimation(nextAnimation)
		if err != nil {
			fmt.Println(err)
		}
	}
}
