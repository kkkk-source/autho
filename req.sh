#!/usr/bin/bash

hostname='http://localhost:8080'
tmpfile='token_v1'

# request a token from login
curl \
	-d '{"username":"abcd","password":"dcba"}' \
	-H "Content-Type: application/json" \
	-X POST $hostname/login > $tmpfile 2> /dev/null

# get the access token
access_token=$(awk -F "\"" '{print $4}' $tmpfile)

# use the access token to access the resource
curl \
	$hostname/say \
	-H "Accept: application/json" \
	-H "Authorization: Bearer $access_token"

# use the access token to logged out from the server.
curl \
	$hostname/logout \
	-H "Accept: application/json" \
	-H "Authorization: Bearer $access_token"

# question: if i'm already logged out, can/should i use the refresh_token to
# request an access token?
#
# answer: can    [yes]
#         should [idk]

# get the refresh token
refresh_token=$(awk -F "\"" '{print $8}' $tmpfile)

# use the refresh_token to request other access token
curl \
	-d "{\"refresh_token\":\"$refresh_token\"}" \
	-H "Accept: application/json" \
	-X POST $hostname/refresh

