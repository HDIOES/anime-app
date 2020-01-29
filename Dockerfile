FROM debian:stretch
COPY anime-app migrations/ ./
ENTRYPOINT ["./anime-app"]
