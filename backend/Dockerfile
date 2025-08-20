FROM alpine:latest
WORKDIR /app
COPY app/app .
COPY app/static/ static/
EXPOSE 8080
ENTRYPOINT ["/app/app"]
