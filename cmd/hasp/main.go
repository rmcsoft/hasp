package main

import (
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/rmcsoft/chanim"
	"github.com/rmcsoft/hasp"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/haspaws"
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

	SplashScreenPath string `long:"splash-screen" description:"Image for splash screen (ppixmap format)"`

	Config func(s string) error `long:"config" no-ini:"true"`
	Debug bool `long:"debug" description:"debug information in log outputs"`
	Trace bool `long:"trace" description:"trace-level debugging in log outputs"`
}

type logrusProxy struct {
}

// Log is a utility function to comply with the AWS signature
func (logrusProxy) Log(args ...interface{}) {
	log.Info(args...)
}

func parseOpts() options {
	log.Info("Parsing command line arguments")

	var opts options
	var cmdParser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)

	opts.Config = func(filename string) error {
		log.Infof("Parsing configuration file: '%s'", filename)
		iniParser := flags.NewIniParser(cmdParser)
		iniParser.ParseAsDefaults = false
		return iniParser.ParseFile(filename)
	}

	_, err := cmdParser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			fmt.Println(flagsErr)
			os.Exit(0)
		}
		log.Fatal(err)
	}

	if opts.PackedImageDir, err = filepath.Abs(opts.PackedImageDir); err != nil {
		log.Fatal(err)
	}

	return opts
}

func showSplashScreen(opts options, paintEngine chanim.PaintEngine) {
	if len(opts.SplashScreenPath) == 0 {
		log.Warn("Splash screen is not set")
		return
	}

	splashScreen, err := chanim.LoadPackedPixmap(opts.SplashScreenPath)
	if err != nil {
		log.Errorf("Unable to load splash screen from '%s': %v", opts.SplashScreenPath, err)
		return
	}

	log.Debug("Show splash screen")
	top := image.Point{
		X: 0,
		Y: 0,
	}

	err = paintEngine.Begin()
	if err != nil {
		log.Errorf("Failed to init show splash screen: %v", err)
	}

	if err := paintEngine.Begin(); err != nil {
		log.Errorf("Failed to show splash screen: %v", err)
	}
	if err := paintEngine.Clear(image.Rect(0, 0, paintEngine.GetWidth(), paintEngine.GetHeight())); err != nil {
		log.Errorf("Failed to show splash screen: %v", err)
	}
	if err := paintEngine.DrawPackedPixmap(top, splashScreen); err != nil {
		log.Errorf("Failed to show splash screen: %v", err)
	}
	if err := paintEngine.End(); err != nil {
		log.Errorf("Failed to show splash screen: %v", err)
	}
	err = paintEngine.End()
	if err != nil {
		log.Errorf("Failed to end show splash screen: %v", err)
	}
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
		log.Fatal(err)
	}
	showSplashScreen(opts, paintEngine)
	return paintEngine
}

func makeAnimator(opts options) *chanim.Animator {
	log.Debug("Making paint engine")
	paintEngine := makePaintEngine(opts)
	log.Debug("Creating animator")
	animator, err := hasp.CreateAnimator(paintEngine, opts.PackedImageDir)
	if err != nil {
		log.Fatal(err)
	}
	return animator
}

func makeSoundPlayer(opts options) *sound.SoundPlayer {
	player, err := sound.NewSoundPlayer(opts.PlayDevice)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	return hotWordDetector
}

func makeAwsSession(opts options) *session.Session {
	awsSess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(opts.AwsID, opts.AwsSecret, ""),
		LogLevel:    aws.LogLevel(aws.LogDebugWithRequestErrors),
		Logger:      logrusProxy{},
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Info("AWS session started...")
	return awsSess
}

func loadAudioData(fileName string) *sound.AudioData {
	audioData, err := sound.LoadMonoS16LEFromPCM(fileName, 16000)
	if err != nil {
		log.Fatal(err)
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
			time.Duration(2)*time.Minute,
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
			Name: sound.StopEventName,
			Src:  []string{"listens"},
			Dst:  "idle",
		},
		hasp.EventDesc{
			Name: haspaws.AwsRepliedEventName,
			Src:  []string{"processing"},
			Dst:  "tells-aws",
		},
		hasp.EventDesc{
			Name: sound.StopEventName,
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
		log.Fatal(err)
	}

	if opts.VisualizeFSM {
		graphviz := character.Visualize()
		ioutil.WriteFile("character.dot", []byte(graphviz), 0644)
	}

	return character
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.999",
	})
	err := os.Mkdir("./logs", 0777)
	if err != nil {
		//ignore
	}
	f, err := os.OpenFile(fmt.Sprintf("./logs/start-%v.log", time.Now().Format("20060102150405")),
		os.O_WRONLY | os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	opts := parseOpts()
	if opts.Trace {
		log.SetLevel(log.TraceLevel)
	} else if opts.Debug {
		log.SetLevel(log.DebugLevel)
	}

	character := makeCharacter(opts)
	err = character.Run()
	if err != nil {
		log.Fatal(err)
	}
}
