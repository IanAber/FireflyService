if [ "$USER" != "root" ]
then
    echo "Please run this as root or with sudo"
    exit 2
fi
if ! test -d "/FireflyService"; then
  mkdir /FireflyService
fi
if ! test -d "/FireflyService/web"; then
  mkdir /FireflyService/web
fi
rm -r /FireflyService/web/*
cp -r web/* /FireflyService/web
