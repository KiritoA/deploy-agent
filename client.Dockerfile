FROM debian:buster-slim

RUN apt-get update && \ 
    apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg-agent \
    software-properties-common

ADD ./build/deploy-agent-client /usr/local/bin

RUN chmod 755 /usr/local/bin/deploy-agent-client
