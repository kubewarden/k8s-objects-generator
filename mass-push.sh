#!/bin/bash

# A simple bash scripts that generates k8s types for a series
# of kubernetes releases.
# The generated objects are then added to a local checkout of the
# github.com/kubewarden/k8s-objects repository

set -ex

GIT_DIR=~/hacking/kubernetes/kubewarden/k8s-objects

for KUBEMINOR in {14..28}
do
  echo ==================================
  echo PROCESSING KUBERNETES 1.$KUBEMINOR
  echo ==================================

  BRANCH=release-1.$KUBEMINOR

  cd $GIT_DIR
  git checkout $BRANCH
  git push origin $BRANCH
done

git push --tags
