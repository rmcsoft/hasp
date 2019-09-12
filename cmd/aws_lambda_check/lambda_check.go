package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

func getCoworkersJson() ([]byte, error) {
	bucketName := "rmc-haspbot"
	keyName := "CoworkersCache"

	region := "us-east-1"
	downloader := s3manager.NewDownloader(session.New(&aws.Config{Region: &region}))

	buff := &aws.WriteAtBuffer{}
	result, err := downloader.Download(buff, &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &keyName,
	})
	if err != nil || result == 0 {
		return nil, err
	}

	return buff.Bytes(), nil
}

type Coworker struct {
	Email       string
	FullName    string
	MobilePhone string
	CompanyName string
}

func getClosestCoworker(body []byte, toFind string) *Coworker {
	coworkers := make([]Coworker, 1000)
	err := json.Unmarshal(body, &coworkers)
	if err != nil {
		fmt.Println("Failed to parse data. ", err)
		return nil
	}

	bestFound := Coworker{}
	bestRank := -1
	for _, v := range coworkers {
		rank := fuzzy.LevenshteinDistance(v.FullName, toFind)

		if bestRank < 0 || rank < bestRank || (rank == bestRank && len(bestFound.MobilePhone) == 0) {
			bestRank = rank
			bestFound = v
		}
	}

	if bestRank < 0 {
		return nil
	} else {
		return &bestFound
	}
}

func getClosestCompany(body []byte, toFind string) *Coworker {
	coworkers := make([]Coworker, 1000)
	err := json.Unmarshal(body, &coworkers)
	if err != nil {
		fmt.Println("Failed to parse data. ", err)
		return nil
	}

	bestFound := Coworker{}
	bestRank := -1
	for _, v := range coworkers {
		if len(v.CompanyName) > 0 {
			rank := fuzzy.LevenshteinDistance(v.CompanyName, toFind)

			if bestRank < 0 || rank < bestRank || (rank == bestRank && len(bestFound.MobilePhone) == 0) {
				bestRank = rank
				bestFound = v
			}
		}
	}

	if bestRank < 0 {
		return nil
	} else {
		return &bestFound
	}
}

func processCoworker(event events.LexEvent) (*events.LexResponse, error) {
	if event.CurrentIntent.ConfirmationStatus == "Denied" {
		fmt.Println("Denied!")
		resp := events.LexResponse{}
		resp.DialogAction.IntentName = event.CurrentIntent.Name
		resp.DialogAction.SlotToElicit = "CoworkerFirstName"
		resp.DialogAction.Type = "ElicitSlot"
		fmt.Println(resp)
		return &resp, nil
	}

	nameFirst, ok := event.CurrentIntent.Slots["CoworkerFirstName"]
	if !ok {
		resp := events.LexResponse{}
		resp.DialogAction.IntentName = event.CurrentIntent.Name
		resp.DialogAction.SlotToElicit = "CoworkerFirstName"
		resp.DialogAction.Slots = event.CurrentIntent.Slots
		resp.DialogAction.Type = "ElicitSlot"
		fmt.Println(resp)
		return &resp, nil
	}

	nameLast, ok := event.CurrentIntent.Slots["CoworkerLastName"]
	if !ok {
		resp := events.LexResponse{}
		resp.DialogAction.IntentName = event.CurrentIntent.Name
		resp.DialogAction.SlotToElicit = "CoworkerLastName"
		resp.DialogAction.Slots = event.CurrentIntent.Slots
		resp.DialogAction.Type = "ElicitSlot"
		fmt.Println(resp)
		return &resp, nil
	}

	if event.CurrentIntent.ConfirmationStatus == "Confirmed" {
		resp := events.LexResponse{}
		resp.DialogAction.Type = "Close"
		resp.DialogAction.FulfillmentState = "Fulfilled"

		resp.DialogAction.Message = make(map[string]string)
		resp.DialogAction.Message["contentType"] = "PlainText"
		resp.DialogAction.Message["content"] = "Ok, I notified " + *nameFirst + " " + *nameLast + " of your visit. Call me if you need my help again."

		fmt.Println(resp)
		return &resp, nil
	}

	toFind := *nameFirst + " " + *nameLast
	fmt.Println("searching for ", toFind)

	body, err := getCoworkersJson()
	if err != nil {
		fmt.Println("Failed to get data. ", err)
		return nil, err
	}

	coworker := getClosestCoworker(body, toFind)

	if coworker == nil {
		fmt.Println("No match found")
	} else {
		fmt.Println("Match found ", coworker)
	}

	fmt.Println("resp")
	resp := events.LexResponse{}
	if event.SessionAttributes != nil {
		resp.SessionAttributes = event.SessionAttributes
	} else {
		resp.SessionAttributes = make(map[string]string)
	}

	resp.DialogAction.Type = "ConfirmIntent"
	resp.DialogAction.IntentName = event.CurrentIntent.Name
	resp.DialogAction.Slots = event.CurrentIntent.Slots
	resp.DialogAction.Message = make(map[string]string)
	resp.DialogAction.Message["contentType"] = "PlainText"

	if coworker == nil {
		resp.DialogAction.Message["content"] = fmt.Sprintf("Could not find a coworker with that name.")
	} else {
		resp.DialogAction.Message["content"] = fmt.Sprintf("Please, say Yes if you want to inform %v you are here",
			coworker.FullName)

		names := strings.Split(coworker.FullName, " ")

		lastName := strings.Join(names[1:], " ")
		resp.DialogAction.Slots["CoworkerFirstName"] = &names[0]
		resp.DialogAction.Slots["CoworkerLastName"] = &lastName
	}
	return &resp, nil
}

