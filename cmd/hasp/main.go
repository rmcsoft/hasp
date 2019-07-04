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
	PackedImageDir string `short:"i" long:"image-dir" description:"Packed image directory needed for animation"`
	UseSDL         bool   `short:"s" long:"use-sdl"   description:"Render with sdl"`
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
		paintEngine, err = chanim.NewSDLPaintEngine(600, 1024)
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

func main() {
	opts := parseCmd()
	animator := makeAnimator(opts)
	animationNames := animator.GetAnimationNames()

	err := animator.Start(animationNames[0])
	if err != nil {
		panic(err)
	}
	for {
		fmt.Printf("Available animations:\n")
		for _, animationName := range animationNames {
			fmt.Printf("\t%s:\n", animationName)
		}

		nextAnimation := ""
		fmt.Printf("Next animation -> ")
		fmt.Scanf("%s", &nextAnimation)

		err = animator.ChangeAnimation(nextAnimation)
		if err != nil {
			fmt.Println(err)
		}
	}
}
