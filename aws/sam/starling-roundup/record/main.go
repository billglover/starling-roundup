package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/billglover/starling"
)

var table string
var region string
var db *dynamodb.DynamoDB

func handler(request events.DynamoDBEvent) {
	for _, r := range request.Records {
		wh := new(starling.WebHookPayload)
		UnmarshalStreamImage(r.Change.NewImage, &wh)

		fmt.Println("UID:", wh.UID)
		fmt.Println("Amount:", wh.Content.Amount)
		fmt.Println("Description:", wh.Content.ForCustomer)
		fmt.Println("Change:", r.EventName)
	}
}

func main() {
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

// https://stackoverflow.com/a/50017398
func UnmarshalStreamImage(attribute map[string]events.DynamoDBAttributeValue, out interface{}) error {

	dbAttrMap := make(map[string]*dynamodb.AttributeValue)

	for k, v := range attribute {

		var dbAttr dynamodb.AttributeValue

		bytes, marshalErr := v.MarshalJSON()
		if marshalErr != nil {
			return marshalErr
		}

		json.Unmarshal(bytes, &dbAttr)
		dbAttrMap[k] = &dbAttr
	}

	return dynamodbattribute.UnmarshalMap(dbAttrMap, out)

}
