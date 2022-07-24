#!/bin/bash

sandbox_type=$(cat `pwd`/default-ol/config.json | jq -r .sandbox)
logfile=${sandbox_type}-mem-$(date +%FT%T).log
interval=2

watch -n $interval -t "free -m | awk 'NR==2 {print \$3}' | tee $logfile -a"