func processCompany(event events.LexEvent) (*events.LexResponse, error) {
	cn, ok := event.CurrentIntent.Slots["AdventCompany"]
	if !ok || event.CurrentIntent.ConfirmationStatus == "Denied" {
		fmt.Println("No company name got!")
		resp := events.LexResponse{}
		resp.DialogAction.IntentName = event.CurrentIntent.Name
		resp.DialogAction.SlotToElicit = "AdventCompany"
		resp.DialogAction.Slots = event.CurrentIntent.Slots
		resp.DialogAction.Type = "ElicitSlot"
		fmt.Println(resp)
		return &resp, nil
	}

	toFind := *cn
	if event.CurrentIntent.ConfirmationStatus == "Confirmed" {
		fmt.Println("Confirmed!")

		resp := events.LexResponse{}
		resp.DialogAction.Type = "Close"
		resp.DialogAction.FulfillmentState = "Fulfilled"

		resp.DialogAction.Message = make(map[string]string)
		resp.DialogAction.Message["contentType"] = "PlainText"
		resp.DialogAction.Message["content"] = "Ok, I notified " + toFind + " of your visit. Call me if you need my help again."

		fmt.Println(resp)
		return &resp, nil
	}

	fmt.Println("Intent: ", event.CurrentIntent)
	fmt.Println("ConfirmationStatus: ", event.CurrentIntent.ConfirmationStatus)
	fmt.Println("Searching for ", toFind)

	body, err := getCoworkersJson()
	if err != nil {
		fmt.Println("Failed to get data. ", err)
		return nil, err
	}

	coworker := getClosestCompany(body, toFind)

	if coworker == nil {
		fmt.Println("No match found")
	} else {
		fmt.Println("Match found ", coworker)
	}

	fmt.Println("resp")
	resp := events.LexResponse{}
	if event.SessionAttributes != nil {
		resp.SessionAttributes = event.SessionAttributes
	} else {
		resp.SessionAttributes = make(map[string]string)
	}
	resp.DialogAction.Type = "ConfirmIntent"
	resp.DialogAction.IntentName = event.CurrentIntent.Name
	resp.DialogAction.Slots = event.CurrentIntent.Slots
	resp.DialogAction.Message = make(map[string]string)
	resp.DialogAction.Message["contentType"] = "PlainText"
	if coworker == nil {
		resp.DialogAction.Message["content"] = fmt.Sprintf("Sorry, I Could not find a company with that name.")
	} else {
		resp.DialogAction.Message["content"] = fmt.Sprintf("Please, say Yes if you want to inform %v you are here",
			coworker.CompanyName)
	}
	return &resp, nil
}

func HandleLambdaEvent(ctx context.Context, event events.LexEvent) (*events.LexResponse, error) {
	fmt.Println(event)

	if event.CurrentIntent.Name == "Meeting" {
		resp, _ := processCoworker(event)
		fmt.Println(resp)
		return resp, nil
	} else if event.CurrentIntent.Name == "Company" {
		resp, _ := processCompany(event)
		fmt.Println(resp)
		return resp, nil
	} else {
		return nil, errors.New("Intent not supported")
	}
}

func main() {
	lambda.Start(HandleLambdaEvent)

	/*
		nameFirst := "liza"
		nameLast := "speaker"
		HandleLambdaEvent(nil, events.LexEvent{
			MessageVersion:    "",
			InvocationSource:  "",
			UserID:            "",
			InputTranscript:   "",
			SessionAttributes: nil,
			RequestAttributes: nil,
			Bot:               nil,
			OutputDialogMode:  "",
			CurrentIntent: &events.LexCurrentIntent{
				Name: "",
				Slots: events.Slots{
					"CoworkerFirstname": &nameFirst,
					"CoworkerLastname":   &nameLast,
				},
				SlotDetails:        nil,
				ConfirmationStatus: "",
			},
			DialogAction: nil,
		})
	*/
}
