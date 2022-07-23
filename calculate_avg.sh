#!/bin/bash

if [[ -z $1 || ! -f $1 ]]; then
    echo "Usage: $0 <file_path>"
    exit 1
fi

sum=0.0
num_iterations=0
while read -r time_ms; do
    echo $time_ms
    sum=$(echo "$sum+$time_ms" | bc)
    num_iterations=$(( num_iterations + 1 ))
done < $1

echo "num_iterations: $num_iterations"
echo "average: `bc <<< "scale=3; $sum / $num_iterations"` ms"
