#!/bin/bash
FILE=$1_Shiviz.log
echo '(?<host>\S*) (?<clock>{.*})\n(?<event>.*)' > $FILE
echo -e "\n" >> $FILE
cat $1*g.txt >> $FILE

