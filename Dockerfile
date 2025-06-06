FROM golang:1.24.3

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /go-kanban

COPY .env* ./

EXPOSE 8080

CMD ["/go-kanban"]