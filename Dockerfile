# Copyright 2020 VMware, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

### Builder Image ###
FROM golang:1.13-alpine as builder

# Install dependencies
RUN apk update \
    && apk add --no-cache git mercurial openssh ca-certificates

# Create appuser.
ENV USER=appuser
ENV UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

# Go build env settings
ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED=0
ENV GOPROXY="https://proxy.golang.org,direct"

WORKDIR /app

# Cache go modules for CI
COPY go.mod go.sum ./
RUN go mod download

# Copy project files and build
COPY . .
RUN go build -o ./bin/generic-sidecar-injector


### Final Image ###
FROM scratch
WORKDIR /app

# Copy user, group and cert files
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary
COPY --from=builder /app/bin/generic-sidecar-injector .

# Use an unprivileged user.
USER appuser:appuser

CMD ["./generic-sidecar-injector"]
