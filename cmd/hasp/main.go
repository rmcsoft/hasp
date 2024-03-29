package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/krig/go-sox"
	"image"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/rmcsoft/chanim"
	"github.com/rmcsoft/hasp"
	atmel "github.com/rmcsoft/hasp/atmel/periph_gpio"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/haspaws"
	"github.com/rmcsoft/hasp/sound"

	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/lexruntimeservice"
	"github.com/jessevdk/go-flags"
)

const (
	pixFormat = chanim.RGB16
)

type options struct {
	PackedImageDir  string `short:"i" long:"image-dir"   description:"Packed image directory needed for animation" required:"true"`
	UseSDL          bool   `short:"s" long:"use-sdl"     description:"Render with sdl"`
	CaptureDevice   string `short:"c" long:"capture-dev" default:"hw:0" description:"Sound capture device name"`
	PlayDevice      string `short:"p" long:"play-dev"    default:"mono" description:"Sound play device name"`
	ModelParamPath  string `short:"m" long:"model-param" description:"Path to file containing model parameters" required:"true"`
	KeywordPath     string `short:"k" long:"keyword"     description:"Path to keyword file" required:"true"`
	LeftSensorPin   int    `long:"left-pin"              description:"Left sensor pin" required:"true"`
	LeftSensorPort  string `long:"left-port"             description:"Left sensor port" required:"true"`
	RightSensorPin  int    `long:"right-pin"             description:"Right sensor pin" required:"true"`
	RightSensorPort string `long:"right-port"            description:"Right sensor port" required:"true"`
	AwsID           string `short:"a" long:"aws-id"      description:"AWS ID" required:"true"`
	AwsSecret       string `short:"w" long:"aws-secret"  description:"AWS key" required:"true"`

	Debug bool `long:"debug" description:"debug information in log outputs"`
	Trace bool `long:"trace" description:"trace-level debugging in log outputs"`

	VisualizeFSM bool `long:"visualize-fsm" description:"Visualize character FSM in Graphviz format (file character.dot)"`

	SplashScreenPath string `long:"splash-screen" description:"Image for splash screen (ppixmap format)"`

	Config func(s string) error `long:"config" no-ini:"true"`
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
		DebugSound:        opts.Trace,
	}

	hotWordDetector, err := sound.NewHotWordDetector(params)
	if err != nil {
		log.Fatal(err)
	}

	return hotWordDetector
}

func makeAwsSession(opts options) *lexruntimeservice.Client {

	cfg, err := external.LoadDefaultAWSConfig()

	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	cfg.Region = endpoints.UsEast1RegionID
	cfg.Logger = logrusProxy{}
	cfg.LogLevel = aws.LogDebug

	awsClient := lexruntimeservice.New(cfg)

	if awsClient == nil {
		log.Fatal("Failed to create AWS Lex client")
	}

	log.Info("AWS session started...")
	return awsClient
}

func loadAudioData(fileName string) *sound.AudioData {
	audioData, err := sound.LoadMonoS16LEFromPCM(fileName, 16000)
	if err != nil {
		log.Fatal(err)
	}
	return audioData
}

