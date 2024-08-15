# Build the manager binary
FROM golang:1.22 as builder

WORKDIR /workspace
# Build
RUN --mount=type=bind,target=. CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -a -o manager cmd/training-operator.v2alpha1/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
ENTRYPOINT ["/manager"]
