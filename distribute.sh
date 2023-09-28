#!/bin/zsh
rm -r dist/*
zip -r dist/web web
cp bin/amd64/FireflyIO dist
cp UpdateWeb.sh dist
cp Install.sh dist
