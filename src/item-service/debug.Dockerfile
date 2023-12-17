FROM golang:1.21 AS build-stage

WORKDIR /usr/src/app

RUN --mount=type=cache,mode=0755,target=/go/pkg/mod- CGO_ENABLED=0 go install github.com/go-delve/delve/cmd/dlv@latest

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN --mount=type=cache,mode=0755,target=/go/pkg/mod CGO_ENABLED=0 go build -v -o /usr/local/bin/app ./...

## Run the tests in the container
#FROM build-stage AS run-test-stage
#RUN go test -v ./...

FROM alpine:latest AS build-release-stage
FROM ubuntu:latest AS build-release-stage

WORKDIR /

COPY --from=build-stage /go/bin/dlv /dlv
RUN chmod u+x /dlv
COPY --from=build-stage /usr/local/bin/app /app

EXPOSE 8080 40000

#ENTRYPOINT ["/app"]
#CMD ["/bin/sh"]
CMD ./dlv --listen=:40000 --headless=true --api-version=2 --log exec ./app -- --config /.env