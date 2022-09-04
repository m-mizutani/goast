FROM golang:1.19 AS build-go
ADD . /src
WORKDIR /src
RUN go build -o goast .

FROM gcr.io/distroless/base
COPY --from=build-go /src/goast /goast
WORKDIR /
ENTRYPOINT ["/goast"]
