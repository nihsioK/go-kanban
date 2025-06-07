FROM golang:1.24.3 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o go-kanban ./cmd/server

CMD ["./go-kanban"]
