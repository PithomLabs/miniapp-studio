# Phase D artifact-test toolchain image.
#
# Base pinned by digest (supply chain); content verified at use by
# TestToolchainImageContent (behavioral pin — rebuilds legitimately
# change the image id, so the id is recorded in evidence, not code).
# Pre-seeds the module cache with the pinned miniapp module and every
# package it contains, so GOPROXY=off builds resolve offline.
# Fail-closed grep: the two pinned go.sum lines must resolve.
FROM golang:1.27.3@sha256:e519d3b0a5b285e5aa53ab9eef02bc6bbb9cb04608389b743a136a9ce75f0d1e

# GOTOOLCHAIN=local fails closed instead of auto-downloading toolchains.
# GOPROXY/GOFLAGS/GOCACHE lockdowns are applied at RUN time by the
# runner (explicit allowlist); the image build itself needs network.
ENV GOTOOLCHAIN=local

RUN useradd -m -u 65532 tester
WORKDIR /seed
COPY seed/go.mod seed/main.go ./
RUN go mod tidy \
 && grep -qxF 'github.com/PithomLabs/miniapp v0.0.0-20261123181255-73351a801fc5 h1:Qz8Y8O57EGqPIT5EK1jmWJ4qiGiXSXrtMawIjqv9i64=' go.sum \
 && grep -qxF 'github.com/PithomLabs/miniapp v0.0.0-20261123181255-73351a801fc5/go.mod h1:CVJyU7TjwuIiKHNqXHPkkGlPt61ykKZM516NN2cnXjY=' go.sum \
 && GOBIN=/tmp/gobin go install github.com/PithomLabs/miniapp/...@v0.0.0-20261123181255-73351a801fc5 \
 && rm -rf /tmp/gobin /seed/main.go \
 && chmod -R a+rX /go/pkg/mod \
 && go clean -cache
LABEL studio.contract="studio-artifact/v1"
LABEL studio.toolchain.go="1.27.3"
LABEL studio.toolchain.miniapp="v0.0.0-20261123181255-73351a801fc5"
USER tester
WORKDIR /work
