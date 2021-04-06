FROM golang:1.16-alpine AS build
RUN apk add --update --no-cache git
WORKDIR /src
COPY ./go.* ./
RUN go mod download
COPY . .

ENV CGO_ENABLED 0
RUN go build -o /dtn -ldflags "-s -w"

FROM alpine
COPY --from=build /dtn /usr/local/bin/
ENTRYPOINT ["dtn"]
