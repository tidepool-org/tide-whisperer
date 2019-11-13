# Development
FROM golang:1.11.4-alpine AS development
WORKDIR /go/src/github.com/tidepool-org/tide-whisperer
RUN adduser -D tidepool && \
    chown -R tidepool /go/src/github.com/tidepool-org/tide-whisperer
USER tidepool
COPY --chown=tidepool . .
RUN ./build.sh
CMD ["./dist/tide-whisperer"]

# Production
FROM alpine:latest AS production
WORKDIR /home/tidepool
RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add --no-cache ca-certificates && \
    adduser -D tidepool
USER tidepool
COPY --from=development --chown=tidepool /go/src/github.com/tidepool-org/tide-whisperer/dist/tide-whisperer .
CMD ["./tide-whisperer"]
