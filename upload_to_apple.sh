#!/bin/bash

source .env

xcrun altool --upload-app -f "${IOSUPLOADPKG}" -t ios -asc-provider-id "${IOSASCPROV}" -u "${IOSUPLOADUSER}" -p "${IOSUPLOADPASS}"
