#!/bin/bash

if [[ -z $1 ]]; then
    echo "Usage: $0 <num_iterations>"
    exit 1
fi

num_iterations=$1

registry_dir=`pwd`/default-ol/registry
neutrino_registry_dir=`pwd`/../neutrino/registry
sandbox_type=$(cat `pwd`/default-ol/config.json | jq -r .sandbox)
log_file=${sandbox_type}-$(date +%FT%T).log

if [[ $sandbox_type == "graal" ]]; then
    file_ext=neutrino.so
else
    file_ext=py
fi

func=echo
ori_file=$func.$file_ext

# Link registry dir
if [[ ! -L $registry_dir ]]; then
    rm -rf $registry_dir
    ln -s $neutrino_registry_dir $registry_dir
fi

sum=0.0
for i in `seq $num_iterations`; do
    func_name=${func}$i
    new_file=$func_name.$file_ext
    if [[ ! -f $registry_dir/$new_file ]]; then
        cp $registry_dir/$ori_file $registry_dir/$new_file
    fi

    time_ms=$(python3 ./test-cs.py localhost 5000 $func_name $log_file)

    sum=$(echo "$sum+$time_ms" | bc)

    #rm -f $registry_dir/$new_file
done

#echo "sum: $sum"
echo "average: `bc <<< "scale=3; $sum / $num_iterations"` ms"
echo "log file: $log_file"
