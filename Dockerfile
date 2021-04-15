#  MIT License
#
#  (C) Copyright [2019-2021] Hewlett Packard Enterprise Development LP
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
# Dockerfile for building hms-meds.

## Prepare Builder ##
FROM arti.dev.cray.com/baseos/golang:1.14-alpine3.12 AS build-base

RUN set -ex \
    && apk update \
    && apk add build-base

FROM build-base AS base

# Copy all the necessary files to the image.
COPY cmd $GOPATH/src/stash.us.cray.com/HMS/hms-meds/cmd
COPY internal $GOPATH/src/stash.us.cray.com/HMS/hms-meds/internal
COPY vendor $GOPATH/src/stash.us.cray.com/HMS/hms-meds/vendor


### Build Stage ###
FROM base AS builder

# Now build
RUN set -ex \
    && go build -i -o /usr/local/bin/meds stash.us.cray.com/HMS/hms-meds/cmd/meds \
    && go build -i -o /usr/local/bin/vault_loader stash.us.cray.com/HMS/hms-meds/cmd/vault_loader


### Final Stage ###
FROM arti.dev.cray.com/baseos/alpine:3.12
LABEL maintainer="Cray, Inc."
STOPSIGNAL SIGTERM

# Setup environment variables.
ENV HSM_URL=https://api-gateway.default.svc.cluster.local/apis/smd/hsm/v1
ENV MEDS_OPTS=""

ENV VAULT_ADDR="http://cray-vault.vault:8200"
ENV VAULT_SKIP_VERIFY="true"

# These will be seen directly by MEDS, bypassing the cmdline
ENV MEDS_NTP_TARG=""
ENV MEDS_SYSLOG_TARG=""
ENV MEDS_NP_RF_URL="/redfish/v1/Managers/BMC/NetworkProtocol"
ENV MEDS_ROOT_SSH_KEY=""

# Include curl in the final image.
RUN set -ex \
    && apk update \
    && apk add --no-cache curl

# Copy built binaries from above build step.
COPY --from=builder /usr/local/bin/meds /usr/local/bin
COPY --from=builder /usr/local/bin/vault_loader /usr/local/bin

# Set up the command to start the service, the run the init script.
CMD meds -hsm ${HSM_URL} ${MEDS_OPTS}
