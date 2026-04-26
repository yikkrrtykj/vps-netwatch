FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN go build -o /out/dashboard ./cmd/dashboard
RUN go build -o /out/agent ./cmd/agent
RUN go build -o /out/collector ./cmd/collector

FROM alpine:3.20
WORKDIR /opt/vps-netwatch
RUN apk add --no-cache ca-certificates
COPY --from=build /out/dashboard /usr/local/bin/vps-netwatch-dashboard
COPY --from=build /src/web/dist ./web/dist
COPY config.example.yaml ./config.example.yaml
EXPOSE 8787
CMD ["vps-netwatch-dashboard", "-config", "/opt/vps-netwatch/config.yaml"]
