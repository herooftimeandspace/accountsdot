FROM golang:1.26.3 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/provisioner ./cmd/provisioner

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/provisioner /provisioner
COPY --from=build /src/docs/reference-inputs /app/docs/reference-inputs

ENV WIZARD_REFERENCE_INPUT_ROOT=/app

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/provisioner"]
