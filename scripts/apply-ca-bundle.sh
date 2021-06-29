#!/bin/bash
# Copyright 2020 VMware, Inc.
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

set -e

usage() {
  cat <<EOF
Add the CA bundle to the mutating webhook configuration manifest.

usage: ${0} --context <k8s-context>

The following flags are required:

       --context          Kubernetes context.
EOF
  exit 1
}

while [[ $# -gt 0 ]]; do
  case ${1} in
  --context)
    CONTEXT="$2"
    shift
    ;;
  *)
    usage
    ;;
  esac
  shift
done

# Check if context is set
[ -z "${CONTEXT}" ] && usage

# Check if jq and kubectl exist
command -v jq >/dev/null || {
  echo "jq command not found."
  exit 1
}
command -v kubectl >/dev/null || {
  echo "kubectl command not found."
  exit 1
}

# Get correct path
PROJECT_ROOT=$(dirname $(dirname $(realpath "${0}")))
DATA_DIR="${PROJECT_ROOT}/kubernetes"

# Get CA bundle from kubernetes
CA_BUNDLE=$(kubectl config view --raw -o json | jq -c --arg context "${CONTEXT}" '[ .clusters[] | select( .name | contains($context))]' | jq -r '.[0].cluster."certificate-authority-data"' | tr -d '"')
[ "${CA_BUNDLE}" = "null" ] && echo "Context ${CONTEXT} does not exist." && exit 1

# Add CA_BUNDLE to mutatingwebhookconfiguration.yaml
sed -e "s|{{ CA_BUNDLE }}|${CA_BUNDLE}|g" "${DATA_DIR}"/mutatingwebhookconfiguration.yaml.template >"$DATA_DIR"/mutatingwebhook.yaml
echo "Cluster ${CONTEXT} CA bundle added to mutating webhook configuration."
