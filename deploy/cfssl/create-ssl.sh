#!/usr/bin/env bash

set -ex

rm -f *.pem *.csr

cfssl gencert --initca=true dac-ca-csr.json | cfssljson --bare dac-ca
cfssl gencert --ca dac-ca.pem --ca-key dac-ca-key.pem --config dac-gencert.json dac-csr.json | cfssljson --bare dac
