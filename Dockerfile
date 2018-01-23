# Development
FROM golang:1.9.2-alpine AS development

WORKDIR /go/src/github.com/tidepool-org/tide-whisperer

COPY . .

RUN  ./build.sh

CMD ["./dist/tide-whisperer"]

# Release
FROM alpine:latest AS release

RUN ["apk", "add", "--no-cache", "ca-certificates"]

RUN ["adduser", "-D", "tide-whisperer"]

WORKDIR /home/tide-whisperer

USER tide-whisperer

COPY --from=development --chown=tide-whisperer /go/src/github.com/tidepool-org/tide-whisperer/dist/tide-whisperer .

CMD ["./tide-whisperer"]
