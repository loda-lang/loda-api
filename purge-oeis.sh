#!/bin/bash

folder=$HOME/data/oeis/b

while true; do
  avail=$(df --total --output=avail $folder | tail -n 1)
  echo $avail
  if [ $avail -gt 2097152 ]; then
    break
  fi
  id=$(( $RANDOM % 350 ))
  id=$(printf %03d $id)
  echo "Deleting $folder/$id"
  sudo rm -rf $folder/$id
done
