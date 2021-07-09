#!/usr/bin/bash

curl \
	-d '{"username":"root","password":"toor"}' \
	-H "Content-Type: application/json" \
	-X POST http://localhost:8080/login
