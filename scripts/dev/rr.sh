#!/bin/bash 
# Recursive replace of ${1} with ${2}
# e.g. to replace package names
#eval egrep -lRZ $1 . | xargs -0 -l sed -i -e 
sub="egrep -lRZ $1 . | xargs -0 -l sed -i -e 's:${1}:${2}:g'"
eval $sub 



