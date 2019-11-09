ARG PYTHON_VER=3.8
ARG ALPINE_VER=3.10

FROM python:${PYTHON_VER}-alpine${ALPINE_VER}

RUN apk add --no-cache \
      libressl-dev \
      libxml2-dev \
      libxslt-dev

ENV WORKDIR /app
COPY requirements.txt ${WORKDIR}/

ENV XMLSEC_VER=1.2.29

RUN apk add --no-cache --virtual .build-deps \
      build-base \
      libressl \
      libffi-dev && \
    cd /tmp && \
    wget http://www.aleksey.com/xmlsec/download/xmlsec1-${XMLSEC_VER}.tar.gz && \
    tar -xvf xmlsec1-${XMLSEC_VER}.tar.gz  && \
    cd xmlsec1-${XMLSEC_VER} && \
    ./configure --enable-crypto-dl=no && \
    make && \
    make install && \
    cd .. && \
    rm -rf /tmp/xmlsec* && \
    pip3 install --no-cache-dir -r ${WORKDIR}/requirements.txt && \
    apk del --no-cache .build-deps

RUN apk add --no-cache wireguard-tools-wg

WORKDIR ${WORKDIR}
COPY . .

ENV FLASK_APP ${WORKDIR}/index.py
ENTRYPOINT ${WORKDIR}/docker-entrypoint.sh

VOLUME ["${WORKDIR}/wireguard"]
EXPOSE 5000

