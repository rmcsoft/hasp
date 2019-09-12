package main

import (
	"context"
	"fmt"
	"os"
	"time"

	dialogflow "cloud.google.com/go/dialogflow/apiv2"
	"github.com/rmcsoft/hasp/sound"
	"github.com/twinj/uuid"
	//"google.golang.org/api/iterator"
	dialogflowpb "google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

func doTheJob(hwd *sound.HotWordDetector, sessionID string, player *sound.SoundPlayer) {
	projectID := "test-kuxabp"

	soundCapturerEventSource, _ := hwd.StartSoundCapture()
	for event := range soundCapturerEventSource.Events() {
		if event.Name != "SoundEmpty" {
			data, _ := event.Args[0].(sound.SoundCapturedEventData)
			audioBytes := data.AudioData.Samples()

			//audioBytes, err := ioutil.ReadFile("./tmp/test.wav")

			ctx := context.Background()

			sessionClient, err := dialogflow.NewSessionsClient(ctx)
			if err != nil {
				return
			}
			defer sessionClient.Close()

			// In this example, we hard code the encoding and sample rate for simplicity.
			audioConfig := dialogflowpb.InputAudioConfig{
				AudioEncoding:   dialogflowpb.AudioEncoding_AUDIO_ENCODING_LINEAR_16,
				SampleRateHertz: 16000,
				LanguageCode:    "en",
			}
			voice := dialogflowpb.VoiceSelectionParams{
				Name:       "en-US-Standard-E",
				SsmlGender: dialogflowpb.SsmlVoiceGender_SSML_VOICE_GENDER_FEMALE,
			}
			synthCfg := dialogflowpb.SynthesizeSpeechConfig{
				Voice:        &voice,
				Pitch:        4,
				SpeakingRate: 1.15,
			}
			outputAudioConfig := dialogflowpb.OutputAudioConfig{
				AudioEncoding:          dialogflowpb.OutputAudioEncoding_OUTPUT_AUDIO_ENCODING_LINEAR_16,
				SampleRateHertz:        16000,
				SynthesizeSpeechConfig: &synthCfg,
			}
			queryAudioInput := dialogflowpb.QueryInput_AudioConfig{
				AudioConfig: &audioConfig,
			}
			queryInput := dialogflowpb.QueryInput{
				Input: &queryAudioInput,
			}

			sessionPath := fmt.Sprintf("projects/%s/agent/sessions/%s", projectID, sessionID)
			request := dialogflowpb.DetectIntentRequest{
				Session:           sessionPath,
				QueryInput:        &queryInput,
				InputAudio:        audioBytes,
				OutputAudioConfig: &outputAudioConfig,
			}

			response, err := sessionClient.DetectIntent(ctx, &request)
			if err != nil {
				return
			}

			//			queryResult := response.GetQueryResult()
			buffer := response.OutputAudio

			if err != nil || len(buffer) == 0 {
				fmt.Print("!!!!! Unable to read audio data from the google dialog-flow response")
				return
			}
			ad := sound.NewMonoS16LE(16000, buffer)

			{
				t := time.Now()
				f, _ := os.Create(fmt.Sprintf("./tmp/%v-GOOG.pcm", t.Format("20060102150405")))
				f.Write(buffer)
				f.Close()
			}

			player.PlaySync(ad)
		}
	}
}

func main() {
	uid := uuid.NewV4()

	hwd, _ := sound.NewHotWordDetector(
		sound.HotWordDetectorParams{
			DebugSound:        true,
			ModelPath:         "porcupine_params.pv",
			KeywordPath:       "alexa_linux.ppn",
			CaptureDeviceName: "default",
		},
	)
	player, _ := sound.NewSoundPlayer("default")

	doTheJob(hwd, uid.String(), player)
}
