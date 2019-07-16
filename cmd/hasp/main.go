package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/rmcsoft/chanim"
	"github.com/rmcsoft/hasp"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"

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
	AwsID          string `short:"a" long:"aws-id"     description:"AWS ID" required:"true"`
	AwsSecret      string `short:"w" long:"aws-secret" description:"AWS key" required:"true"`

	VisualizeFSM bool `long:"visualize-fsm" description:"Visualize character FSM in Graphviz format (file character.dot)"`
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

func makeSoundPlayer(opts options) *sound.SoundPlayer {
	player, err := sound.NewSoundPlayer(opts.PlayDevice)
	if err != nil {
		fail(err)
	}
	return player
}

func makeHotWordDetector(opts options) *sound.HotWordDetector {
	params := sound.HotWordDetectorParams{
		CaptureDeviceName: opts.CaptureDevice,
		KeywordPath:       opts.KeywordPath,
		ModelPath:         opts.ModelParamPath,
	}

	hotWordDetector, err := sound.NewHotWordDetector(params)
	if err != nil {
		fail(err)
	}

	return hotWordDetector
}

func makeAwsSession(opts options) *session.Session {
	awsSess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(opts.AwsID, opts.AwsSecret, ""),
		LogLevel:    aws.LogLevel(aws.LogDebugWithRequestRetries),
	})

	if err != nil {
		fail(err)
	}

	fmt.Println("AWS session started...")
	return awsSess
}

func loadAudioData(fileName string) *sound.AudioData {
	audioData, err := sound.LoadMonoS16LEFromPCM(fileName, 16000)
	if err != nil {
		fail(err)
	}
	return audioData
}

func makeCharacter(opts options) *hasp.Character {

	awsSess := makeAwsSession(opts)
	svc := lexruntimeservice.New(awsSess)
	hotWordDetector := makeHotWordDetector(opts)

	states := hasp.States{
		"idle": hasp.NewIdleState(
			[]string{"lotus", "reading", "giggles", "reading"},
			time.Duration(10)*time.Second,
			hotWordDetector,
		),
		"tells-help": hasp.NewTellsHelpState(
			[]string{"tells"},
			loadAudioData("../wavs/hello-help.wav"),
		),
		"tells-there": hasp.NewTellsHelpState(
			[]string{"tells"},
			loadAudioData("../wavs/still-there.wav"),
		),
		"tells-aws": hasp.NewTellsState(
			[]string{"tells"},
		),
		"tells-bye": hasp.NewTellsByeState(
			[]string{"tells"},
		),
		"listens": hasp.NewListensState(
			[]string{"silent"},
			hotWordDetector,
		),
		"processing": hasp.NewProcessingState(
			[]string{"SMS"},
			svc,
		),
		"goodbye": hasp.NewSingleAniState(
			"goodbye",
		),
	}

	eventDescs := hasp.EventDescs{
		hasp.EventDesc{
			Name: sound.HotWordDetectedEventName,
			Src:  []string{"idle"},
			Dst:  "tells-help",
		},
		hasp.EventDesc{
			Name: sound.HotWordWithDataDetectedEventName,
			Src:  []string{"idle"},
			Dst:  "processing",
		},
		hasp.EventDesc{
			Name: sound.SoundPlayedEventName,
			Src:  []string{"tells-help", "tells-aws", "tells-there"},
			Dst:  "listens",
		},
		hasp.EventDesc{
			Name: sound.SoundCapturedEventName,
			Src:  []string{"listens"},
			Dst:  "processing",
		},
		hasp.EventDesc{
			Name: events.AwsRepliedEventName,
			Src:  []string{"processing"},
			Dst:  "tells-aws",
		},
		hasp.EventDesc{
			Name: events.StopEventName,
			Src:  []string{"processing"},
			Dst:  "tells-bye",
		},
		hasp.EventDesc{
			Name: sound.SoundPlayedEventName,
			Src:  []string{"tells-bye"},
			Dst:  "goodbye",
		},
		hasp.EventDesc{
			Name: events.StateGoIdleName,
			Src:  []string{"goodbye"},
			Dst:  "idle",
		},
		hasp.EventDesc{
			Name: sound.SoundEmptyEventName,
			Src:  []string{"listens"},
			Dst:  "tells-there",
		},
	}

	eventSources := events.EventSources{}

	animator := makeAnimator(opts)
	soundPlayer := makeSoundPlayer(opts)
	character, err := hasp.NewCharacter("idle", states, eventDescs, eventSources, animator, soundPlayer)
	if err != nil {
		fail(err)
	}

	if opts.VisualizeFSM {
		graphviz := character.Visualize()
		ioutil.WriteFile("character.dot", []byte(graphviz), 0644)
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
