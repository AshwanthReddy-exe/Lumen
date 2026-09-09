FROM golang:1.27.1 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/lumen-host ./cmd/lumen-host

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/lumen-host /usr/local/bin/lumen-host
COPY --chown=nonroot:nonroot deploy/docker/data/ /var/lib/lumen/
ENV LUMEN_DATA_DIR=/var/lib/lumen
ENV LUMEN_OPERATOR_CREDENTIAL_FILE=/var/lib/lumen/operator.credential
ENV LUMEN_HERMES_BEARER_FILE=/var/lib/lumen/hermes.token
VOLUME ["/var/lib/lumen"]
STOPSIGNAL SIGTERM
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/lumen-host"]
CMD ["serve"]
# The foreground command is lumen-host serve. Run with --restart on-failure:3 and mount secret files read_only; credentials never belong in flags.
