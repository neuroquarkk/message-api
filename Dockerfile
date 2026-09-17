FROM golang:1.27.0 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server


FROM gcr.io/distroless/static-debian13 AS run
WORKDIR /app

COPY --from=build /app/index.html .
COPY --from=build /app/server ./server
USER nonroot:nonroot

CMD [ "./server" ]
