if [ "$USER" != "root" ]
then
    echo "Please run this as root or with sudo"
    exit 2
fi
if ! test -d "/etc/FireflyService"; then
  mkdir /etc/FireflyService
fi
if ! test -d "/etc/FireflyService/web"; then
  mkdir /etc/FireflyService/web
fi
rm -r /etc/FireflyService/web/*
cp -r web/* /etc/FireflyService/web
