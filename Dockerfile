FROM golang:1.26 AS build-go
ADD . /src
WORKDIR /src
RUN go build -o goast ./cmd/goast
# Bundle the license texts of every dependency into the image (pinned for reproducibility).
RUN go run github.com/google/go-licenses@v1.6.0 save ./cmd/goast --save_path=/licenses --force

FROM gcr.io/distroless/base
COPY --from=build-go /src/goast /goast
COPY --from=build-go /licenses /licenses
WORKDIR /
ENTRYPOINT ["/goast"]
