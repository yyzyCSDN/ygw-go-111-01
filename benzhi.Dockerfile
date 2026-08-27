FROM golang:1.23.12 AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .
ENV GOPROXY=off GOSUMDB=off CGO_ENABLED=0
RUN go build -mod=vendor -o /out/server ./cmd/server

FROM golang:1.23.12
ENV GOPROXY=off GOSUMDB=off
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .
COPY --from=build /out/server /app/server
EXPOSE 8090
CMD ["/app/server", "-addr", "0.0.0.0:8090", "-dir", "/app/data"]
