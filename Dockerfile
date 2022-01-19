# Development
FROM golang:1.17-alpine AS development
ARG GOPRIVATE
ARG GITHUB_TOKEN
ENV GO111MODULE=on
WORKDIR /go/src/github.com/tidepool-org/tide-whisperer
RUN adduser -D tidepool && \
    apk add --no-cache gcc musl-dev git && \
    chown -R tidepool /go/src/github.com/tidepool-org/tide-whisperer
USER tidepool
COPY --chown=tidepool . .
RUN git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/" && \ 
    ./build.sh && \
    git config --global --unset url."https://${GITHUB_TOKEN}@github.com/".insteadOf
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
