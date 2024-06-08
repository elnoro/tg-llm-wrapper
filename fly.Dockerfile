FROM debian:bullseye-slim

RUN apt-get update && \
    apt-get install -y curl gnupg && \
    curl -L https://fly.io/install.sh | sh

ENV PATH="/root/.fly/bin:${PATH}"

WORKDIR /app

COPY . .

CMD ["flyctl", "deploy"]
