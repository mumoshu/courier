package main

import (
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	helloworld "hello-world/hello-world"
)

func main() {
	lambda.Start(cfn.LambdaWrap(helloworld.CourierALBResource))
}
