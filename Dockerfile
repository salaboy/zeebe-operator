FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY /workspace/manager .
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
