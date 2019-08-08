package haspaws

import "C"

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lexruntimeservice"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
)

type awsLexRuntime struct {
	eventChan          chan *events.Event
	lrs                *lexruntimeservice.Client
	audioData          *sound.AudioData
	repliedAudioFormat sound.AudioFormat
	userId             string
	debug              bool
}

// NewLexEventSource creates LexEventSource
func NewLexEventSource(lrs *lexruntimeservice.Client,
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

func (h *awsLexRuntime) sendRequest() ([]byte, *lexruntimeservice.PostContentResponse, error) {
	req := h.lrs.PostContentRequest(
		&lexruntimeservice.PostContentInput{
			BotAlias:    aws.String("Prod"),
			BotName:     aws.String("HASPBot"),
			ContentType: aws.String(h.audioData.Mime()),
			UserId:      aws.String(h.userId),
			InputStream: h.makeInputStream(),
			Accept:      aws.String("audio/pcm"),
		})

	log.Debug("Sending request to runtime.lex")
	resp, err := req.Send(context.TODO())
	if err != nil {
		log.Errorf("Failed to send request to runtime.lex: %v", err)

		return nil, nil, fmt.Errorf("Failed to send request to runtime.lex: %v", err)
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
		return nil, nil, fmt.Errorf("Response from runtime.lex does not contain AudioStream")
	}

	samples, err := ioutil.ReadAll(resp.AudioStream)
	if err != nil || len(samples) == 0 {
		log.Errorf("Unable to read audio data from the runtime.lex response")
		return nil, nil, fmt.Errorf("Unable to read audio data from the runtime.lex response")
	}

	if h.debug {
		t := time.Now()
		f, _ := os.Create(fmt.Sprintf("./tmp/%v-got.pcm", t.Format("20060102150405")))
		defer f.Close()
		f.Write(samples)
	}

	return samples, resp, err
}

func (h *awsLexRuntime) run() {
	defer close(h.eventChan)

	samples, resp, err := h.sendRequest()
	if err != nil {
		// NOT-A-FIX! This workaround is here just to understand the problem better!
		log.Info(" ===>>> Error appeared communicating with AWS! RETRYING!!!!!!")
		samples, resp, err = h.sendRequest()
		if err != nil {
			// NOT-A-FIX! This workaround is here just to understand the problem better!
			log.Info(" ===>>> Error appeared AGAIN communicating with AWS! RETRYING ONCE AGAIN!!!!!!")
			samples, resp, err = h.sendRequest()
			if err != nil {
				log.Error(" ===>>> 3 errors already!!! Giving up")
				h.gotStop(nil) // TODO: Reaction to an error
				return
			}
		}
	}
	repliedSpeech := sound.NewAudioData(h.repliedAudioFormat, samples)

	if resp.IntentName == nil {
		log.Debug("GOT: EMPTY Intent; State=", resp.DialogState)
		h.gotReply(repliedSpeech)
		return
	}

	log.Debug("GOT: Intent=", *resp.IntentName, "; State=", resp.DialogState)
	switch *resp.IntentName {
	case "StopInteraction", "NoThankYou":
		log.Debug("stopping...")
		h.gotStop(repliedSpeech)
	case "Hell":
		log.Debug("stopping...")
		h.gotStop(nil)
	case "AxeOso", "Catawba", "Codescape", "DontKnowTheLastName", "Event", "Goodbye", "ThankYou", "TourSubscription", "TradeLore":
		if resp.DialogState == "Fulfilled" {
			log.Debug("stopping...")
			h.gotStop(repliedSpeech)
		} else {
			log.Debug("reply...")
			h.gotReply(repliedSpeech)
		}
	case "ContactAdvent", "HowCanIhelpYou", "Delivery", "Chatter", "NoNameMeeting", "NoNameDelivery", "RepeatPhoneNumber", "SmthUnclear", "Mistake", "WebsitePhoneNumber", "WhatIsYourName":
		log.Debug("reply...")
		h.gotReply(repliedSpeech)

	case "Meeting":
		if resp.DialogState == "ConfirmIntent" {
			log.Debug("meeting confirmation...")
			h.gotConfirmation(repliedSpeech)
		} else if resp.DialogState == "Fulfilled" {
			log.Debug("meeting fullfilled...")
			h.gotCall(repliedSpeech)
		} else {
			log.Debug("reply...")
			h.gotReply(repliedSpeech)
		}
	default:
		log.Debug("reply...")
		h.gotReply(repliedSpeech)
	}
}

func (h *awsLexRuntime) gotReply(data *sound.AudioData) {
	h.eventChan <- NewAwsRepliedEvent(data)
}

func (h *awsLexRuntime) gotStop(repliedSpeech *sound.AudioData) {
	h.eventChan <- sound.NewStopEvent(repliedSpeech)
}

func (h *awsLexRuntime) gotCall(data *sound.AudioData) {
	h.eventChan <- NewAwsRepliedEventState(data, AwsRepliedCallEventName)
}

func (h *awsLexRuntime) gotConfirmation(data *sound.AudioData) {
	h.eventChan <- NewAwsRepliedEventState(data, AwsRepliedTypeEventName)
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
