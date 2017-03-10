#!/bin/bash

go build .
export CORE_CHAINCODE_ID_NAME=auction
export CORE_PEER_ADDRESS=0.0.0.0:7051
./auction &