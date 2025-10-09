FROM gcr.io/oss-fuzz-base/base-builder-go

COPY . $SRC/project
WORKDIR $SRC/project

COPY ./.clusterfuzzlite/build.sh /src/
