#!/bin/bash

# adapted from: https://github.com/grpc/grpc-go/blob/master/examples/data/x509/create.sh

# Create the manager manager_CA certs.
openssl req -x509                                     \
  -newkey rsa:4096                                    \
  -nodes                                              \
  -days 3650                                          \
  -keyout manager_ca_key.pem                                  \
  -out manager_ca_cert.pem                                    \
  -subj /C=FR/O=gRPC/CN=test-manager_ca/   \
  -config ./openssl.cnf                               \
  -extensions test_ca                                 \
  -sha256

# Create the node node_CA certs.
openssl req -x509                                     \
  -newkey rsa:4096                                    \
  -nodes                                              \
  -days 3650                                          \
  -keyout node_ca_key.pem                           \
  -out node_ca_cert.pem                             \
  -subj /C=FR/O=gRPC/CN=test-node_ca/   \
  -config ./openssl.cnf                               \
  -extensions test_ca                                 \
  -sha256

# Generate a manager cert.
openssl genrsa -out manager_key.pem 4096
openssl req -new                                    \
  -key manager_key.pem                               \
  -days 3650                                        \
  -out manager_csr.pem                               \
  -subj /C=FR/O=gRPC/CN=test-manager1/   \
  -config ./openssl.cnf                             \
  -reqexts test_manager
openssl x509 -req           \
  -in manager_csr.pem        \
  -CAkey manager_ca_key.pem         \
  -CA manager_ca_cert.pem           \
  -days 3650                \
  -set_serial 1000          \
  -out manager_cert.pem      \
  -extfile ./openssl.cnf    \
  -extensions test_manager   \
  -sha256
openssl verify -verbose -CAfile manager_ca_cert.pem  manager_cert.pem

# Generate a node certs.
for i in 1 2
do
    openssl genrsa -out node${i}_key.pem 4096
    openssl req -new                                    \
      -key node${i}_key.pem                               \
      -days 3650                                        \
      -out node${i}_csr.pem                               \
      -subj /C=FR/O=gRPC/CN=test-node1/   \
      -config ./openssl.cnf                             \
      -reqexts test_node
    openssl x509 -req           \
      -in node${i}_csr.pem        \
      -CAkey node_ca_key.pem  \
      -CA node_ca_cert.pem    \
      -days 3650                \
      -set_serial 1000          \
      -out node${i}_cert.pem      \
      -extfile ./openssl.cnf    \
      -extensions test_node   \
      -sha256
    openssl verify -verbose -CAfile node_ca_cert.pem  node${i}_cert.pem
done
rm *_csr.pem