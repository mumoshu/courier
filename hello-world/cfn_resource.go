package helloworld

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/google/uuid"
	"github.com/mumoshu/terraform-provider-eksctl/pkg/courier"
	"github.com/mumoshu/terraform-provider-eksctl/pkg/sdk/gensdk"
	"log"
	"runtime"
)

const (
	ResourceTypeListenerRule = "CourierManagedALBListenerRule"
)

var albSchema = &courier.ALBSchema{
	Address:                   "Address",
	ListenerARN:               "ListenerArn",
	Priority:                  "Priority",
	Destination:               "Destinations",
	DestinationTargetGroupARN: "TargetGroupArn",
	DestinationWeight:         "Weight",
	StepWeight:                "StepWeight",
	StepInterval:              "StepInterval",
	Hosts:                     "Hosts",
	PathPatterns:              "PathPatterns",
	Methods:                   "Methods",
	SourceIPs:                 "SourceIPs",
	Headers:                   "Headers",
	QueryStrings:              "QueryStrings",
}

var metricSchema = &courier.MetricSchema{
	DatadogMetric:    "DatadogMetric",
	CloudWatchMetric: "CloudWatchMetric",
	Min:              "Min",
	Max:              "Max",
	Interval:         "Interval",
	Address:          "Address",
	Query:            "Query",
	AWSProfile:       "AwsProfile",
	AWSRegion:        "AwsRegion",
}

func CourierALBResource(ctx context.Context, event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			var buf bytes.Buffer

			fmt.Fprintf(&buf, "%+v\n", e)
			for depth := 0; ; depth++ {
				_, file, line, ok := runtime.Caller(depth)
				if !ok {
					break
				}
				fmt.Fprintf(&buf, "%2d] %v:%d\n", depth, file, line)
			}

			err = fmt.Errorf("recovered panic: %s", buf.String())
		}
	}()

	log.Printf("Event: %v", event)

	if event.ResourceType != ResourceTypeListenerRule {
		err = fmt.Errorf("unsupported resource type %q: the only supported type is %q", event.ResourceType, ResourceTypeListenerRule)
		return
	}

	data = map[string]interface{}{
		"Version": "v0.0.0",
	}

	physicalResourceID = event.PhysicalResourceID

	if physicalResourceID == "" {
		physicalResourceID = uuid.New().String()
	}

	switch event.RequestType {
	case cfn.RequestCreate, cfn.RequestUpdate:
		d := &gensdk.MapReader{M: event.ResourceProperties}

		if err = courier.CreateOrUpdateCourierALB(d, albSchema, metricSchema); err != nil {
			return
		}
	case cfn.RequestDelete:
		// Otherwise you get "Invalid PhysicalResourceId" no rolling-back failed creation
		// See https://github.com/aws/aws-cdk/issues/5796
		physicalResourceID = ""
	default:
		return "", nil, fmt.Errorf("unsupported request type: %s", event.RequestType)
	}

	return
}
