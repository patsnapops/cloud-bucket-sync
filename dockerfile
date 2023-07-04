FROM centos:7

ENV TZ "Asia/Shanghai"

COPY ./cbs /usr/local/bin/cbs
COPY ./entrypoint.sh /root/entrypoint.sh

WORKDIR /root
RUN yum install -y bind-utils && \
    yum clean all && \
    rm -rf /var/cache/yum/* && \
    chmod +x /usr/local/bin/cbs && \
    chmod +x /root/entrypoint.sh 

USER root

ENTRYPOINT /root/entrypoint.sh