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
GOOS=linux GOARCH=amd64 go build -o ottoapp.exe || exit 2

# create a compressed tarball of the assets and components.
# ensure that it doesn't contain Mac junk attributes.
echo " info: creating tarball..."
tar -cz --no-xattrs --no-mac-metadata -f ottoapp.tgz assets components ottoapp.exe || exit 2

# push the file to our production server
echo " info: pushing tarball..."
scp ottoapp.tgz mdhender@tribenet:/var/www/ottomap.mdhenderson.com/ottoapp.tgz || exit 2

# execute the installation script
echo " info: executing the installation script..."
ssh mdhender@tribenet /home/mdhender/bin/install.sh || {
  echo "error: installation script failed"
  exit 2
}

# next
echo " info: if this succeeded, you should restart the services"
echo "       ssh tribenet systemctl restart ottoapp.service"
echo "       ssh tribenet systemctl status  ottoapp.service"
echo "       ssh tribenet journalctl -f -u  ottoapp.service"

exit 0
