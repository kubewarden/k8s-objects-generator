#!/bin/bash

# A simple bash scripts that generates k8s types for a series
# of kubernetes releases.
# The generated objects are then added to a local checkout of the
# github.com/kubewarden/k8s-objects repository

set -e

OUT_DIR=~/k8s-data-types
GIT_DIR=~/checkout/kubernetes/kubewarden/k8s-objects

# Able to define Kubernetes versions range (for testing)
KUBERNETES_VERSION_MIN="${KUBEMINOR_MIN:-14}"
KUBERNETES_VERSION_MAX="${KUBEMINOR_MAX:-26}"

while [[ $# -gt 0 ]]; do
  case $1 in
    -m|--message)
      GIT_COMMIT_MSG_FILE="$(readlink -f "$2")"
      shift # past argument
      shift # past value
      ;;
    -g|--git-dir)
      GIT_DIR="$(readlink -f "$2")"
      shift # past argument
      shift # past value
      ;;
    -o|--out-dir)
      OUT_DIR="$(readlink -f "$2")"
      shift # past argument
      shift # past value
      ;;
    -*|--*)
      echo "Unknown option $1"
      exit 1
      ;;
    *)
      echo "Invalid positional arg"
      exit 1
      ;;
  esac
done

if [ -z "$GIT_COMMIT_MSG_FILE" ]; then
  echo "git commit message must be provided via the -m flag"
  exit 1
fi

make build
for KUBEMINOR in $(eval "echo {$KUBERNETES_VERSION_MIN..$KUBERNETES_VERSION_MAX}");
do
  echo ==================================
  echo PROCESSING KUBERNETES "1.$KUBEMINOR"
  echo ==================================

  ./k8s-objects-generator -kube-version "1.$KUBEMINOR" -o "$OUT_DIR"

  BRANCH=release-1.$KUBEMINOR

  cd "$GIT_DIR"
  git checkout main
  upstream=$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' | awk -F "/" '{print $1}')
  git fetch "$upstream"

  if [ $((n=$(git branch -r | grep -wic "$BRANCH"))) -gt 0 ]; then
    git checkout "$BRANCH"
    git rebase "$upstream"/"$BRANCH" "$BRANCH"

    n=$(git tag | grep -wic "v1.$KUBEMINOR")

    GIT_TAG="v1.$KUBEMINOR.0-kw$((n+1))"
  else
    git checkout --orphan "$BRANCH"
    GIT_TAG="v1.$KUBEMINOR.0-kw1"
  fi
  rsync -av --exclude '.git' --delete-after "$OUT_DIR"/src/github.com/kubewarden/k8s-objects/ "$GIT_DIR"
  golangci-lint run ./...
  git add .
  git commit -F "$GIT_COMMIT_MSG_FILE"
  git tag -s -a -m "$GIT_TAG" "$GIT_TAG"
  cd -
done
