#!/bin/sh

if [ $# -ne 1 ]; then
  echo "File name is required" 1>&2
  exit 1
fi

TS=`date "+%s"`
DATE=`date "+%Y-%m-%d %H:%M:%S"`

echo "-- $DATE" > "./migrations/${TS}_${1}.up.sql"
echo "-- $DATE" > "./migrations/${TS}_${1}.down.sql"
