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
  cd $GIT_DIR
  git checkout --orphan release-1.$KUBEMINOR
  git reset --hard
  cp -r $OUT_DIR/src/github.com/kubewarden/k8s-objects/* $GIT_DIR
  git add *
  git commit -a -m "initial release"
  git tag v1.$KUBEMINOR.0+kw1
  cd -
done
