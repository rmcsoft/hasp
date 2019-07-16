package haspaws

import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lexruntimeservice"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
)

type awsLexRuntime struct {
	eventChan  chan *events.Event
	lrs        *lexruntimeservice.LexRuntimeService
	audioData  *sound.AudioData
	sampleRate int
	userId     string
}

// NewLexEventSource creates LexEventSource
func NewLexEventSource(lrs *lexruntimeservice.LexRuntimeService,
	audioData *sound.AudioData, userId string) (events.EventSource, error) {
	h := &awsLexRuntime{
		eventChan: make(chan *events.Event),
		lrs:       lrs,
		audioData: audioData,
		userId:    userId,
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

	fmt.Println("Send!..")
	err := req.Send()
	fmt.Println(resp, err)

	if err != nil || resp.AudioStream == nil {
		fmt.Println(err)
		// h.gotNoReply ?
		return
	}

	outbuf, err := ioutil.ReadAll(resp.AudioStream)
	if err != nil || len(outbuf) == 0 {
		fmt.Println(err)
		return
	}

	r := bytes.NewReader(outbuf)
	frames := make([]int16, len(outbuf)/2)
	err = binary.Read(r, binary.LittleEndian, &frames)
	if err != nil {
		fmt.Println(err)
		return
	}

	if resp.IntentName != nil && *resp.IntentName == "StopIteraction" {
		h.gotStop(frames)
	} else {
		h.gotReply(frames)
	}
}

func (h *awsLexRuntime) gotReply(samples []int16) {
	h.eventChan <- NewAwsRepliedEvent(samples, 16000)
}

func (h *awsLexRuntime) gotStop(samples []int16) {
	h.eventChan <- events.NewStopEvent(samples, 16000)
}

func (h *awsLexRuntime) makeInputStream() io.ReadSeeker {
	samples := h.audioData.Samples()
	reader := bytes.NewReader(samples)
	return aws.ReadSeekCloser(reader)
}
