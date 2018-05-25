package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Amount struct {
	Currency   string `json:"currency"`
	MinorUnits int    `json:"minorUnits"`
}

type WebHookPayload struct {
	WebhookNotificationUID string    `json:"webhookNotificationUid"`
	CustomerUID            string    `json:"customerUid"`
	WebhookType            string    `json:"webhookType"`
	EventUID               string    `json:"eventUid"`
	TransactionAmount      Amount    `json:"transactionAmount"`
	SourceAmount           Amount    `json:"sourceAmount"`
	Direction              string    `json:"direction"`
	Description            string    `json:"description"`
	MerchantUID            string    `json:"merchantUid"`
	MerchantLocationUID    string    `json:"merchantLocationUid"`
	Status                 string    `json:"status"`
	TransactionMethod      string    `json:"transactionMethod"`
	TransactionTimestamp   time.Time `json:"transactionTimestamp"`
	MerchantPosData        struct {
		PosTimestamp       string `json:"posTimestamp"`
		CardLast4          string `json:"cardLast4"`
		AuthorisationCode  string `json:"authorisationCode"`
		Country            string `json:"country"`
		MerchantIdentifier string `json:"merchantIdentifier"`
	} `json:"merchantPosData"`
}

func handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	wh := new(WebHookPayload)
	err := json.Unmarshal([]byte(request.Body), &wh)
	if err != nil {
		return serverError(err)
	}

	sig := request.Headers["X-Hook-Signature"]
	log.Println("Signature:", sig)
	log.Println("Body:", request.Body)

	log.Println("Customer UID:", wh.CustomerUID)
	log.Println("Method", wh.TransactionMethod)
	log.Println("Amount", wh.TransactionAmount.MinorUnits, wh.TransactionAmount.Currency)
	log.Println("Time", wh.TransactionTimestamp.Format(time.RFC3339Nano))

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "",
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

func clientError(status int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       http.StatusText(status),
	}, nil
}

func main() {
	lambda.Start(handle)
}
