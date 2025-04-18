#  MIT License
#
#  (C) Copyright [2019-2021,2025] Hewlett Packard Enterprise Development LP
#
#  Permission is hereby granted, free of charge, to any person obtaining a
#  copy of this software and associated documentation files (the "Software"),
#  to deal in the Software without restriction, including without limitation
#  the rights to use, copy, modify, merge, publish, distribute, sublicense,
#  and/or sell copies of the Software, and to permit persons to whom the
#  Software is furnished to do so, subject to the following conditions:
#
#  The above copyright notice and this permission notice shall be included
#  in all copies or substantial portions of the Software.
#
#  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
#  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
#  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
#  THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
#  OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
#  ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
#  OTHER DEALINGS IN THE SOFTWARE.

FROM artifactory.algol60.net/docker.io/alpine:3.21 AS build-base

# set a directory for the app
WORKDIR /usr/src/app

# copy all the files to the container
COPY tests/dummy-endpoint .

# install dependencies
RUN set -ex \
    && apk -U upgrade \
    && apk add \
        build-base \
        python3 \
        python3-dev \
        py3-pip \
        openssl \
        openssl-dev \
        libffi-dev \
        gcc \
        musl-dev \
        cargo

RUN pip3 install --upgrade pip \
    && pip3 install --no-cache-dir -r requirements.txt

# define the port number the container should expose
EXPOSE 80

ENV FLASK_APP=endpoint.py

# run the command
#CMD ["flask", "run"]
CMD ["python3", "endpoint.py"]
