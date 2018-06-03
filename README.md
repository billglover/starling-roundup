# starling-roundup

This serverless application allows you to round-up your Starling bank transactions to the nearest £1 and transfer the delta to a savings goal. 

This implementation is targetted at Amazon's AWS Serverless Application Model (SAM), but the principle could easily be deployed elsewhere.

## How it works

1. Starling Bank triggers a web-hook on each transaction.
2. This web-hook is configured to call an API deployed on API Gateway.
3. The API call is handled by a Lambda function which checks the signature of the request and submits it to a DynamoDB table for processing.
4. A second Lambda function looks for new entries to the table and performs the round-up before sending a request back to Starling Bank to move the delta to a savings goal.

![alt text][arch]

[arch]: img/StarlingRoundUp.png "AWS Architecture Diagram"

## Questions

**How do you store secure parameters?** This application retrieves all parameters from the System Manager Parameter Store. This allows the parameters to be stored securely and only accessed by the Lambda funtion.

 - `starling-webhook-secret` - used to validate inbound requests
 - `starling-personal-token` - used to request transfers to savings goal
 - `starling-savings-goal` - the identifier of the target savings goal

**Why do you need the DynamoDB?** An early version of the solution didn't include a DynamoDB table. The first Lamda function was responsible for rounding-up transaction values and requesting the transfer to a savings goal. The DynamoDB was introduced because occasionally a web-hook would fire twice for the same transaction and the intermediary data store allows us to detect duplicate transactions and only perform the round-up once.

## Installation

### Pre-Requisites

* A [Starling Bank](https://starlingbank.com) account
* A [Starling Bank Developer](https://developer.starlingbank.com) account
* An [AWS](https://aws.amazon.com) account

### Configureing Your App

* Deploy the Serverless Application, making a note of the 'WebHook' URL that is returned as part of the output. You'll need this to configure your Starling application.

```
aws cloudformation deploy --region eu-west-2 --capabilities CAPABILITY_IAM --template-file /Users/bill/go/src/github.com/billglover/starling-roundup/aws/sam/starling-roundup/template.yaml --stack-name starling-roundup
```

*Note:* this assumes you are using the code packages hosted in my s3 bucket. The bucket name and path to the code packages are exposed as parameters should you wish to override them.

* Register an application with your Starling developer account.
* Create a personal web-hook using the URL returned above.
* Make a note of the web-hook secret and the personal access token.
* Create the three secure parameters, named as above, in the Parameter Store.

### Contributing

Issues and pull requests are both welcome. I'd be particularly interested in help around packaging this up to simplify the deployment process.
