#!/bin/bash

# A simple bash scripts that generates k8s types for a series
# of kubernetes releases.
# The generated objects are then added to a local checkout of the
# github.com/kubewarden/k8s-objects repository

set -e

OUT_DIR=~/k8s-data-types
GIT_DIR=~/checkout/kubernetes/kubewarden/k8s-objects

for KUBEMINOR in {14..24}
do
  echo ==================================
  echo PROCESSING KUBERNETES 1.$KUBEMINOR
  echo ==================================

  ./k8s-objects-generator -kube-version 1.$KUBEMINOR -o $OUT_DIR

  BRANCH=release-1.$KUBEMINOR

  cd $GIT_DIR
  if [ $((n=$(git branch | grep -wic "$BRANCH"))) -ge 0 ]; then
    git checkout $BRANCH
    GIT_COMMIT_MSG="Update definitions"
    GIT_TAG="v1.$KUBEMINOR.0-kw$((n+1))"
  else
    git checkout --orphan $BRANCH
    GIT_COMMIT_MSG="initial release"
    GIT_TAG="v1.$KUBEMINOR.0-kw1"
  fi
  git reset --hard
  cp -r $OUT_DIR/src/github.com/kubewarden/k8s-objects/* $GIT_DIR
  git add *
  git commit -m "$GIT_COMMIT_MSG"
  git tag $GIT_TAG
  cd -
done
