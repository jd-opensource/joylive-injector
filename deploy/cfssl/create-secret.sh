#!/usr/bin/env bash

set -ex

#kubectl create secret generic \
#        dynamic-admission-control-certs \
#        --namespace joylive \
#        --from-file=dac.pem \
#        --from-file=dac-key.pem

CA_BUNDLE=$(cat dac-ca.pem | base64 | tr -d '\n')
sed -i '' "s@\${CA_BUNDLE}@${CA_BUNDLE}@g" ../*/*.yaml

CA_KEY_BUNDLE=$(cat dac-key.pem | base64 | tr -d '\n')
sed -i '' "s@\${CA_KEY_BUNDLE}@${CA_KEY_BUNDLE}@g" ../*/*.yaml

CA_PUB_BUNDLE=$(cat dac.pem | base64 | tr -d '\n')
sed -i '' "s@\${CA_PUB_BUNDLE}@${CA_PUB_BUNDLE}@g" ../*/*.yaml