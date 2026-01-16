#!/bin/bash

# Configuration
METADATA_URL="https://us-central1-apt.pkg.dev/projects/antigravity-auto-updater-dev/dists/antigravity-debian/main/binary-amd64/Packages"
BASE_URL="https://us-central1-apt.pkg.dev/projects/antigravity-auto-updater-dev/"

echo "Fetching package list..."

# 1. Fetch metadata, filter for Filename, and extract the path
# 2. Sort by Version (-V) to ensure 1.14 > 1.2
# 3. Take the last line (highest version)
LATEST_PATH=$(curl -s "$METADATA_URL" | grep "Filename:" | awk '{print $2}' | sort -V | tail -n 1)

# Check if we actually found a file
if [ -z "$LATEST_PATH" ]; then
    echo "Error: Could not find any filenames. Check the URL or network connection."
    exit 1
fi

echo "Found latest version path: $LATEST_PATH"

# Construct the full URL
FULL_URL="${BASE_URL}${LATEST_PATH}"

# Extract just the filename (e.g., antigravity_1.0.0_amd64.deb) from the long path
LOCAL_FILENAME=$(basename "$LATEST_PATH")

echo "Downloading $LOCAL_FILENAME..."

# Download the file
# -O forces the output filename to ensure it matches what we pass to dpkg
wget -q --show-progress -O "$LOCAL_FILENAME" "$FULL_URL"

# Check if download succeeded
if [ $? -ne 0 ]; then
    echo "Error: Download failed."
    exit 1
fi

echo "Installing $LOCAL_FILENAME..."

# Install the package
# This will prompt for your password if you didn't run the script as root
sudo dpkg --install "$LOCAL_FILENAME"
