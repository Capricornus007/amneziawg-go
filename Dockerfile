# ── Go 吃尖端（golang/go master），不吃發布版 ────────────────────────────────
# 用戶 2026-10-11：「docker 的隨便，但我必須看到結果運行的包含最尖端」。官方不提供
# 未發佈版本的二進位，所以用 base 镜像那顆發布版 Go 當 GOROOT_BOOTSTRAP 從源碼編尖端，
# 編好裝進 /usr/local/go 蓋掉發布版。
# 省時：編好的 GOROOT 以 commit 為鍵存進 BuildKit cache mount（/opt/gotip-cache），
# 同一個尖端只編一次（那 15-25 分鐘），之後每次 build 只是幾秒的 cp。
# GO_TIP_SHA 一定要傳、沒傳就紅：留空靠 layer 快取的話，命中時會拿到「上一次的尖端」
# 還聲稱是尖端，那是騙人。取法：git ls-remote https://github.com/golang/go master | cut -f1
# （.github/workflows/build-if-tag.yml 這樣算出來再 --build-arg 帶進來。）
# 結尾兩道硬驗（toolchain 一道、產物一道）：`go version` 沒帶 -devel 就讓 build 失敗，
# 不准退回發布版、不准靜默放過。
FROM --platform=$BUILDPLATFORM golang:1.27-bookworm as awg
ARG GO_TIP_SHA
RUN --mount=type=cache,target=/opt/gotip-cache \
    --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    set -eux; \
    test -n "${GO_TIP_SHA}" || { echo "ERROR: 沒傳 --build-arg GO_TIP_SHA=<golang/go master 的 commit sha>"; exit 1; }; \
    apt-get update; \
    apt-get install -y --no-install-recommends git ca-certificates; \
    tip="/opt/gotip-cache/${GO_TIP_SHA}/go"; \
    if [ ! -x "$tip/bin/go" ]; then \
        git init -q /tmp/gosrc; \
        git -C /tmp/gosrc remote add origin https://github.com/golang/go; \
        git -C /tmp/gosrc fetch -q --depth 1 origin "${GO_TIP_SHA}"; \
        git -C /tmp/gosrc checkout -q --detach FETCH_HEAD; \
        ( cd /tmp/gosrc/src && GOTOOLCHAIN=local GOROOT_BOOTSTRAP=/usr/local/go ./make.bash ) || { echo "ERROR: 尖端的 make.bash 失敗"; exit 1; }; \
        mkdir -p "$(dirname "$tip")"; \
        mv /tmp/gosrc "$tip"; \
    fi; \
    rm -rf /usr/local/go; \
    cp -a "$tip" /usr/local/go; \
    go version; \
    go version | grep -q -- '-devel' || { echo "ERROR: 裝進 /usr/local/go 的不是尖端（go version 沒帶 -devel）"; exit 1; }
# 不許 go 去下載發布版頂替尖端（同 feed 倉 setup-feed-go 的釘法）。
ENV GOTOOLCHAIN=local
COPY . /awg
WORKDIR /awg
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    set -eux; \
    go mod download; \
    go mod verify; \
    go build -ldflags '-linkmode external -extldflags "-fno-PIC -static"' -v -o /usr/bin; \
    go version /usr/bin/amneziawg-go
RUN go version /usr/bin/amneziawg-go | grep -q -- '-devel' \
    || { echo "ERROR: 產物 amneziawg-go 不是尖端編出來的"; exit 1; }

FROM alpine:3.19 as tools
ARG AWGTOOLS_COMMIT="v3.1.20260812"

RUN apk add --no-cache git build-base linux-headers && \
    git clone https://github.com/amnezia-vpn/amneziawg-tools.git /amneziawg-tools && \
    cd /amneziawg-tools && git checkout ${AWGTOOLS_COMMIT} && \
    cd src && make

FROM alpine:3.19

RUN apk --no-cache add iproute2 iptables bash
COPY --from=tools /amneziawg-tools/src/wg /usr/bin/awg
COPY --from=tools /amneziawg-tools/src/wg-quick/linux.bash /usr/bin/awg-quick
RUN chmod +x /usr/bin/awg /usr/bin/awg-quick && \
    ln -s /usr/bin/awg /usr/bin/wg && \
    ln -s /usr/bin/awg-quick /usr/bin/wg-quick
COPY --from=awg /usr/bin/amneziawg-go /usr/bin/amneziawg-go
