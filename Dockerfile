FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o fritzbox-cloudflare-dnsupdater .

FROM scratch
WORKDIR /
COPY --from=builder /app/fritzbox-cloudflare-dnsupdater /
EXPOSE 80
CMD ["/fritzbox-cloudflare-dnsupdater"]
