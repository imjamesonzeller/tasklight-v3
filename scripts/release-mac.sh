#!/usr/bin/env bash
set -euo pipefail

APP_NAME="Tasklight"
BUILD_DIR="bin"
APP="${BUILD_DIR}/${APP_NAME}.app"
DMG="${BUILD_DIR}/${APP_NAME}.dmg"
ZIP="${BUILD_DIR}/${APP_NAME}.zip"
VOL="${APP_NAME}"
STAGE="${BUILD_DIR}/dist-dmg"

VER="${1:-v0.1.0}"         # usage: ./scripts/release-mac.sh v0.1.0
TITLE="${APP_NAME} ${VER}"
NOTES="${2:-Unsigned preview build. First run: Right-click → Open, or run:
xattr -dr com.apple.quarantine /Applications/${APP_NAME}.app }"

echo "🔨 Packaging ${APP_NAME} for macOS…"
wails3 package

# --- sanity on bundle ---
echo "🧼 Sanitizing app bundle…"
chmod +x "${APP}/Contents/MacOS/${APP_NAME}" || true
# make sure icon exists (won't fail build if missing)
if [[ ! -f "${APP}/Contents/Resources/icons.icns" ]]; then
  echo "⚠️  ${APP}/Contents/Resources/icons.icns not found (ensure CFBundleIconFile matches)"
fi
# strip ALL extended attributes so we don't bake in quarantine/ACLs
xattr -cr "${APP}"
# tidy
find "${APP}" -name ".DS_Store" -delete || true

# --- DMG (read-only, finalized) ---
echo "🧺 Staging DMG payload…"
rm -rf "${STAGE}" && mkdir -p "${STAGE}"
# use ditto (better metadata than cp -R)
ditto "${APP}" "${STAGE}/${APP_NAME}.app"
ln -s /Applications "${STAGE}/Applications"

echo "💽 Creating read-only DMG…"
hdiutil create -volname "${VOL}" \
  -srcfolder "${STAGE}" \
  -fs HFS+ -format UDZO "${DMG}"

echo "🔎 Verifying DMG…"
hdiutil verify "${DMG}"

# --- ZIP (alternative download path) ---
echo "🗜️  Creating ZIP…"
ditto -c -k --keepParent "${APP}" "${ZIP}"

# --- checksums ---
echo "🔐 Checksums…"
shasum -a 256 "${DMG}" > "${DMG}.sha256"
shasum -a 256 "${ZIP}" > "${ZIP}.sha256"

# --- GitHub release ---
echo "🚀 Publishing GitHub release ${VER}…"
# ensure we're at repo root when invoking gh
if [[ "$(pwd)" != */"${BUILD_DIR}" ]]; then
  : # already at root
else
  cd ..
fi

# create tag if it doesn't exist
if ! git rev-parse "${VER}" >/dev/null 2>&1; then
  git tag "${VER}"
  git push origin "${VER}"
fi

gh release create "${VER}" \
  "${BUILD_DIR}/${APP_NAME}.dmg" \
  "${BUILD_DIR}/${APP_NAME}.dmg.sha256" \
  "${BUILD_DIR}/${APP_NAME}.zip" \
  "${BUILD_DIR}/${APP_NAME}.zip.sha256" \
  --title "${TITLE}" \
  --notes "${NOTES}"

echo "✅ Release ${VER} created with DMG + ZIP + checksums."