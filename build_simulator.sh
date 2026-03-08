#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SIM_BUILD_DIR="/tmp/GooeyLifeSim"
SIM_APP="$SIM_BUILD_DIR/GooeyLife.app"
DEVICE_APP="$SCRIPT_DIR/GooeyLife.app"
METADATA_FILE="$SCRIPT_DIR/fyne_metadata_init.go"

# Parse metadata from FyneApp.toml
APP_ID=$(awk -F'"' '/^ID/{print $2}' "$SCRIPT_DIR/FyneApp.toml")
APP_NAME=$(awk -F'"' '/^Name/{print $2}' "$SCRIPT_DIR/FyneApp.toml")
APP_VERSION=$(awk -F'"' '/^Version/{print $2}' "$SCRIPT_DIR/FyneApp.toml")
APP_BUILD=$(awk '/^Build/{print $3}' "$SCRIPT_DIR/FyneApp.toml")

mkdir -p "$SIM_APP"

# Step 1: Generate fyne_metadata_init.go with embedded icon
cat > "$METADATA_FILE" <<EOF
package main

import (
	_ "embed"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

//go:embed Icon.png
var bundledIconData []byte

var bundledIcon = &fyne.StaticResource{
	StaticName:    "Icon.png",
	StaticContent: bundledIconData,
}

func init() {
	app.SetMetadata(fyne.AppMetadata{
		ID:      "$APP_ID",
		Name:    "$APP_NAME",
		Version: "$APP_VERSION",
		Build:   $APP_BUILD,
		Icon:    bundledIcon,
	})
}
EOF

cleanup() {
    rm -f "$METADATA_FILE"
}
trap cleanup EXIT

# Step 2: Build simulator binary with iphonesimulator SDK
echo "Building for iOS Simulator (metadata: $APP_NAME $APP_VERSION/$APP_BUILD)..."
SDK=$(xcrun --sdk iphonesimulator --show-sdk-path)
CC=$(xcrun --sdk iphonesimulator --find clang)
FLAGS="-arch arm64 -isysroot $SDK -miphonesimulator-version-min=16.0"

GOOS=ios GOARCH=arm64 CGO_ENABLED=1 \
    CC="$CC" \
    CGO_CFLAGS="$FLAGS" \
    CGO_LDFLAGS="$FLAGS" \
    go build -tags ios -o "$SIM_APP/main" "$SCRIPT_DIR" 2>&1 | grep -v "deprecated" | grep -v "note:" || true

if [ ! -f "$SIM_APP/main" ]; then
    echo "Build failed — binary not produced." >&2
    exit 1
fi

# Step 3: Build device .app for resources if stale or missing
if [ ! -f "$DEVICE_APP/Info.plist" ] || \
   [ "$SCRIPT_DIR/FyneApp.toml" -nt "$DEVICE_APP/Info.plist" ] || \
   [ "$SCRIPT_DIR/Icon.png" -nt "$DEVICE_APP/Info.plist" ]; then
    echo "Rebuilding .app resources (FyneApp.toml or Icon.png changed)..."
    fyne package --target ios
fi

# Step 4: Assemble simulator .app
cp "$DEVICE_APP/Info.plist" "$SIM_APP/"
cp "$DEVICE_APP/PkgInfo"    "$SIM_APP/"
cp "$DEVICE_APP/Assets.car" "$SIM_APP/"
[ -d "$DEVICE_APP/assets" ] && cp -r "$DEVICE_APP/assets" "$SIM_APP/" || true

echo "Done: $SIM_APP"
echo ""
echo "Install:  xcrun simctl install booted $SIM_APP"
echo "Launch:   xcrun simctl launch booted $APP_ID"
