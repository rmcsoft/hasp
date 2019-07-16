package haspaws

import "C"

import (
	"bytes"
	"io"
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lexruntimeservice"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
)

type awsLexRuntime struct {
	eventChan           chan *events.Event
	lrs                 *lexruntimeservice.LexRuntimeService
	audioData           *sound.AudioData
	replaiedAudioFormat sound.AudioFormat
	userId              string
}

// NewLexEventSource creates LexEventSource
func NewLexEventSource(lrs *lexruntimeservice.LexRuntimeService,
	audioData *sound.AudioData, userId string) (events.EventSource, error) {
	h := &awsLexRuntime{
		eventChan:           make(chan *events.Event),
		lrs:                 lrs,
		audioData:           audioData,
		userId:              userId,
		replaiedAudioFormat: audioData.Format(),
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

func (h *awsLexRuntime) run() {
	defer close(h.eventChan)

	req, resp := h.lrs.PostContentRequest(&lexruntimeservice.PostContentInput{
		BotAlias:    aws.String("Prod"),
		BotName:     aws.String("HASPBot"),
		ContentType: aws.String(h.audioData.Mime()),
		UserId:      aws.String(h.userId),
		InputStream: h.makeInputStream(),
		Accept:      aws.String("audio/pcm"),
	})

	log.Infof("Send request to runtime.lex")
	err := req.Send()
	if err != nil {
		// TODO: Reaction to an error
		log.Errorf("Failed to send request to runtime.lex: %v", err)
		return
	}

	log.Infof("Response runtime.lex: %v", resp)

	if resp.AudioStream == nil {
		// TODO: Reaction to an error, h.gotNoReply ?
		log.Errorf("Response from runtime.lex does not contain AudioStream")
		return
	}

	samples, err := ioutil.ReadAll(resp.AudioStream)
	if err != nil || len(samples) == 0 {
		// TODO: Reaction to an error
		log.Errorf("Unable to read audio data from the runtime.lex response")
		return
	}
	replaiedSpeech := sound.NewAudioData(h.replaiedAudioFormat, samples)

	if resp.IntentName != nil && *resp.IntentName == "StopIteraction" {
		h.gotStop(replaiedSpeech)
	} else {
		h.gotReply(replaiedSpeech)
	}
}

func (h *awsLexRuntime) gotReply(replaiedSpeech *sound.AudioData) {
	h.eventChan <- NewAwsRepliedEvent(replaiedSpeech)
}

func (h *awsLexRuntime) gotStop(replaiedSpeech *sound.AudioData) {
	h.eventChan <- NewStopEvent(replaiedSpeech)
}

func (h *awsLexRuntime) makeInputStream() io.ReadSeeker {
	samples := h.audioData.Samples()
	reader := bytes.NewReader(samples)
	return aws.ReadSeekCloser(reader)
}
