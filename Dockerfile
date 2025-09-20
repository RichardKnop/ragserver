ARG ONNXRUNTIME_VERSION=1.22.0
ARG GO_VERSION=1.25.1
ARG BUILD_PLATFORM=linux/amd64

FROM --platform=$BUILD_PLATFORM public.ecr.aws/amazonlinux/amazonlinux:2023 AS ragserver-runtime
ARG ONNXRUNTIME_VERSION
ARG GO_VERSION
ARG CMD_PKG

ENV PATH="$PATH:/usr/local/go/bin" \
    GOPJRT_NOSUDO=1

COPY ./scripts/download-onnxruntime.sh /download-onnxruntime.sh
RUN --mount=src=./go.mod,dst=/go.mod \
    dnf --allowerasing -y install gcc jq bash tar xz gzip glibc-static libstdc++ wget zip git dirmngr sudo which && \
    ln -s /usr/lib64/libstdc++.so.6 /usr/lib64/libstdc++.so && \
    dnf clean all && \
    # tokenizers
    tokenizer_version=$(grep 'github.com/daulet/tokenizers' /go.mod | awk '{print $2}') && \
    tokenizer_version=$(echo $tokenizer_version | awk -F'-' '{print $NF}') && \
    echo "tokenizer_version: $tokenizer_version" && \
    curl -LO https://github.com/daulet/tokenizers/releases/download/${tokenizer_version}/libtokenizers.linux-amd64.tar.gz && \
    tar -C /usr/lib -xzf libtokenizers.linux-amd64.tar.gz && \
    rm libtokenizers.linux-amd64.tar.gz && \
    # onnxruntime cpu
    sed -i 's/\r//g' /download-onnxruntime.sh && chmod +x /download-onnxruntime.sh && \
    /download-onnxruntime.sh ${ONNXRUNTIME_VERSION} && \
    # XLA/goMLX
    curl -sSf https://raw.githubusercontent.com/gomlx/gopjrt/main/cmd/install_linux_amd64_amazonlinux.sh | bash && \
    # go
    curl -LO https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz

# NON-PRIVILEGED USER
# create non-privileged ragserver user with id: 1000
RUN useradd -u 1000 -m ragserver && usermod -a -G wheel ragserver && \
    echo "ragserver ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers.d/ragserver

COPY . /build
WORKDIR /build
RUN cp -R db/ /db && chown -R 1000:1000 /db
RUN mkdir /data && chown -R 1000:1000 /data
RUN mkdir /models && chown -R 1000:1000 /models
RUN mkdir /files && chown -R 1000:1000 /files
RUN mkdir /templates && chown -R 1000:1000 /templates
RUN cd ${CMD_PKG} && CGO_ENABLED=1 CGO_LDFLAGS="-L/usr/lib/" GOOS=linux GOARCH=amd64 go build -tags "ALL" -a -o /ragserver main.go
RUN chown 1000:1000 /ragserver
WORKDIR /
RUN rm -rf /build

EXPOSE 8080

USER 1000:1000

CMD ["/ragserver"]
