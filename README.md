# Build


    docker build . --platform linux/amd64 -t kalys/tiligram:latest
    docker push kalys/tiligram:latest

On the server:

    docker compose up -d

## Old

    env GOOS=linux GOARCH=amd64 go build
