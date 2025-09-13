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

# Create the agent agent_CA certs.
openssl req -x509                                     \
  -newkey rsa:4096                                    \
  -nodes                                              \
  -days 3650                                          \
  -keyout agent_ca_key.pem                           \
  -out agent_ca_cert.pem                             \
  -subj /C=FR/O=gRPC/CN=test-agent_ca/   \
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

# Generate a agent certs.
for i in 1 2
do
    openssl genrsa -out agent${i}_key.pem 4096
    openssl req -new                                    \
      -key agent${i}_key.pem                               \
      -days 3650                                        \
      -out agent${i}_csr.pem                               \
      -subj /C=FR/O=gRPC/CN=test-agent1/   \
      -config ./openssl.cnf                             \
      -reqexts test_agent
    openssl x509 -req           \
      -in agent${i}_csr.pem        \
      -CAkey agent_ca_key.pem  \
      -CA agent_ca_cert.pem    \
      -days 3650                \
      -set_serial 1000          \
      -out agent${i}_cert.pem      \
      -extfile ./openssl.cnf    \
      -extensions test_agent   \
      -sha256
    openssl verify -verbose -CAfile agent_ca_cert.pem  agent${i}_cert.pem
done
rm *_csr.pem