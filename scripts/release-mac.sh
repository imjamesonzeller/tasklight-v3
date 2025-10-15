#!/usr/bin/env bash
set -euo pipefail

APP_NAME="Tasklight"
BUILD_DIR="bin"
APP_PATH="${APP_NAME}.app"
VER="${1:-v0.1.0}"

echo "Building ${APP_NAME}..."
wails3 build -platform darwin/universal -clean

echo "Entering ${BUILD_DIR}..."
cd "${BUILD_DIR}"

echo "Packaging DMG..."
STAGE="dist-dmg"
rm -rf "$STAGE" && mkdir "$STAGE"
cp -R "$APP_PATH" "$STAGE/"
ln -s /Applications "$STAGE/Applications"
hdiutil create -volname "$APP_NAME" -srcfolder "$STAGE" -fs HFS+ -format UDZO "${APP_NAME}.dmg"

echo "Packaging ZIP..."
ditto -c -k --keepParent "$APP_PATH" "${APP_NAME}.zip"

echo "Checksums..."
shasum -a 256 "${APP_NAME}.dmg" > "${APP_NAME}.dmg.sha256"
shasum -a 256 "${APP_NAME}.zip" > "${APP_NAME}.zip.sha256"

echo "Creating GitHub Release ${VER}..."
cd ../../
gh release create "${VER}" \
  "${BUILD_DIR}/${APP_NAME}.dmg" "${BUILD_DIR}/${APP_NAME}.dmg.sha256" \
  "${BUILD_DIR}/${APP_NAME}.zip" "${BUILD_DIR}/${APP_NAME}.zip.sha256" \
  --title "${APP_NAME} ${VER}" \
  --notes "Unsigned preview build. First run: Right-click → Open."

echo "✅ Release ${VER} created successfully!"