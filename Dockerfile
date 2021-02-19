FROM klakegg/hugo:latest-ext AS website-builder

ADD ./website /data
RUN hugo --source=/data --destination=/static


FROM golang:alpine AS go-builder
RUN apk add gcc musl-dev

WORKDIR /build
ADD server .
RUN rm -rf static
COPY --from=website-builder /static static
RUN go test ./...
RUN go build -o lcl-service main.go

FROM alpine:latest
WORKDIR /app
ADD subdomain_aliases.yml /app/subdomain_aliases.yml
COPY --from=go-builder /build/lcl-service /app/lcl-service

EXPOSE 8080
ENTRYPOINT ["/app/lcl-service"]
