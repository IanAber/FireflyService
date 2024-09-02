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

if ! test -d "/FireflyService"; then
  mkdir /FireflyService
fi
if ! test -d "/FireflyService/web"; then
  mkdir /FireflyService/web
fi
if ! test -f "/etc/FireflyService.json"; then
  cp /etc/FireFlyIO.json /etc/FireflyService.json
fi
cp -r web/* /etc/FireflyService/web
systemctl start FireflyService

echo "Done"
