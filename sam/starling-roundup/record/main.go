package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/billglover/starling"
	"golang.org/x/oauth2"
)

var token string
var goal string
var table string
var region string
var db *dynamodb.DynamoDB

func handler(request events.DynamoDBEvent) {
	for _, r := range request.Records {
		wh := new(starling.WebHookPayload)
		UnmarshalStreamImage(r.Change.NewImage, &wh)

		fmt.Println("INFO: UID:", wh.UID)
		fmt.Println("INFO: Amount:", wh.Content.Amount)
		fmt.Println("INFO: Description:", wh.Content.ForCustomer)
		fmt.Println("INFO: Change:", r.EventName)

		// Don't round-up repeat transactions
		if r.EventName != "INSERT" {
			fmt.Println("INFO: updated transaction, skipping round-up")
			return
		}

		// Don't round-up anything other than card transactions
		if wh.Content.Type != "TRANSACTION_CARD" && wh.Content.Type != "TRANSACTION_MOBILE_WALLET" {
			fmt.Println("INFO: ignoring non-card transaction")
			return
		}

		// Don't round-up incoming (i.e. positive) amounts
		if wh.Content.Amount >= 0.0 {
			fmt.Println("INFO: ignoring inbound transaction")
			return
		}

		// Round up to the nearest major unit
		amtMinor := math.Round(wh.Content.Amount * -100)
		ra := roundUp(int64(amtMinor))
		fmt.Println("INFO: round-up yields:", ra)

		// Don't try and transfer a zero value to the savings goal
		if ra == 0 {
			fmt.Println("INFO: nothing to round-up")
			return
		}

		// Transfer the funds to the savings goal
		ctx := context.Background()
		sb := newClient(ctx, token)
		amt := starling.Amount{
			MinorUnits: ra,
			Currency:   wh.Content.SourceCurrency,
		}

		txn, resp, err := sb.AddMoney(ctx, goal, amt)
		if err != nil {
			fmt.Println("ERROR: failed to move money to savings goal:", err)
			fmt.Println("ERROR: Starling Bank API returned:", resp.Status)
			return
		}

		fmt.Println("INFO: round-up successful:", txn)
	}
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

// UnmarshalStreamImage takes a StreamImage and unmarshalls it into a Go struct. Source code taken from
// this Stack Overflow answer: https://stackoverflow.com/a/50017398
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

func requestParameters() error {
	svc := ssm.New(session.New())
	spt := "starling-personal-token"
	gUID := "starling-savings-goal"

	decrypt := true
	paramsIn := ssm.GetParametersInput{
		Names:          []*string{&spt, &gUID},
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

	token = params[spt]
	goal = params[gUID]

	return nil
}

func newClient(ctx context.Context, token string) *starling.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	baseURL, _ := url.Parse(starling.ProdURL)
	opts := starling.ClientOptions{BaseURL: baseURL}
	return starling.NewClientWithOptions(tc, opts)
}

func roundUp(txn int64) int64 {
	// By using 99 we ensure that a 0 value rounds is not rounded up
	// to the next 100.
	amtRound := (txn + 99) / 100 * 100
	return amtRound - txn

}
