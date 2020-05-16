FROM debian:buster-slim

ADD ./build/deploy-agent-client /usr/local/bin
ADD ./build/deploy-agent-login /usr/local/bin

RUN chmod 755 /usr/local/bin/deploy-agent-client
RUN chmod 755 /usr/local/bin/deploy-agent-login
