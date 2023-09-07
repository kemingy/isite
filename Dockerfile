FROM golang:1.19-bookworm as builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download && go mod tidy

COPY . .
RUN make build

FROM ubuntu:22.04 as runner

COPY --from=builder /workspace/bin/isite /usr/local/bin/isite

ENTRYPOINT ["/usr/local/bin/isite"]