func makeCharacter(opts options) *hasp.Character {

	svc := makeAwsSession(opts)
	soundPlayer := makeSoundPlayer(opts)
	hotWordDetector := makeHotWordDetector(opts)

	inSound := loadAudioData("../wavs/bing-bong.wav")
	outSound := loadAudioData("../wavs/bong-bing.wav")

	states := hasp.States{
		"idle": hasp.NewIdleState(
			[]string{"lotus", "reading", "giggles", "reading"},
			time.Duration(2)*time.Minute,
			hotWordDetector,
			atmel.AtmelGpioPins{
				atmel.AtmelGpioPin{Number: opts.LeftSensorPin, Name: opts.LeftSensorPort},
				atmel.AtmelGpioPin{Number: opts.RightSensorPin, Name: opts.RightSensorPort},
			},
		),
		"sensor-triggered": hasp.NewTriggeredState(
			"silent",
			hotWordDetector,
			atmel.AtmelGpioPins{
				atmel.AtmelGpioPin{Number: opts.LeftSensorPin, Name: opts.LeftSensorPort},
				atmel.AtmelGpioPin{Number: opts.RightSensorPin, Name: opts.RightSensorPort},
			},
			10*time.Second,
		),
		"tells-fullhelp": hasp.NewTellsHelpState(
			[]string{"tells"},
			loadAudioData("../wavs/fullhelp.wav"),
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
			soundPlayer,
			inSound,
			outSound,
		),
		"processing": hasp.NewProcessingState(
			[]string{"silent"},
			svc,
			opts.Debug || opts.Trace,
		),
		"goodbye": hasp.NewSingleAniState(
			"goodbye",
		),
		"call": hasp.NewTellsState(
			[]string{"calls2"},
		),
		"tell-type": hasp.NewTellsState(
			[]string{"tells"},
		),
		"type": hasp.NewListensState(
			[]string{"SMS"},
			hotWordDetector,
			soundPlayer,
			inSound,
			outSound,
		),
		"tell-msg-sent": hasp.NewTellsHelpState(
			[]string{"tells"},
			loadAudioData("../wavs/msg-sent.wav"),
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
			Name: events.GpioEventName,
			Src:  []string{"idle"},
			Dst:  "sensor-triggered",
		},
		hasp.EventDesc{
			Name: events.StateWaitTimeoutName,
			Src:  []string{"sensor-triggered"},
			Dst:  "idle",
		},
		hasp.EventDesc{
			Name: events.StateFullHelpName,
			Src:  []string{"sensor-triggered"},
			Dst:  "tells-fullhelp",
		},
		hasp.EventDesc{
			Name: sound.HotWordDetectedEventName,
			Src:  []string{"sensor-triggered"},
			Dst:  "tells-help",
		},
		hasp.EventDesc{
			Name: sound.HotWordWithDataDetectedEventName,
			Src:  []string{"sensor-triggered"},
			Dst:  "processing",
		},

		hasp.EventDesc{
			Name: sound.SoundPlayedEventName,
			Src:  []string{"tells-help", "tells-aws", "tells-there", "tells-fullhelp"},
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
			Name: haspaws.AwsRepliedTypeEventName,
			Src:  []string{"processing"},
			Dst:  "tell-type",
		},
		hasp.EventDesc{
			Name: sound.SoundCapturedEventName,
			Src:  []string{"type"},
			Dst:  "processing",
		},
		hasp.EventDesc{
			Name: haspaws.AwsRepliedCallEventName,
			Src:  []string{"processing"},
			Dst:  "call",
		},
		hasp.EventDesc{
			Name: sound.SoundPlayedEventName,
			Src:  []string{"call"},
			Dst:  "tell-msg-sent",
		},
		hasp.EventDesc{
			Name: sound.SoundPlayedEventName,
			Src:  []string{"tell-msg-sent"},
			Dst:  "idle",
		},
		hasp.EventDesc{
			Name: sound.StopEventName,
			Src:  []string{"processing"},
			Dst:  "tells-bye",
		},
		hasp.EventDesc{
			Name: sound.SoundPlayedEventName,
			Src:  []string{"tell-type"},
			Dst:  "type",
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
		hasp.EventDesc{
			Name: sound.SoundEmptyEventName,
			Src:  []string{"type"},
			Dst:  "tells-there",
		},
	}

	eventSources := events.EventSources{}

	animator := makeAnimator(opts)
	character, err := hasp.NewCharacter("idle", states, eventDescs, eventSources, animator, soundPlayer)
	if err != nil {
		log.Fatal(err)
	}

	if opts.Debug || opts.Trace {
		character.SetDebug(true)
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
		os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	opts := parseOpts()
	if opts.Trace {
		log.Info("Tracing enabled!")
		log.SetLevel(log.TraceLevel)
	} else if opts.Debug {
		log.SetLevel(log.DebugLevel)
	}

	// All libSoX applications must start by initializing the SoX library
	if !sox.Init() {
		log.Fatal("Failed to initialize SoX")
	}
	// Make sure to call Quit before terminating
	defer sox.Quit()

	character := makeCharacter(opts)
	err = character.Run()
	if err != nil {
		log.Fatal(err)
	}
}
