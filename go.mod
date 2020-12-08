require (
	github.com/aws/aws-lambda-go v1.13.3
	github.com/google/go-cmp v0.5.2
	github.com/google/uuid v1.1.1
	github.com/jarcoal/httpmock v1.0.6
	github.com/mumoshu/terraform-provider-eksctl v0.0.0-00010101000000-000000000000
)

module hello-world

go 1.13

replace github.com/mumoshu/terraform-provider-eksctl => ../../terraform-provider-eksctl
