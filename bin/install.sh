#!/bin/bash

WEB_ROOT=/var/www/ottomap.mdhenderson.com
echo " info: WEB_ROOT     == '${WEB_ROOT}'"

if [ ! -d "${WEB_ROOT}" ]; then
  echo "error: missing web root"
  exit 2
elif [ ! -d "${WEB_ROOT}/bin" ]; then
  echo "error: missing web root bin"
  exit 2
elif [ ! -f "${WEB_ROOT}/ottoweb.tgz" ]; then
  echo "error: missing tarball"
  exit 2
fi

echo " info: setting def to web root..."
cd "${WEB_ROOT}"  || exit 2


if [ -f "${WEB_ROOT}/bin/ottoweb" ]; then
  echo " info: removing old executable..."
  rm "${WEB_ROOT}/bin/ottoweb" || exit 2
fi

echo " info: extracting tarball..."
tar xzf ottoweb.tgz || exit 2
mv ottoweb.exe "${WEB_ROOT}/bin/ottoweb" || exit 2

echo " info: forcing bits on executable..."
chmod 755 "${WEB_ROOT}/bin/ottoweb" || exit 2

echo " info: testing executable..."
"${WEB_ROOT}/bin/ottoweb" version || exit 2

echo " info: removing tarball..."
rm ottoweb.tgz || exit 2

echo " info: installation completed successfully"
exit 0
