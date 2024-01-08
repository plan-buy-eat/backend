FROM golang:1.21 AS build-stage

WORKDIR /usr/src/app

RUN go install github.com/go-delve/delve/cmd/dlv@latest
# --mount=type=cache,mode=0755,target=/go/pkg/mod- CGO_ENABLED=0

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY ../../go.mod ../../go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN --mount=type=cache,mode=0755,target=/go/pkg/mod CGO_ENABLED=0 go build -v -o /usr/local/bin/app ./item-service/main.go

## Run the tests in the container
#FROM build-stage AS run-test-stage
#RUN go test -v ./...

FROM alpine:latest AS build-release-stage

WORKDIR /

COPY --from=build-stage /go/bin/dlv /dlv
RUN chmod u+x /dlv
COPY --from=build-stage /usr/local/bin/app /app

EXPOSE 8080 40000

ARG COUCHBASE_CONNECTION_STRING
ARG COUCHBASE_USERNAME
ARG COUCHBASE_PASSWORD
ARG COUCHBASE_BUCKET
ARG SERVICE_NAME
ARG SERVICE_VERSION

ENV COUCHBASE_CONNECTION_STRING $COUCHBASE_CONNECTION_STRING
ENV COUCHBASE_USERNAME $COUCHBASE_USERNAME
ENV COUCHBASE_PASSWORD $COUCHBASE_PASSWORD
ENV SERVICE_NAME $SERVICE_NAME
ENV SERVICE_VERSION $SERVICE_VERSION


#ENTRYPOINT ["/app"]
#CMD ["/bin/sh"]
CMD ./dlv --listen=:40000 --headless=true --api-version=2 --log exec ./app