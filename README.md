# courier

`courier` is a universal tool for blue-green and canary deployment of AWS target groups and load balancers.

## Deployment Models

`courier` supports the following deployment models:

- CloudFormation Custom Resource
- AWS CDK Custom Resource
- Terraform provider (currently part of [terraform-provider-eksctl](https://github.com/mumoshu/terraform-provider-eksctl))

### CloudFormation Custom Resource

`courier` can be deployed as an AWS Lambda function that backs CloudFormation custom resources.

Clone this repository and run the below to deploy:


```
$ make deploy
```

which internally runs:

```
cd funcs/resource && \
  sam build --parallel && \
  if [ -e samconfig.toml ]; then sam deploy --no-confirm-changeset; else sam deploy --guided; fi
```

Grab the stack output named `HelloWorldFunction` from the `sam deploy` output.
This is the AWS Lambda function ARN which must be specified for each CloudFormation custom resource within your stack
template.

`courier` currently exposes `CourierManagedALBListenerRule` type.

Let's say you have:

- Two target groups named `blue` and `green` to deploy in a blue-green/canary deployment manner, and
- An AWS Application LoadBalancer and its "Listener" which serves the user-facing traffic,

You can define a `CourierManagedALBListenerRule` resource like: 

```yaml
Resources:
  MyCourierManagedALBListenerRule:
    Type: Custom::CourierManagedALBListenerRule
    Properties:
      # This is the AWS lambda function ARN you've grabbed in the previous step
      ServiceToken: arn:aws:lambda:us-east-2:${AWS_ACCOUNT_ID}:function:sam-app-HelloWorldFunction-TOVCGV92O5PE
      ListenerArn: arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:listener/app/existingvpc2/14f06032761c98f0/e0a5c8bba468222b
      Priority: 11
      Hosts:
        - example.com
      Destinations:
        - TargetGroupArn: arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:targetgroup/blue/d5dc323e7ee6d8b1
          Weight: 0
        - TargetGroupArn: arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:targetgroup/green/138dc51089e322f9
          Weight: 100
      StepWeight: 5
      StepInterval: 5s
```

Now create a CloudFormation stack, and the `courier` Lambda function is invoked by CloudFormation to:

- Create a new ALB listener rule with the priority `11` for the listener `arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:listener/app/existingvpc2/14f06032761c98f0/e0a5c8bba468222b`
- Create a new ["forward" action config](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-listeners.html#rule-action-types) for the target groups

The "forward" action config has two target groups with respective weights. In other words the ALB API receives a configuration like:

```
[
  {
      "Type": "forward",
      "ForwardConfig": {
          "TargetGroups": [
              {
                  "TargetGroupArn": "arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:targetgroup/blue/d5dc323e7ee6d8b1",
                  "Weight": 0
              },
              {
                  "TargetGroupArn": "arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:targetgroup/green/138dc51089e322f9",
                  "Weight": 100
              }
          ]
      }
  }
]
```

Now, trigger a linear deployment by swapping `Weight` for the two target groups:

```yaml
Resources:
  MyCourierManagedALBListenerRule:
    Type: Custom::CourierManagedALBListenerRule
    Properties:
      # This is the AWS lambda function ARN you've grabbed in the previous step
      ServiceToken: arn:aws:lambda:us-east-2:${AWS_ACCOUNT_ID}:function:sam-app-HelloWorldFunction-TOVCGV92O5PE
      ListenerArn: arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:listener/app/existingvpc2/14f06032761c98f0/e0a5c8bba468222b
      Priority: 11
      Hosts:
        - example.com
      Destinations:
        - TargetGroupArn: arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:targetgroup/blue/d5dc323e7ee6d8b1
          Weight: 100
        - TargetGroupArn: arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:targetgroup/green/138dc51089e322f9
          Weight: 0
      StepWeight: 5
      StepInterval: 5s
```

On stack update, `courier` function periodically updates the forward action config for each 5 seconds (StepInterval) in the following sequence:

```
- (start  ) 00:00                   green 100, blue   0
- (step  1) 00:05 (+ step interval) green  95, blue   5 (+ step weight)
- (step  3) 00:10 (+ step interval) green  90, blue  10 (+ step weight)
- ...
- (step 20) 01:40 (+ step interval) green   0, blue 100 (+ step weight)
```

Note that `courier` does nothing on resource and stack deletion as of today.
Please submit a feature request with detailed use-cases if you're interested.

To make it a canary deployment, you need to specify what you need to analyze the canary - metric conditions

`courier` currently supports two types of metric:

- `DatadogMetric`
- `CloudWatchMetric`

For `CloudWatchMetric`, I would recommend getting started by using some kind of error rate.

Let's see how the config look like when we used [`RequestCount` and `HTTPCode_Target_5XX_Count`](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-cloudwatch-metrics.html#load-balancer-metrics-alb):

```
CloudWatchMetric:
  Max: "5.0"
  Interval: "300s"
  Query: |
    [
        {
            "Id": "e1",
            "Expression": "m1 / m2",
            "Label": "ErrorRate"
        },
        {
            "Id": "m1",
            "MetricStat": {
                "Metric": {
                    "Namespace": "AWS/ApplicationELB",
                    "MetricName": "RequestCount",
                    "Dimensions": [
                        {
                            "Name": "TargetGroup",
                            "Value": "targetgroup/green/138dc51089e322f9"
                        },
                        {
                            "Name": "TargetGroup",
                            "Value": "app/${ALB_NAME}/14f06032761c98f0"
                        }
                    ]
                },
                "Period": 300,
                "Stat": "Sum",
                "Unit": "Count"
            },
            "ReturnData": false
        },
        {
            "Id": "m2",
            "MetricStat": {
                "Metric": {
                    "Namespace": "AWS/ApplicationELB",
                    "MetricName": "HTTPCode_Target_5XX_Count",
                    "Dimensions": [
                        {
                            "Name": "TargetGroup",
                            "Value": "targetgroup/green/138dc51089e322f9"
                        },
                        {
                            "Name": "TargetGroup",
                            "Value": "app/${ALB_NAME}/14f06032761c98f0"
                        }
                    ]
                },
                "Period": 300,
                "Stat": "Sum",
                "Unit": "Count"
            },
            "ReturnData": false
        }
    ]
```

The whole custom resource definition would now look like:

```yaml
Resources:
  MyCourierManagedALBListenerRule:
    Type: Custom::CourierManagedALBListenerRule
    Properties:
      # This is the AWS lambda function ARN you've grabbed in the previous step
      ServiceToken: arn:aws:lambda:us-east-2:${AWS_ACCOUNT_ID}:function:sam-app-HelloWorldFunction-TOVCGV92O5PE
      ListenerArn: arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:listener/app/existingvpc2/14f06032761c98f0/e0a5c8bba468222b
      Priority: 11
      Hosts:
        - example.com
      Destinations:
        - TargetGroupArn: arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:targetgroup/blue/d5dc323e7ee6d8b1
          Weight: 100
        - TargetGroupArn: arn:aws:elasticloadbalancing:us-east-2:${AWS_ACCOUNT_ID}:targetgroup/green/138dc51089e322f9
          Weight: 0
      StepWeight: 5
      StepInterval: 5s
      CloudWatchMetric:
        Query: |
          ...
```

### AWS CDK Custom Resource

Given that you've already made `courier` custom resource types to CloudFormation following steps in [CloudFormation Custom Resource](#cloudFormation-custom-resource),
you can naturally use it from your AWS CDK project.

In nutshell, you would use `cdk.CfnResource` constructor to replicate the custom resource definition.

```typescript
const funcARN = 'arn:aws:lambda:${awsRegion}:${awsAccount}:function:${funcName}';

const courierManagedListenerRule = new cdk.CfnResource(this, 'MyCourierManagedALBListenerRule', {
    type: "Custom::CourierManagedALBListenerRule",
    properties: {
        // See https://github.com/aws/aws-cdk/issues/4810#issuecomment-581490027
        // ServiceToken: cdk.Fn.importValue('FuncArnExport')),
        ServiceToken: funcARN,
        ListenerArn: "arn:aws:elasticloadbalancing:${awsRegion}:${awsAccount}:listener/app/${loadBalancerName}/${listenerID}",
        //
        Priority: 11,
        Hosts: [
            "example.com"
        ],
        Destinations: [
            {
                TargetGroupArn: "arn:aws:elasticloadbalancing:${awsRegion}:${awsAccount}:targetgroup/blue/d5dc323e7ee6d8b1",
                Weight: 0,
            },
            {
                TargetGroupArn: "arn:aws:elasticloadbalancing:${awsRegion}:${awsAccount}:targetgroup/green/138dc51089e322f9",
                Weight: 100,
            }
        ],
        StepWeight: 5,
        StepInterval: "5s",
    }
});
```

Now deploy your CDK application as usual with e.g.:

```
npm run build && cdk deploy -f
```

## Other notes

This is a sample template for courier - Below is a brief explanation of what we have generated for you:

```bash
.
├── Makefile                    <-- Make to automate build
├── README.md                   <-- This instructions file
├── hello-world                 <-- Source code for a lambda function
│   ├── main.go                 <-- Lambda function code
│   └── main_test.go            <-- Unit tests
└── template.yaml
```

## Requirements

* AWS CLI already configured with Administrator permission
* [Docker installed](https://www.docker.com/community-edition)
* [Golang](https://golang.org)
* SAM CLI - [Install the SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)

## Setup process

### Installing dependencies & building the target 

In this example we use the built-in `sam build` to automatically download all the dependencies and package our build target.   
Read more about [SAM Build here](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/sam-cli-command-reference-sam-build.html) 

The `sam build` command is wrapped inside of the `Makefile`. To execute this simply run
 
```shell
make
```

### Local development

**Invoking function locally through local API Gateway**

```bash
sam local start-api
```

If the previous command ran successfully you should now be able to hit the following local endpoint to invoke your function `http://localhost:3000/hello`

**SAM CLI** is used to emulate both Lambda and API Gateway locally and uses our `template.yaml` to understand how to bootstrap this environment (runtime, where the source code is, etc.) - The following excerpt is what the CLI will read in order to initialize an API and its routes:

```yaml
...
Events:
    HelloWorld:
        Type: Api # More info about API Event Source: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#api
        Properties:
            Path: /hello
            Method: get
```

## Packaging and deployment

AWS Lambda Golang runtime requires a flat folder with the executable generated on build step. SAM will use `CodeUri` property to know where to look up for the application:

```yaml
...
    FirstFunction:
        Type: AWS::Serverless::Function
        Properties:
            CodeUri: hello_world/
            ...
```

To deploy your application for the first time, run the following in your shell:

```bash
sam deploy --guided
```

The command will package and deploy your application to AWS, with a series of prompts:

* **Stack Name**: The name of the stack to deploy to CloudFormation. This should be unique to your account and region, and a good starting point would be something matching your project name.
* **AWS Region**: The AWS region you want to deploy your app to.
* **Confirm changes before deploy**: If set to yes, any change sets will be shown to you before execution for manual review. If set to no, the AWS SAM CLI will automatically deploy application changes.
* **Allow SAM CLI IAM role creation**: Many AWS SAM templates, including this example, create AWS IAM roles required for the AWS Lambda function(s) included to access AWS services. By default, these are scoped down to minimum required permissions. To deploy an AWS CloudFormation stack which creates or modified IAM roles, the `CAPABILITY_IAM` value for `capabilities` must be provided. If permission isn't provided through this prompt, to deploy this example you must explicitly pass `--capabilities CAPABILITY_IAM` to the `sam deploy` command.
* **Save arguments to samconfig.toml**: If set to yes, your choices will be saved to a configuration file inside the project, so that in the future you can just re-run `sam deploy` without parameters to deploy changes to your application.

You can find your API Gateway Endpoint URL in the output values displayed after deployment.

### Testing

We use `testing` package that is built-in in Golang and you can simply run the following command to run our tests:

```shell
go test -v ./hello-world/
```
# Appendix

### Golang installation

Please ensure Go 1.x (where 'x' is the latest version) is installed as per the instructions on the official golang website: https://golang.org/doc/install

A quickstart way would be to use Homebrew, chocolatey or your linux package manager.

#### Homebrew (Mac)

Issue the following command from the terminal:

```shell
brew install golang
```

If it's already installed, run the following command to ensure it's the latest version:

```shell
brew update
brew upgrade golang
```

#### Chocolatey (Windows)

Issue the following command from the powershell:

```shell
choco install golang
```

If it's already installed, run the following command to ensure it's the latest version:

```shell
choco upgrade golang
```

## Bringing to the next level

Here are a few ideas that you can use to get more acquainted as to how this overall process works:

* Create an additional API resource (e.g. /hello/{proxy+}) and return the name requested through this new path
* Update unit test to capture that
* Package & Deploy

Next, you can use the following resources to know more about beyond hello world samples and how others structure their Serverless applications:

* [AWS Serverless Application Repository](https://aws.amazon.com/serverless/serverlessrepo/)
