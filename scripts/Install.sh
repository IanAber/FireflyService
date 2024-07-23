#!/bin/bash
if [ "$USER" != "root" ]
then
    echo "Please run this as root or with sudo"
    exit 2
fi
if test -f "/etc/systemd/system/FireflyIO.service"; then
  systemctl stop FireflyIO
  systemctl disable FireflyIO
  rm /etc/systemd/system/FireflyIO.service
  rm /usr/bin/FireflyIO
  cp FireflyService.service /etc/systemd/system
  systemctl enable FireflyService
else
  systemctl stop FireflyService
fi
cp dist/amd64/FireflyService /usr/bin

chmod +x /usr/bin/FireflyService

if ! test -d "/etc/FireflyService"; then
  mkdir /etc/FireflyService
fi
if ! test -d "/etc/FireflyService/web"; then
  mkdir /etc/FireflyService/web
fi
if ! test -f "/etc/FireflyService.json"; then
  cp /etc/FireFlyIO.json /etc/FireflyService.json
fi
cp -r web/* /etc/FireflyService/web
systemctl start FireflyService

echo "Done"
