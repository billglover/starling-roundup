.PHONY: clean build deploy

clean: 
	go clean ./hook
	go clean ./record
	rm -rf ./bin/*
	
build:
	GOOS=linux GOARCH=amd64 go build -o bin/main ./hook
	zip -j bin/hook.zip bin/main
	rm -f bin/main

	GOOS=linux GOARCH=amd64 go build -o bin/main ./record
	zip -j bin/record.zip bin/main
	rm -f bin/main

deploy:
	aws s3 cp bin/hook.zip s3://me.billglover.starling/hook.zip
	aws s3 cp bin/record.zip s3://me.billglover.starling/record.zip
	aws lambda update-function-code --function-name sam-demo-WebHookHandler-ABGU7GL875BS --s3-bucket me.billglover.starling --s3-key hook.zip --region eu-west-2
	aws lambda update-function-code --function-name sam-demo-RecordHandler-1WDM9L8P7FSOX --s3-bucket me.billglover.starling --s3-key record.zip --region eu-west-2 

sam:
	aws cloudformation deploy --region eu-west-2 --capabilities CAPABILITY_IAM --template-file /Users/bill/go/src/github.com/billglover/starling-roundup/aws/sam/starling-roundup/template.yaml --stack-name sam-demo
	aws cloudformation list-exports --region eu-west-2 --query "Exports[].Value" --no-paginate
