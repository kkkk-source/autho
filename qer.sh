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

# get the refresh token
refresh_token=$(awk -F "\"" '{print $8}' $tmpfile)

# use the refresh_token to request other access token
curl \
	-d "{\"refresh_token\":\"$refresh_token\"}" \
	-H "Accept: application/json" \
	-X POST $hostname/refresh

# question: after the refresh token has been used to request other access
# token, the old access token (before it expires) can/should be use to access a
# private resouce?
#
# answer: can    [yes]
#         should [idk]

# use the access token to access the resource
curl \
	$hostname/say \
	-H "Accept: application/json" \
	-H "Authorization: Bearer $access_token"
