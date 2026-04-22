FROM golang:1.25-bookworm

WORKDIR /workspace/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend ./
RUN go build -o /usr/local/bin/snowpanel-backend ./cmd/server

EXPOSE 8080

CMD ["/usr/local/bin/snowpanel-backend"]
