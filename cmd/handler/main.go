package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	helloworld "hello-world/hello-world"
)

func main() {
	lambda.Start(helloworld.Handler)
}
