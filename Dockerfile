FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/agent ./cmd/agent
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/dbwriter ./cmd/dbwriter
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/producer ./cmd/producer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/alerter ./cmd/alerter
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/wsgateway ./cmd/wsgateway

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/agent /usr/local/bin/agent
COPY --from=build /out/dbwriter /usr/local/bin/dbwriter
COPY --from=build /out/producer /usr/local/bin/producer
COPY --from=build /out/alerter /usr/local/bin/alerter
COPY --from=build /out/wsgateway /usr/local/bin/wsgateway

EXPOSE 8080
