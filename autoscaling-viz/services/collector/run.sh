#!/bin/bash
env $(cat .env | xargs) go run main.go