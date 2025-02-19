#!/bin/bash

set -e

source .env 

rm -f GooeyLife.aab
fyne release --key-store ${ANDROIDKEYSTORE} --key-name "${ANDROIDKEYNAME}" --key-pass "${ANDROIDKEYPASS}" --target android

rm -f GooeyLife.ipa
fyne release --target ios --certificate "${IOSDISTROCERT}" --profile "${IOSDISTROPROF}"
