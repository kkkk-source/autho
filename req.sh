#!/usr/bin/bash

curl \
	-d '{"username":"root","password":"toor"}' \
	-H "Content-Type: application/json" \
	-X POST http://localhost:8080/login

curl \
	http://localhost:8080/say \
	-H "Accept: application/json" \
	-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NfdXVpZCI6IjVlY2Q3OGQ5LTg2YzktNGExYy05MWNmLTNlOWZmMjY1MzhhNCIsImF1dGhvcml6ZWQiOnRydWUsImV4cCI6MTYyNTkyMDgzMywidXNlcl9pZCI6MX0.SCP7P-fss3wBPlDZHQSR-5C6xr8i0pTvJcb6pAxzWPw"
