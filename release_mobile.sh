#!/bin/bash

set -e

fyne release --keyStore ~/.keystore --keyName apksigning --target android

fyne release --target ios --certificate "iPhone Developer: Mitchell Ross Patenaude (VQWB76ZRJG)" --profile GooeyLifeDevProfile
