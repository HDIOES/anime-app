#!/bin/bash

dep ensure
go build -o anime-app
docker build -t ivantimofeev/anime-app .
docker push ivantimofeev/anime-app

