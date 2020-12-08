# See ECR Gallery and AWS docs for more information
# https://gallery.ecr.aws/lambda/go
# https://docs.aws.amazon.com/lambda/latest/dg/go-image.html
FROM public.ecr.aws/lambda/go:latest

ARG APP_BIN_PATH

COPY ${APP_BIN_PATH} ${LAMBDA_TASK_ROOT}/courier

CMD [ "courier" ]
