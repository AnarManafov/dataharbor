#!/bin/bash

# Variables
APP_NAME="data-lake-ui-backend"
VERSION="0.5.0"
RELEASE="1"
SOURCE_DIR="app"
BUILD_DIR="$HOME/rpmbuild"
SPEC_FILE="packaging/${APP_NAME}.spec"

# Ensure rpmbuild directories exist
mkdir -p ${BUILD_DIR}/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Step 1: Cross-compile the Go application for Linux
echo "Building the Go application..."
cd ${SOURCE_DIR}
GOOS=linux GOARCH=amd64 go build -o ${APP_NAME}
cd ..

# Step 2: Copy the binary to the SOURCES directory
echo "Copying binary to SOURCES directory..."
cp ${SOURCE_DIR}/${APP_NAME} ${BUILD_DIR}/SOURCES/

# Step 3: Create a source tarball
echo "Creating source tarball..."
tar czvf ${BUILD_DIR}/SOURCES/${APP_NAME}-${VERSION}.tar.gz -C ${SOURCE_DIR} ${APP_NAME}

# Step 4: Copy the spec file to the SPECS directory
echo "Copying spec file..."
cp ${SPEC_FILE} ${BUILD_DIR}/SPECS/

# Step 5: Build the RPM package
echo "Building the RPM package..."
rpmbuild -ba ${BUILD_DIR}/SPECS/${APP_NAME}.spec

# Check if the RPM was created successfully
if [ $? -eq 0 ]; then
    echo "RPM package created successfully."
else
    echo "Failed to create RPM package."
    exit 1
fi
