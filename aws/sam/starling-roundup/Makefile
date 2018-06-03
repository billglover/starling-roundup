.PHONY: clean build deploy

clean: 
	go clean ./hook
	rm -rf ./bin/*
	
build:
	GOOS=linux GOARCH=amd64 go build -o bin/main ./hook
	zip -j bin/hook.zip bin/main
	rm -f bin/main

deploy:
	aws s3 cp bin/hook.zip s3://me.billglover.starling/hook.zip
	aws lambda update-function-code --function-name sam-demo-WebHookFunction-1AWHY8VAOS13Y --s3-bucket me.billglover.starling --s3-key main.zip --region eu-west-2

sam:
	aws cloudformation deploy --region eu-west-2 --capabilities CAPABILITY_IAM --template-file /Users/bill/go/src/github.com/billglover/starling-roundup/aws/sam/starling-roundup/template.yaml --stack-name sam-demo
	 aws cloudformation list-exports --region eu-west-2 --query "Exports[].Value" --no-paginate
