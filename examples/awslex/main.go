package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/lexruntimeservice"
	"github.com/rmcsoft/hasp/sound"
	"golang.org/x/net/context"
	//"github.com/tosone/minimp3"
	//"github.com/twinj/uuid"
)

func makeAwsSession() *lexruntimeservice.Client {

	cfg, err := external.LoadDefaultAWSConfig()

	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	cfg.Region = endpoints.UsEast1RegionID

	awsClient := lexruntimeservice.New(cfg)

	if awsClient == nil {
		log.Fatal("Failed to create AWS Lex client")
	}

	return awsClient
}

func doTheJob(hwd *sound.HotWordDetector, svc *lexruntimeservice.Client, uid string, player *sound.SoundPlayer) {
	soundCapturerEventSource, _ := hwd.StartSoundCapture()
	for event := range soundCapturerEventSource.Events() {
		if event.Name != "SoundEmpty" {
			data, _ := event.Args[0].(sound.SoundCapturedEventData)

			samples := data.AudioData.Samples()

			reader := bytes.NewReader(samples)

			soundCapturerEventSource.Close()

			req := svc.PostContentRequest(&lexruntimeservice.PostContentInput{
				BotAlias:    aws.String("$LATEST"),
				BotName:     aws.String("HASPBot"),
				ContentType: aws.String(data.AudioData.Mime()),
				UserId:      aws.String(uid),
				InputStream: reader,
				Accept:      aws.String("audio/pcm"),
				//Accept:      aws.String("audio/mpeg"),
			})
			fmt.Println("Sending request to runtime.lex")

			resp, err := req.Send(context.TODO())
			if err != nil {
				fmt.Print("Failed to send request to runtime.lex: %v", err)
				return
			}

			log.Println("Response runtime.lex: ", resp)
			if resp.InputTranscript != nil {
				fmt.Println("InputTranscript: ", *resp.InputTranscript)
			}
			if resp.Message != nil {
				fmt.Println("Message: ", *resp.Message)
			}

			if resp.AudioStream == nil {
				fmt.Print("Response from runtime.lex does not contain AudioStream")
				return
			}

			outSamples, err := ioutil.ReadAll(resp.AudioStream)
			if err != nil || len(outSamples) == 0 {
				fmt.Print("!!!!! Unable to read audio data from the runtime.lex response")
				return
			}

			//			dec, pcm, err := minimp3.DecodeFull(outSamples)

			ad := sound.NewMonoS16LE(16000, outSamples)
			player.PlaySync(ad)
		}
	}
}

func main() {
	svc := makeAwsSession()

	hwd, _ := sound.NewHotWordDetector(
		sound.HotWordDetectorParams{
			DebugSound:        true,
			ModelPath:         "porcupine_params.pv",
			KeywordPath:       "alexa_linux.ppn",
			CaptureDeviceName: "default",
			//KeywordPath:       "francesca_beaglebone.ppn",
			//CaptureDeviceName: "hw:0",
		},
	)
	player, _ := sound.NewSoundPlayer("default")

	//uid := uuid.NewV4()
	//doTheJob(hwd, svc, uid.String(), player)
	doTheJob(hwd, svc, "User1", player)
}
