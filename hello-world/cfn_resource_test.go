package helloworld

import (
	"context"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/google/go-cmp/cmp"
	"github.com/jarcoal/httpmock"
	"net/http"
	"testing"
)

func TestResource(t *testing.T) {

	// See `sam local generate-event cloudformation create-request`
	event := cfn.Event{
		RequestType:  cfn.RequestCreate,
		RequestID:    "unique id for this create request",
		ResponseURL:  "http://pre-signed-S3-url-for-response",
		ResourceType: "Custom::TestResource",
		//PhysicalResourceID:    "",
		LogicalResourceID: "MyTestResource",
		StackID:           "arn:aws:cloudformation:us-east-1:123456789012:stack/MyStack/guid",
		ResourceProperties: map[string]interface{}{
			"StackName": "MyStack",
			"List": []string{
				"1", "2", "3",
			},
			"Echo": "foobar",
		},
	}

	_, data, err := CourierALBResource(context.TODO(), event)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]interface{}{
		"Echo": "foobar",
	}

	if d := cmp.Diff(expected, data); d != "" {
		t.Errorf("unexpected diff: want(-), got (+)\n%s", d)
	}

	// LambdaWrap uses http.DefaultClient, that defaults to http.DefaultTransport

	handler := cfn.LambdaWrap(CourierALBResource)

	// We use httpmock to alter DefaulTransport

	httpmock.Activate()

	// We don't need to DefaultTransport.Reset() as DeactivateAndReset does that
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(http.MethodPut, "http://pre-signed-S3-url-for-response",
		httpmock.NewJsonResponderOrPanic(200, `[{"id": 1, "name": "My Great Article"}]`))

	reason, err := handler(context.TODO(), event)
	if err != nil {
		t.Errorf("reason: %s", reason)
		t.Fatalf("unexpected error: %v", err)
	}
}
