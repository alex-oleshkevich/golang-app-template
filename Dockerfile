# syntax=docker/dockerfile:1

FROM golang:1.27.1-alpine3.24 AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates && mkdir /data

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w" \
    -o /out/nisaba ./cmd/nisaba

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/nisaba /nisaba
# A new named volume copies this dir's ownership, so the nonroot user can write the database.
COPY --from=build --chown=nonroot:nonroot /data /data

USER nonroot:nonroot

ENV NISABA_DB=/data/nisaba.db
EXPOSE 8080

ENTRYPOINT ["/nisaba"]
CMD ["serve", "--address", "0.0.0.0", "--port", "8080"]
