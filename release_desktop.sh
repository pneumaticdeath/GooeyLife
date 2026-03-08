#!/bin/bash

# set -e

source .env 

# rm -rf GooeyLife.app
fyne release --target darwin --certificate "${MACOSDEVCERT}" --profile "${MACOSDEVPROF}" --category games
