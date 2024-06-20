#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=$(go env GOPATH)/pkg/mod/k8s.io/code-generator@$(go list -m k8s.io/code-generator | awk '{print $2}')

source "${CODEGEN_PKG}/kube_codegen.sh"

THIS_PKG="github.com/jd-opensource/joylive-injector"

kube::codegen::gen_helpers \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}"

kube::codegen::gen_client \
    --with-watch \
    --with-applyconfig \
    --output-dir "${SCRIPT_ROOT}/client-go" \
    --output-pkg "${THIS_PKG}/client-go" \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}/client-go/apis"
