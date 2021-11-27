FROM golang:1.16-alpine3.12 as builder

RUN mkdir /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
ADD . /src
WORKDIR /src
RUN  GOPROXY=https://goproxy.cn go build -o application-auto-scaling-service cmd/application-auto-scaling-service/application-auto-scaling-service.go  && chmod +x application-auto-scaling-service

FROM alpine:3.12
ENV ZONEINFO=/app/zoneinfo.zip
RUN mkdir /app
WORKDIR /app

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /app
COPY --from=builder /src/application-auto-scaling-service /app
RUN mkdir /app/resources
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo "Asia/Shanghai" > /etc/timezone

ENTRYPOINT  ["./application-auto-scaling-service"]
#EXPOSE 80
