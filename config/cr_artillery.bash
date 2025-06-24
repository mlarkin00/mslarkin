#!/bin/bash
export ID_TOKEN="$(gcloud auth print-identity-token)"
artillery run ~/workspace/msl-playground/config/artillery-cr-test.yaml
