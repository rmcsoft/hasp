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
	eventChan          chan *events.Event
	lrs                *lexruntimeservice.LexRuntimeService
	audioData          *sound.AudioData
	repliedAudioFormat sound.AudioFormat
	userId             string
}

// NewLexEventSource creates LexEventSource
func NewLexEventSource(lrs *lexruntimeservice.LexRuntimeService,
	audioData *sound.AudioData, userId string) (events.EventSource, error) {
	h := &awsLexRuntime{
		eventChan:          make(chan *events.Event),
		lrs:                lrs,
		audioData:          audioData,
		userId:             userId,
		repliedAudioFormat: audioData.Format(),
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

	log.Debug("Send request to runtime.lex")
	err := req.Send()

	if err != nil {
		h.gotStop(nil) // TODO: Reaction to an error
		log.Errorf("Failed to send request to runtime.lex: %v", err)

		return
	}

	log.Trace("Response runtime.lex: %v", resp)
	if resp.InputTranscript != nil {
		log.Infof("InputTranscript: ", *resp.InputTranscript)
	}
	if resp.Message != nil {
		log.Infof("Message: ", *resp.Message)
	}

	if resp.AudioStream == nil {
		h.gotStop(nil) // TODO: Reaction to an error, h.gotNoReply ?
		log.Errorf("Response from runtime.lex does not contain AudioStream")
		return
	}

	samples, err := ioutil.ReadAll(resp.AudioStream)
	if err != nil || len(samples) == 0 {
		h.gotStop(nil) // TODO: Reaction to an error
		log.Errorf("Unable to read audio data from the runtime.lex response")
		return
	}
	repliedSpeech := sound.NewAudioData(h.repliedAudioFormat, samples)

	if resp.IntentName != nil &&
		(*resp.IntentName == "StopInteraction" || *resp.IntentName == "NoThankYou") {
		h.gotStop(repliedSpeech)
	} else {
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
/*
	t := time.Now()
	f, _ := os.Create(fmt.Sprintf("./tmp/data-%v.pcm", t.Format("20060102150405")))
	defer f.Close()
	f.Write(samples)
	*/

	reader := bytes.NewReader(samples)
	return aws.ReadSeekCloser(reader)
}
