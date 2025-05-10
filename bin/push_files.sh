#!/bin/bash

# confirm that we're running from the root of the repository
[ -d assets ] || {
  echo error: must run from the root of the repository
  exit 2
}
[ -d components ] || {
  echo error: must run from the root of the repository
  exit 2
}

# build the executable
echo " info: building executable..."
GOOS=linux GOARCH=amd64 go build -o ottoweb.exe || exit 2

# create a compressed tarball of the assets and components.
# ensure that it doesn't contain Mac junk attributes.
echo " info: creating tarball..."
tar -cz --no-xattrs --no-mac-metadata -f ottoweb.tgz assets components ottoweb.exe || exit 2

# push the file to our production server
echo " info: pushing tarball..."
scp ottoweb.tgz mdhender@tribenet:/var/www/ottomap.mdhenderson.com/ottoweb.tgz || exit 2

# execute the installation script
echo " info: executing the installation script..."
ssh mdhender@tribenet /home/mdhender/bin/install.sh || {
  echo "error: installation script failed"
  exit 2
}

# next
echo " info: if this succeeded, you should restart the services"
echo "       ssh tribenet systemctl restart ottoweb.service"
echo "       ssh tribenet systemctl status  ottoweb.service"
echo "       ssh tribenet journalctl -f -u  ottoweb.service"

exit 0
