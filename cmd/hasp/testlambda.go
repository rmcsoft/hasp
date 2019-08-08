package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/polly"
)

type MyEvent struct {
	CurrentIntent struct {
		Name  string `json:"name"`
		Slots struct {
			Slot string `json:"slot"`
		} `json:"slots"`
		SlotDetails struct {
			Slot struct {
				Resolutions []struct {
					Value string `json:"value"`
				} `json:"resolutions"`
				OriginalValue string `json:"originalValue"`
			} `json:"slot"`
		} `json:"slotDetails"`
		ConfirmationStatus string `json:"confirmationStatus"`
	} `json:"currentIntent"`
	Bot struct {
		Name    string `json:"name"`
		Alias   string `json:"alias"`
		Version string `json:"version"`
	} `json:"bot"`
	UserID            string `json:"userId"`
	InputTranscript   string `json:"inputTranscript"`
	InvocationSource  string `json:"invocationSource"`
	OutputDialogMode  string `json:"outputDialogMode"`
	MessageVersion    string `json:"messageVersion"`
	SessionAttributes map[string]interface{} `json:"sessionAttributes"`
	RequestAttributes struct {
		Key string `json:"key"`
	} `json:"requestAttributes"`
}

type IntentResponse struct {
	SessionAttributes map[string]interface{} `json:"sessionAttributes"`
	DialogAction struct {
		Type string `json:"type"`
		FulfillmentState string `json:"fulfillmentState"`
//		IntentName string `json:"intentName""`
	} `json:"dialogAction"`
}

func HandleLambdaEvent(event MyEvent) (IntentResponse, error) {
	/*
	`{
	  "sessionAttributes": {},
	  "dialogAction": {
	    "type": "Close",
	    "fulfillmentState": "Fulfilled",
	    "message": {
	      "contentType": "PlainText",
	      "content": "Okay, I have ordered your large meat pizza on thin crust."
	    }
	}`	*/

	resp := IntentResponse{}
	if event.SessionAttributes != nil {
		resp.SessionAttributes = event.SessionAttributes
	} else {
		resp.SessionAttributes = make(map[string] interface{})
	}
	resp.SessionAttributes["LMessage"] = fmt.Sprintf("Hello from lambda %v!", event.CurrentIntent.Name)
	resp.DialogAction.Type = "Close"
	resp.DialogAction.FulfillmentState = "Fulfilled"

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := polly.New(sess)
	input := &polly.SynthesizeSpeechInput{OutputFormat: aws.String("pcm"), Text: aws.String(resp.SessionAttributes["LMessage"].(string)), VoiceId: aws.String("Joanna")}
	output, err := svc.SynthesizeSpeech(input)
	if err != nil {
		fmt.Println("Got error calling SynthesizeSpeech:")
		fmt.Print(err.Error())
	} else {
		aStream, _ := ioutil.ReadAll(output.AudioStream)
		resp.SessionAttributes["BMessage"] = base64.StdEncoding.EncodeToString(aStream)
	}

	return resp, nil
}

func main() {
	lambda.Start(HandleLambdaEvent)
}
