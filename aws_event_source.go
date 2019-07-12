package hasp

import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"unsafe"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lexruntimeservice"
	"github.com/rmcsoft/hasp/events"
)

type awsLexRuntime struct {
	eventChan  chan *events.Event
	lrs        *lexruntimeservice.LexRuntimeService
	data       events.SoundCapturedEventData
	sampleRate int
}

// NewLexEventSource creates LexEventSource
func NewLexEventSource(lrs *lexruntimeservice.LexRuntimeService, data events.SoundCapturedEventData) (events.EventSource, error) {
	h := &awsLexRuntime{
		eventChan:  make(chan *events.Event),
		lrs:        lrs,
		data:       data,
		sampleRate: 16000,
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
	req, resp := h.lrs.PostContentRequest(&lexruntimeservice.PostContentInput{
		BotAlias:    aws.String("Prod"),
		BotName:     aws.String("HASPBot"),
		ContentType: aws.String("audio/l16; rate=16000; channels=1"),
		UserId:      aws.String("go_user1"),
		InputStream: aws.ReadSeekCloser(h.createReaderForSample()),
		Accept:      aws.String("audio/pcm"),
	})

	fmt.Println("Send!..")
	err := req.Send()
	fmt.Println(resp, err)

	outbuf, err := ioutil.ReadAll(resp.AudioStream)
	if err != nil {
		fmt.Println(err)
		return
	}

	r := bytes.NewReader(outbuf)
	frames := make([]int16, len(outbuf)/2)
	binary.Read(r, binary.LittleEndian, &frames)

	h.gotReply(frames)
}

func (h *awsLexRuntime) gotReply(samples []int16) {
	h.eventChan <- events.NewAwsRepliedEvent(samples, h.sampleRate)
}

func (h *awsLexRuntime) createReaderForSample() io.Reader {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&h.data.Samples))
	var buf []byte
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	bh.Data = sh.Data
	bh.Cap = sh.Cap * 2
	bh.Len = sh.Len * 2

	return bytes.NewBuffer(buf)
}
