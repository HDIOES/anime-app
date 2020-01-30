FROM debian:stretch
COPY anime-app ./
COPY settings.json ./
COPY migrations/ ./ 
ENTRYPOINT ["./anime-app"]
