#!/bin/bash

BOOK_LIST_URL="https://github.com/cxumol/misc_script/raw/refs/heads/master/fichub/watchlist.json"
UA_API="http://headers.scrapeops.io/v1/user-agents?api_key=6ed5ed82-1938-4ba2-9495-5291c4596945"

book_urls=$(curl -s "$BOOK_LIST_URL" | jq -r '.[]')
user_agents=($(curl -s "$UA_API" | jq -r '.result[]'))

num_uas=${#user_agents[@]}
i=0

for url in $book_urls; do
    ua="${user_agents[$((i % num_uas))]}"
    curl -A "$ua" "https://fichub.net/api/v0/epub?q=$url"
    ((i++))
done
