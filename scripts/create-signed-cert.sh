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
Create a signed certificate and store it in a Kubernetes "secret".

Steps in this script are based on https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/

usage: ${0} --context <k8s-context> [--openssl] [--cfssl]

The following flags can be use:

       --context          Kubernetes context [required].
       --openssl          Use openssl to issue the certificate [default].
       --cfssl            Use cfssl to issue the certificate.
EOF
  exit 1
}

while (("$#")); do
  case "$1" in
  --context)
    CONTEXT="$2"
    shift
    ;;
  --openssl)
    SSL_CLI="openssl"
    shift
    ;;
  --cfssl)
    SSL_CLI="cfssl"
    shift
    ;;
  *)
    shift
    ;;
  esac
done

# Check if context is set
[ -z "${CONTEXT}" ] && usage
[ -z "${SSL_CLI}" ] && SSL_CLI="openssl"

# Check if openssl/cfssl and kubectl exist
if [ ${SSL_CLI} = "openssl" ]; then
  command -v openssl >/dev/null || {
    echo "openssl command not found."
    exit 1
  }
elif [ ${SSL_CLI} = "cfssl" ]; then
  command -v cfssl >/dev/null || {
    echo "cfssl command not found."
    exit 1
  }
fi
command -v kubectl >/dev/null || {
  echo "openssl command not found."
  exit 1
}

# Set variables
SERVICE=generic-sidecar-injector
SECRET=generic-sidecar-injector-certs
NAMESPACE=default

# Switch kubernetes context
printf "\e[36m--> Changing kubernetes context\e[0m\n"
kubectl config use-context "$CONTEXT"

# Create temporary directory
TEMP_DIR=$(mktemp -d -t generic-injector)
printf "\e[36m--> Generating certs in \e[96m${TEMP_DIR}\e[0m\n"

# Create openssl configuration
if [ ${SSL_CLI} = "openssl" ]; then
  cat <<EOF >${TEMP_DIR}/openssl.cnf
[req]
default_bits       = 2048
req_extensions     = v3_req
distinguished_name = req_distinguished_name

[ req_distinguished_name ]

[ v3_req ]
basicConstraints = CA:FALSE
keyUsage         = digitalSignature, nonRepudiation, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName   = @alt_names

[ alt_names ]
DNS.1 = ${SERVICE}
DNS.2 = ${SERVICE}.${NAMESPACE}
DNS.3 = ${SERVICE}.${NAMESPACE}.svc
DNS.4 = ${SERVICE}.${NAMESPACE}.svc.cluster
EOF

  # Generate CSR using openssl
  openssl genrsa -out ${TEMP_DIR}/server-key.pem >/dev/null 2>&1 && printf "server-key.pem created\n"
  openssl req -new \
    -config ${TEMP_DIR}/openssl.cnf \
    -key ${TEMP_DIR}/server-key.pem \
    -subj "/CN=${SERVICE}.${NAMESPACE}.svc" \
    -out ${TEMP_DIR}/server.csr &&
    printf "server.csr created\n"

elif [ ${SSL_CLI} = "cfssl" ]; then
  cd ${TEMP_DIR}
  cat <<EOF | cfssl genkey -loglevel 4 - | cfssljson -bare server -loglevel 4 && printf "server-key.pem created\nserver.csr created\n"
{
  "hosts": [
    "${SERVICE}",
    "${SERVICE}.${NAMESPACE}",
    "${SERVICE}.${NAMESPACE}.svc",
    "${SERVICE}.${NAMESPACE}.svc.cluster"
  ],
  "CN": "${SERVICE}.${NAMESPACE}.svc",
  "key": {
    "algo": "ecdsa",
    "size": 256
  }
}
EOF
fi

# Delete the old CSR in Kubernetes if it exists
printf "\e[36m--> Deleting old CSR if it exists. \e[96m[You have 5 seconds to cancel]\e[0m\n"
sleep 5
kubectl delete csr ${SERVICE}.${NAMESPACE} 2>/dev/null || true

# Generate a CSR yaml blob and send it to the apiserver
printf "\e[36m--> Creating the certificate in Kubernetes\e[0m\n"
cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: ${SERVICE}.${NAMESPACE}
spec:
  groups:
  - system:authenticated
  request: $(cat ${TEMP_DIR}/server.csr | base64 | tr -d '\n')
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

# Approve the certificate
printf "\e[36m--> Approving the certificate in Kubernetes\e[0m\n"
kubectl certificate approve ${SERVICE}.${NAMESPACE}

# Save certificate to temporary folder
kubectl get csr ${SERVICE}.${NAMESPACE} -o jsonpath='{.status.certificate}' | base64 --decode >${TEMP_DIR}/server.crt

# Create secret with certificate and key
printf "\e[36m--> Creating a secret with certificate and key\e[0m\n"
cat <<EOF >${TEMP_DIR}/kustomization.yaml
secretGenerator:
- name: ${SECRET}
  namespace: ${NAMESPACE}
  files:
    - server-key.pem
    - server.crt
EOF
cat >>${TEMP_DIR}/kustomization.yaml <<EOF
generatorOptions:
 disableNameSuffixHash: true
EOF
kubectl apply -k ${TEMP_DIR}
