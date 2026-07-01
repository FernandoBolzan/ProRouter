FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY gateway-go/ .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /prorouter ./cmd/prorouter

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /prorouter /usr/local/bin/prorouter
EXPOSE 8080
ENTRYPOINT ["prorouter"]
CMD ["serve"]
