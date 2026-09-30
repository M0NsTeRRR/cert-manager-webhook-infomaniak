FROM golang:1.27.1@sha256:512690a5660563b57d37ecc31129e7f136e831db2aed24a1dbeb8ad7380dc0fa AS builder

ARG VERSION=development
ARG SOURCE_DATE_EPOCH=0

WORKDIR /go/src/app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -trimpath -a -o cert-manager-webhook-infomaniak -ldflags "-w -X main.version=$VERSION -X main.buildTime=$SOURCE_DATE_EPOCH -extldflags '-static'" cmd/webhook/main.go

FROM gcr.io/distroless/static:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3

COPY --from=builder /go/src/app/cert-manager-webhook-infomaniak /bin/cert-manager-webhook-infomaniak

USER 65532

ENTRYPOINT ["/bin/cert-manager-webhook-infomaniak"]
