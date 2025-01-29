#!/bin/bash

set -e

rm -f GooeyLife.aab
fyne release --keyStore ~/.keystore --keyName apksigning --target android

rm -f GooeyLife.ipa
fyne release --target ios --certificate "iPhone Developer: Mitchell Ross Patenaude (VQWB76ZRJG)" --profile GooeyLifeDevProfile
