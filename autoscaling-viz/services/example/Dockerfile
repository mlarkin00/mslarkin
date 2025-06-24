FROM golang:1.21 AS build

WORKDIR /go/src/app
COPY go.* /go/src/app/
RUN go mod download

COPY . /go/src/app/
RUN CGO_ENABLED=0 go build -o /go/bin/app

FROM gcr.io/distroless/base-debian10:nonroot AS run

COPY --from=build /go/bin/app /

ENTRYPOINT [ "/app" ]

