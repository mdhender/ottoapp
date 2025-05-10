#!/bin/bash
############################################################################
# run ottomap for every clan in batch mode

############################################################################
#
cd /var/www/ottomap.mdhenderson.com/userdata || exit 0
for clan in ???? dummy; do
  [ -d "${clan}" ] || continue
  echo
  echo " info: processing clan '${clan}'"
  /var/www/ottomap.mdhenderson.com/bin/ottomap.sh "${clan}"
done
