FROM rust:1.30.0-stretch

ENV CROSVM_REPO=https://chromium.googlesource.com/chromiumos/platform/crosvm
ENV CROSVM_COMMIT=c527c1a7e8136dae1e8ae728dfd9932bf3967e7e
ENV MINIJAIL_REPO=https://android.googlesource.com/platform/external/minijail
ENV MINIJAIL_COMMIT=d45fc420bb8fd9d1fc9297174f3c344db8c20bbd

# Install deps
RUN apt-get update && apt-get install -y libcap-dev libfdt-dev

# Get source code
RUN git clone ${MINIJAIL_REPO} && \
    cd /minijail && \
    git checkout ${MINIJAIL_COMMIT} && \
    cd / && \
    git clone ${CROSVM_REPO} && \
    cd crosvm && \
    git checkout ${CROSVM_COMMIT}

# Compile and install minijail
WORKDIR /minijail
RUN make && \
    cp libminijail.so /usr/lib/ && \
    cp libminijail.h /usr/include/

# Compile crosvm
WORKDIR /crosvm
RUN cargo build --release
    
RUN mkdir /out && \
    cp /minijail/libminijail.so /out && \
    cp /crosvm/target/release/crosvm /out && \
    cp -r /crosvm/seccomp /out

WORKDIR /out
ENTRYPOINT ["tar", "cf", "-", "libminijail.so", "crosvm", "seccomp"]
