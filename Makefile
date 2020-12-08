.PHONY: build

build:
	sam build

acceptance/tests:
	cd funcs/resource && \
	  sam build --cached --parallel &&\
	  cd .aws-sam/build && \
	  sam local invoke HelloWorldFunction --template-file ./template.yaml

	cd funcs/handler && \
	  sam build --cached --parallel &&\
	  cd .aws-sam/build && \
	  sam local invoke HelloWorldFunction --template-file ./template.yaml

TAG ?= dev
REPO ?= public.ecr.aws/$(ALIAS)
IMAGE ?= ${REPO}/courier:$(TAG)

image/run:
	rm -rf build/dist/courier
	GOOS=linux GOARCH=amd64 go build -o build/dist/courier ./cmd/resource
	docker build --build-arg APP_BIN_PATH=dist/courier -t $(IMAGE) build
	docker run -p 9000:8080 $(IMAGE)

image/echo:
	docker run -p 9001:80 -p 9002:443 --rm -t mendhak/http-https-echo

image/test:
	RESPONSE_URL=http://$(shell ipconfig getifaddr en0):9001 ./test.sh

deploy:
	cd funcs/resource && \
	  sam build --parallel &&\
	  if [ -e samconfig.toml ]; then sam deploy --no-confirm-changeset; else sam deploy --guided; fi

deploy/cdk:
	cd examples/cdk && \
	npm run build && cdk deploy

cdk/destroy:
	cd examples/cdk && \
	npm run build && cdk destroy -f

logs:
	cd funcs/resource && \
	  sam logs --name sam-app-HelloWorldFunction-TOVCGV92O5PE --region us-east-2 --start-time "60mins ago"

deps:
	npm uninstall -g aws-cdk
	npm install -g aws-cdk
	npm i -g typescript
	# npx cdk --version
	# and examples/cdk is initialized with:
	# npx cdk init app --language typescript
