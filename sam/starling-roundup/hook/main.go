package main

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/billglover/starling"
)

var secret string
var table string
var region string
var db *dynamodb.DynamoDB

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Calculate the request signature and reject the request if it doesn't match the signature header
	sha512 := sha512.New()
	sha512.Write([]byte(secret + request.Body))
	recSig := base64.StdEncoding.EncodeToString(sha512.Sum(nil))
	reqSig := request.Headers["X-Hook-Signature"]
	if reqSig != recSig {
		fmt.Println("WARN: invalid request signature received")
		return clientError(http.StatusBadRequest)
	}

	// parse the Starling Bank web-hook payload
	wh := new(starling.WebHookPayload)
	err := json.Unmarshal([]byte(request.Body), &wh)
	if err != nil {
		fmt.Println("ERROR: failed to unmarshal web hook payload:", err)
		return serverError(err)
	}
	fmt.Println("INFO: type:", wh.Content.Type)
	fmt.Println("INFO: amount:", wh.Content.Amount)

	// store web-hook payload in DynamoDB
	payload, err := dynamodbattribute.MarshalMap(wh)
	if err != nil {
		fmt.Println("ERROR: Got error marshalling map:", err)
		return serverError(err)
	}

	input := &dynamodb.PutItemInput{
		Item:      payload,
		TableName: aws.String(table),
	}

	_, err = db.PutItem(input)
	if err != nil {
		fmt.Println("ERROR: got error calling PutItem:", err)
		return serverError(err)
	}

	fmt.Println("INFO: successfully submitted record:")
	return success()
}

func main() {
	err := requestParameters()
	if err != nil {
		fmt.Println("ERROR: unable to retrieve parameters:", err)
		return
	}

	table = os.Getenv("STARLING_TABLE")
	region = os.Getenv("STARLING_REGION")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

	if err != nil {
		fmt.Println("ERROR: unable to create session:", err)
		os.Exit(1)
	}

	db = dynamodb.New(sess)
	lambda.Start(handler)
}

func requestParameters() error {
	svc := ssm.New(session.New())
	swh := "starling-webhook-secret"

	decrypt := true
	paramsIn := ssm.GetParametersInput{
		Names:          []*string{&swh},
		WithDecryption: &decrypt,
	}

	paramsOut, err := svc.GetParameters(&paramsIn)
	if err != nil {
		return err
	}
	params := make(map[string]string, len(paramsOut.Parameters))
	for _, p := range paramsOut.Parameters {
		params[*p.Name] = *p.Value
	}

	secret = params[swh]

	return nil
}

func success() (events.APIGatewayProxyResponse, error) {
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
