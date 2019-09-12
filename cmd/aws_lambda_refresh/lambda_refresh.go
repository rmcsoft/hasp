package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func updateCoworkersJson(value []byte) error {
	bucketName := "rmc-haspbot"
	keyName := "CoworkersCache"

	// Upload input parameters
	upParams := &s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &keyName,
		Body:   bytes.NewReader(value),
	}

	uploader := s3manager.NewUploader(session.New())

	// Perform an upload.
	_, err := uploader.Upload(upParams)
	if err != nil {
		fmt.Println("Error uploading: ", err)
		return err
	}

	return nil
}

func HandleCloudWatchEvent(ctx context.Context, event events.CloudWatchEvent) error {
	//fmt.Println(event)

	httpClient := http.Client{
		Timeout: time.Second * 25, // Maximum of 25 secs
	}

	baseUrl := "https://spaces.nexudus.com/api/spaces/coworkers?size=1000"
	req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Setting headers")
	data := make(map[string]string)
	err = json.Unmarshal(event.Detail, &data)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", "Basic "+data["secret"])
	req.Header.Set("Content", "application/json")

	fmt.Println("Do")
	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	fmt.Println("Read")
	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	fmt.Println("Write")
	err = updateCoworkersJson(filterCoworkers(body))
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func filterCoworkers(body []byte) []byte {
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	records := result["Records"].([]interface{})
	filteredRecords := make([]map[string]interface{}, 1000)
	items := 0
	for _, ivalue := range records {
		value := ivalue.(map[string]interface{})
		if value["TariffId"].(float64) > 0 {
			filteredRecords[items] = make(map[string]interface{})
			filteredRecords[items]["Email"] = strings.ToLower(value["Email"].(string))
			filteredRecords[items]["FullName"] = strings.ToLower(value["FullName"].(string))
			filteredRecords[items]["MobilePhone"] = value["MobilePhone"]
			if value["CompanyName"] != nil {
				filteredRecords[items]["CompanyName"] = strings.TrimLeft(strings.Trim(value["CompanyName"].(string), " -"), ".")
			}
			items++
		}
	}
	data, _ := json.Marshal(filteredRecords[:items])
	return data
}

func main() {
	lambda.Start(HandleCloudWatchEvent)
}
