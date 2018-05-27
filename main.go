package main

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/billglover/starling"
	"golang.org/x/oauth2"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Fetch the Personal Webhook Secret from the Parameter Store
	svc := ssm.New(session.New())
	swh := "starling-webhook-secret"
	spt := "starling-personal-token"
	gUID := "starling-savings-goal"

	dec := true
	paramsIn := ssm.GetParametersInput{
		Names:          []*string{&swh, &spt, &gUID},
		WithDecryption: &dec,
	}

	paramsOut, err := svc.GetParameters(&paramsIn)
	if err != nil {
		log.Println(err)
		return serverError(err)
	}
	params := make(map[string]string, len(paramsOut.Parameters))
	for _, p := range paramsOut.Parameters {
		params[*p.Name] = *p.Value
	}

	secret := params[swh]
	token := params[spt]
	goal := params[gUID]

	// Calculate the request signature
	sha512 := sha512.New()
	sha512.Write([]byte(secret + request.Body))
	recSig := base64.StdEncoding.EncodeToString(sha512.Sum(nil))
	reqSig := request.Headers["X-Hook-Signature"]

	// Reject the request if it doesn't match the signature header
	if reqSig != recSig {
		log.Println("WARN: invalid request signature received")
		return clientError(http.StatusBadRequest)
	}

	wh := new(starling.WebHookPayload)
	err = json.Unmarshal([]byte(request.Body), &wh)
	if err != nil {
		log.Println("ERROR failed to unmarshal body:", err)
		return serverError(err)
	}

	log.Println("Customer UID:", wh.CustomerUID)
	log.Println("Type:", wh.Content.Type)
	log.Println("Amount:", wh.Content.Amount)
	log.Println("Time:", wh.Timestamp.Format(time.RFC3339Nano))

	// Don't round-up anything other than card transactions
	if wh.Content.Type != "TRANSACTION_CARD" {
		log.Println("Ignoring non-card transaction")
		return success()
	}

	// Don't round-up incoming (i.e. positive) amounts
	if wh.Content.Amount >= 0.0 {
		log.Println("Ignoring inbound transaction")
		return success()
	}

	// Round up to the nearest major unit
	minorUnits := int64(wh.Content.Amount - 100)
	ra := roundUp(minorUnits)
	log.Println("Rounding up gives:", ra)

	// Don't try and transfer a zero value to the savings goal
	if ra == 0 {
		return success()
	}

	// Transfer the funds to the savings goal
	log.Println("Transfering to goal:", goal)
	ctx := context.Background()
	sb := newClient(ctx, token)
	amt := starling.Amount{
		MinorUnits: ra,
		Currency:   wh.Content.SourceCurrency,
	}

	txn, resp, err := sb.AddMoney(ctx, goal, amt)
	if err != nil {
		log.Println("ERROR failed to move money to savings goal:", err)
		log.Println("API returned:", resp.Status)
		return serverError(err)
	}

	log.Println("Successful round-up:", txn)
	return success()
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

func main() {
	lambda.Start(handle)
}

func newClient(ctx context.Context, token string) *starling.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	baseURL, _ := url.Parse(starling.ProdURL)
	opts := starling.ClientOptions{BaseURL: baseURL}
	return starling.NewClientWithOptions(tc, opts)
}

func roundUp(txn int64) int64 {
	return (txn/100)*100 - txn + 100
}
