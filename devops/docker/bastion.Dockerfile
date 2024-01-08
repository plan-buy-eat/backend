
FROM --platform=linux/amd64 ubuntu:latest AS build-release-stage

WORKDIR /

RUN apt update
RUN apt install -y wget
RUN apt install -y iputils-ping
RUN apt install -y dnsutils
RUN apt install -y mc

RUN rm -rf /var/lib/apt/lists/*

RUN wget https://github.com/couchbaselabs/sdk-doctor/releases/download/v1.0.8/sdk-doctor-linux

RUN chmod u+x sdk-doctor-linux
