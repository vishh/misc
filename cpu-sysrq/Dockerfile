FROM alpine:3.2
MAINTAINER vishh@google.com

ADD cpu-sysrq /usr/bin/cpu-sysrq

#ENTRYPOINT ["ls", "-all", "/usr/bin/*"]
ENTRYPOINT ["/usr/bin/cpu-sysrq", "-logtostderr"]