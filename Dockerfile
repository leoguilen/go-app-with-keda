# syntax=docker/dockerfile:1

FROM golang:1.23 AS build-stage
ARG APP_NAME

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY */${APP_NAME}/*.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin

FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /app/bin /app/bin

USER nonroot:nonroot

ENTRYPOINT ["/app/bin"]