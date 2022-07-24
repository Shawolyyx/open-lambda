#!/bin/bash

sandbox_type=$(cat `pwd`/default-ol/config.json | jq -r .sandbox)
logfile=${sandbox_type}-mem-$(date +%FT%T).log
interval=0.5

if [[ ! -z $1 ]]; then
    interval=$1
fi

watch -n $interval -t "free -m | awk 'NR==2 {print \$3}' | tee $logfile -a"
