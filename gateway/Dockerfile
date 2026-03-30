FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gateway ./cmd/gateway

FROM alpine:3.21
RUN apk add --no-cache docker-cli
COPY --from=build /gateway /usr/local/bin/gateway
ENTRYPOINT ["gateway"]
