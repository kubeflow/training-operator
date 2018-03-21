#!/bin/sh

# This script will sleep a random seconds,
# Then exit a random code

random_sleep=`shuf -i 10-30 -n 1`
echo "sleep $random_sleep"
sleep $random_sleep

random_exit=`shuf -i 0-3 -n 1`
echo "exit $random_exit"
exit $random_exit
