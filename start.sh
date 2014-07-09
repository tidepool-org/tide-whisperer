#! /bin/bash -eu

if [ -r config/goenv.sh ]; then
  . config/goenv.sh
fi
exec ./tide-whisperer