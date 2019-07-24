package haspaws

import "C"

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lexruntimeservice"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
)

type awsLexRuntime struct {
	eventChan          chan *events.Event
	lrs                *lexruntimeservice.LexRuntimeService
	audioData          *sound.AudioData
	repliedAudioFormat sound.AudioFormat
	userId             string
	debug              bool
}

// NewLexEventSource creates LexEventSource
func NewLexEventSource(lrs *lexruntimeservice.LexRuntimeService,
	audioData *sound.AudioData, userId string, debug bool) (events.EventSource, error) {
	h := &awsLexRuntime{
		eventChan:          make(chan *events.Event),
		lrs:                lrs,
		audioData:          audioData,
		userId:             userId,
		repliedAudioFormat: audioData.Format(),
		debug:              debug,
	}

	go h.run()
	return h, nil
}

func (h *awsLexRuntime) Name() string {
	return "AwsLexRuntime"
}

func (h *awsLexRuntime) Events() chan *events.Event {
	return h.eventChan
}

func (h *awsLexRuntime) Close() {
}

func (h *awsLexRuntime) sendRequest() ([]byte, string, error) {
	req, resp := h.lrs.PostContentRequest(&lexruntimeservice.PostContentInput{
		BotAlias:    aws.String("Prod"),
		BotName:     aws.String("HASPBot"),
		ContentType: aws.String(h.audioData.Mime()),
		UserId:      aws.String(h.userId),
		InputStream: h.makeInputStream(),
		Accept:      aws.String("audio/pcm"),
	})

	log.Debug("Send request to runtime.lex")
	err := req.Send()

	if err != nil {
		log.Errorf("Failed to send request to runtime.lex: %v", err)

		return nil, "Error", fmt.Errorf("Failed to send request to runtime.lex: %v", err)
	}

	log.Trace("Response runtime.lex: %v", resp)
	if resp.InputTranscript != nil {
		log.Infof("InputTranscript: ", *resp.InputTranscript)
	}
	if resp.Message != nil {
		log.Infof("Message: ", *resp.Message)
	}

	if resp.AudioStream == nil {
		log.Errorf("Response from runtime.lex does not contain AudioStream")
		return nil, "Error", fmt.Errorf("Response from runtime.lex does not contain AudioStream")
	}

	samples, err := ioutil.ReadAll(resp.AudioStream)
	if err != nil || len(samples) == 0 {
		log.Errorf("Unable to read audio data from the runtime.lex response")
		return nil, "Error", fmt.Errorf("Unable to read audio data from the runtime.lex response")
	}

	if h.debug {
		t := time.Now()
		f, _ := os.Create(fmt.Sprintf("./tmp/%v-got.pcm", t.Format("20060102150405")))
		defer f.Close()
		f.Write(samples)
	}

	intentName := "Error"
	if resp.IntentName != nil {
		intentName = *resp.IntentName
	}

	return samples, intentName, err
}

func (h *awsLexRuntime) run() {
	defer close(h.eventChan)

	samples, intentName, err := h.sendRequest()
	if err != nil {
		// NOT-A-FIX! This workaround is here just to understand the problem better!
		log.Info(" ===>>> Error appeared communicating with AWS! RETRYING!!!!!!")
		samples, intentName, err = h.sendRequest()
		if err != nil {
			// NOT-A-FIX! This workaround is here just to understand the problem better!
			log.Info(" ===>>> Error appeared AGAIN communicating with AWS! RETRYING ONCE AGAIN!!!!!!")
			samples, intentName, err = h.sendRequest()
			if err != nil {
				log.Error(" ===>>> 3 errors already!!! Giving up")
				h.gotStop(nil) // TODO: Reaction to an error
				return
			}
		}
	}
	repliedSpeech := sound.NewAudioData(h.repliedAudioFormat, samples)

	switch intentName {
	case "StopInteraction", "NoThankYou":
		h.gotStop(repliedSpeech)
	default:
		h.gotReply(repliedSpeech)
	}
}

func (h *awsLexRuntime) gotReply(replaiedSpeech *sound.AudioData) {
	h.eventChan <- NewAwsRepliedEvent(replaiedSpeech)
}

func (h *awsLexRuntime) gotStop(replaiedSpeech *sound.AudioData) {
	h.eventChan <- sound.NewStopEvent(replaiedSpeech)
}

func (h *awsLexRuntime) makeInputStream() io.ReadSeeker {
	samples := h.audioData.Samples()

	if h.debug {
		t := time.Now()
		f, _ := os.Create(fmt.Sprintf("./tmp/%v-sent.pcm", t.Format("20060102150405")))
		defer f.Close()
		f.Write(samples)
	}

	reader := bytes.NewReader(samples)
	return aws.ReadSeekCloser(reader)
}
