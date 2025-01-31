#!/bin/bash

while true; do
  echo $$ > $HOME/certbot-renew.pid
  echo
  date
  certbot renew -n
  sleep 7d
done
