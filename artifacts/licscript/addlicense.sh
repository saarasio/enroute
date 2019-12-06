#!/bin/bash  
for x in $*; do  
head -$LICENSELEN $x | diff $LIC_PATH/license.txt - || ( ( cat $LIC_PATH/license.txt; echo; cat $x) > /tmp/file;  
mv /tmp/file $x )  
done
