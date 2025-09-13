FROM ubuntu:latest

RUN apt update && apt install -y bash-completion && apt clean

RUN mkdir -p /etc/bash_completion.d
RUN echo "if [ -f /etc/bash_completion ]; then . /etc/bash_completion; fi" >> /root/.bashrc