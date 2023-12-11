FROM golang:1.21 AS build-stage

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN --mount=type=cache,mode=0755,target=/go/pkg/mod GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -v -o /usr/local/bin/app ./...

## Run the tests in the container
#FROM build-stage AS run-test-stage
#RUN go test -v ./...

# Deploy the application binary into a lean image
#FROM gcr.io/distroless/base-debian11 AS build-release-stage
FROM --platform=linux/amd64 alpine:latest AS build-release-stage
#FROM --platform=linux/amd64 ubuntu:latest AS build-release-stage

RUN addgroup --system nonroot
RUN adduser --system nonroot --ingroup nonroot

WORKDIR /

COPY --from=build-stage /usr/local/bin/app /app
# TODO: remove
COPY --from=build-stage /usr/src/app/.env.docker /.env

RUN apk add libcap && setcap 'cap_net_bind_service=+ep' /app

EXPOSE 8080

USER nonroot:nonroot

ARG COUCHBASE_CONNECTION_STRING
ARG COUCHBASE_USERNAME
ARG COUCHBASE_PASSWORD
ARG COUCHBASE_BUCKET

ENV COUCHBASE_CONNECTION_STRING couchbase://couchbase-0000
ENV COUCHBASE_USERNAME Administrator
ENV COUCHBASE_PASSWORD $COUCHBASE_PASSWORD
ENV COUCHBASE_BUCKET default

ENTRYPOINT ["/app"]
#CMD ["/bin/sh"]