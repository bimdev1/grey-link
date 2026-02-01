#!/bin/bash
set -e
export ANDROID_HOME=/home/tim/Android/Sdk

echo "Checking directories..."
ls -F
ls -F android/

echo "Making gradlew executable..."
chmod +x android/gradlew

echo "Starting build..."
cd android
./gradlew assembleDebug --info
