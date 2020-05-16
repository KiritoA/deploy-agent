FROM debian:buster-slim

ADD ./build/deploy-agent-server /usr/local/bin

RUN chmod 755 /usr/local/bin/deploy-agent-server

ENTRYPOINT [ "/usr/local/bin/deploy-agent-server" ]