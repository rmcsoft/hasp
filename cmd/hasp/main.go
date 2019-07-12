package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rmcsoft/chanim"
	"github.com/rmcsoft/hasp"
	"github.com/rmcsoft/hasp/events"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lexruntimeservice"
	"github.com/jessevdk/go-flags"
)

const (
	pixFormat = chanim.RGB16
)

type options struct {
	PackedImageDir string `short:"i" long:"image-dir" description:"Packed image directory needed for animation" required:"true"`
	UseSDL         bool   `short:"s" long:"use-sdl"   description:"Render with sdl"`
	CaptureDevice  string `short:"c" long:"capture-dev" default:"hw:0" description:"Sound capture device name"`
	PlayDevice     string `short:"p" long:"play-dev"    default:"mono" description:"Sound play device name"`
	ModelParamPath string `short:"m" long:"model-param" description:"Path to file containing model parameters" required:"true"`
	KeywordPath    string `short:"k" long:"keyword"     description:"Path to keyword file" required:"true"`
	AwsId          string `short:"a" long:"aws-id"     description:"AWS ID" required:"true"`
	AwsSecret      string `short:"w" long:"aws-secret" description:"AWS key" required:"true"`
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

	if opts.PackedImageDir, err = filepath.Abs(opts.PackedImageDir); err != nil {
		fail(err)
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

func makeSoundPlayer(opts options) *hasp.SoundPlayer {
	player, err := hasp.NewSoundPlayer(opts.PlayDevice, 16000)
	if err != nil {
		fail(err)
	}
	return player
}

func makeCharacter(opts options) *hasp.Character {

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(opts.AwsId, opts.AwsSecret, ""),
		LogLevel:    aws.LogLevel(aws.LogDebugWithRequestRetries),
	})

	if err != nil {
		fmt.Println(sess)
		return nil
	} else {
		fmt.Println("AWS session started...")
	}

	svc := lexruntimeservice.New(sess)

	states := hasp.States{
		"idle": hasp.NewIdleState(
			[]string{"lotus", "reading", "giggles", "reading"},
			time.Duration(10)*time.Second,
			hasp.HotWordDetectorParams{
				CaptureDeviceName: opts.CaptureDevice,
				KeywordPath:       opts.KeywordPath,
				ModelPath:         opts.ModelParamPath,
			},
		),
		"tells-help": hasp.NewTellsHelpState(
			[]string{"tells",},
			"../wavs/hello-help.wav",
		),
		"tells-there": hasp.NewTellsHelpState(
			[]string{"tells",},
			"../wavs/still-there.wav",
		),
		"tells-aws": hasp.NewTellsState(
			[]string{"tells",},
		),
		"listens": hasp.NewListensState(
			[]string{"silent",},
			opts.CaptureDevice,
		),
		"processing": hasp.NewProcessingState(
			[]string{"calls_typing",},
			svc,
		),
	}

	eventDescs := hasp.EventDescs{
		hasp.EventDesc{
			Name: events.HotWordDetectedEventName,
			Src:  []string{"idle"},
			Dst:  "tells-help",
		},
		hasp.EventDesc{
			Name: hasp.SoundPlayedEventName,
			Src:  []string{"tells-help", "tells-aws", "tells-there"},
			Dst:  "listens",
		},
		hasp.EventDesc{
			Name: events.SoundCapturedEventName,
			Src:  []string{"listens"},
			Dst:  "processing",
		},
		hasp.EventDesc{
			Name: events.AwsRepliedEventName,
			Src:  []string{"processing"},
			Dst:  "tells-aws",
		},
		hasp.EventDesc{
			Name: events.SoundEmptyEventName,
			Src:  []string{"listens"},
			Dst:  "tells-there",
		},
	}

	eventSources := events.EventSources{
	}

	animator := makeAnimator(opts)
	soundPlayer := makeSoundPlayer(opts)
	character, err := hasp.NewCharacter("idle", states, eventDescs, eventSources, animator, soundPlayer)
	if err != nil {
		fail(err)
	}

	return character
}

func main() {
	opts := parseCmd()

	character := makeCharacter(opts)
	err := character.Run()
	if err != nil {
		fail(err)
	}
}
