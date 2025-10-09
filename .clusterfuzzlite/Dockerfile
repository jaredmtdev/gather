FROM gcr.io/oss-fuzz-base/base-builder-go@sha256:07eceb4edd86392bf4ca640171839c5423688c9107dcf3278847ee4ec7cd5606

COPY . $SRC/project
WORKDIR $SRC/project

COPY ./.clusterfuzzlite/build.sh /src/
