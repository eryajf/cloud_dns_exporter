FROM registry.cn-hangzhou.aliyuncs.com/eryajf/golang:1.22.2-alpine3.19  AS builder

WORKDIR /app

ENV GOPROXY="https://goproxy.io"

RUN sed -i "s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g" /etc/apk/repositories \
    && apk upgrade && apk add --no-cache --virtual .build-deps \
    ca-certificates gcc g++ curl upx git make

ADD . .

RUN make build-linux && upx -9 cloud_dns_exporter

FROM registry.cn-hangzhou.aliyuncs.com/eryajf/alpine:3.19

WORKDIR /app

LABEL maintainer="eryajf"

COPY --from=builder /app/config.example.yaml config.yaml
COPY --from=builder /app/cloud_dns_exporter .

EXPOSE 21798

RUN chmod +x /app/cloud_dns_exporter

CMD [ "/app/cloud_dns_exporter" ]