#!/bin/bash
ps aux | grep kubectl
printf "%s " "Press enter to continue"
read ans
pkill kubectl -9     
ps aux | grep kubectl