#!/bin/bash

VERSION="$(git describe --tags --always --dirty)"
NAME=mac-loginwindow-png

echo "Building $NAME version $VERSION"

mkdir -p build

build() {
  echo -n "=> $1-$2: "
  GOOS=$1 GOARCH=$2 CGO_ENABLED=0 go build -o build/$NAME-$1-$2 -ldflags "\
      -X github.com/groob/side-projects/version.appName=${NAME} \
      -X github.com/groob/side-projects/version.version=${VERSION} \
      -X github.com/groob/side-projects/version.branch=$(git rev-parse --abbrev-ref HEAD) \
      -X github.com/groob/side-projects/version.goVersion=$(go version | awk '{print $3}') \
      -X github.com/groob/side-projects/version.buildDate=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
      -X github.com/groob/side-projects/version.buildUser=$(whoami) \
      -X github.com/groob/side-projects/version.revision=$(git rev-parse HEAD)" ./main.go
  du -h build/$NAME-$1-$2
}

dockerBuild() {
  image="gcr.io/$1/${NAME}:${VERSION}"
  docker build -t $image .
  gcloud docker -- push $image

  echo -n "pushed => $image: "
}

kubeDeploy() {
  image="gcr.io/$1/${NAME}:${VERSION}"

  kubectl set image deployment/"$NAME" "$NAME"="$image"
  echo -n "deployed => $image: "
}




while [[ $# -gt 0 ]]; do
  cmd="$1"
  opt="$2"
  case "${cmd}" in
    -B|--build)
      build "linux" "amd64"
      exit $?
      ;;
    -D|--docker)
      imageRepo="${opt:-groob-io}"
      dockerBuild $imageRepo
      exit $?
      ;;
    -K|--deploy)
      imageRepo="${opt:-groob-io}"
      kubeDeploy $imageRepo
      exit $?
      ;;
    *)
      echo "Error: Unknown command: ${cmd}"
      exit 1
      ;;
  esac
done
