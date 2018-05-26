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

type WebHookPayload struct {
	WebhookNotificationUID string    `json:"webhookNotificationUid"`
	Timestamp              time.Time `json:"timestamp"`
	Content                struct {
		Class          string  `json:"class"`
		TransactionUID string  `json:"transactionUid"`
		Amount         float64 `json:"amount"`
		SourceCurrency string  `json:"sourceCurrency"`
		SourceAmount   float64 `json:"sourceAmount"`
		CounterParty   string  `json:"counterParty"`
		Reference      string  `json:"reference"`
		Type           string  `json:"type"`
		ForCustomer    string  `json:"forCustomer"`
	} `json:"content"`
	AccountHolderUID string `json:"accountHolderUid"`
	WebhookType      string `json:"webhookType"`
	CustomerUID      string `json:"customerUid"`
	UID              string `json:"uid"`
}

func handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Fetch the Personal Webhook Secret from the Parameter Store
	svc := ssm.New(session.New())
	swh := "starling-webhook-secret"
	spt := "starling-personal-token"

	dec := true
	paramsIn := ssm.GetParametersInput{
		Names:          []*string{&swh, &spt},
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

	wh := new(WebHookPayload)
	err = json.Unmarshal([]byte(request.Body), &wh)
	if err != nil {
		log.Println("ERROR failed to unmarshal body:", err)
		return serverError(err)
	}

	log.Println("Customer UID:", wh.CustomerUID)
	log.Println("Type:", wh.Content.Type)
	log.Println("Amount:", wh.Content.Amount)
	log.Println("Time:", wh.Timestamp.Format(time.RFC3339Nano))

	ctx := context.Background()
	sb := newClient(ctx, token)
	txn, _, err := sb.Transaction(ctx, wh.Content.TransactionUID)
	if err != nil {
		log.Println("ERROR failed to request transaction detail:", err)
		return serverError(err)
	}

	log.Println("Narrative:", txn.Narrative)

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
