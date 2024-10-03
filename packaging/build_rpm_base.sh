#!/bin/bash

# Parameters
APP_NAME=$1
SOURCE_DIR=$2
SPEC_FILE=$3
VERSION=$4
BUILD_DIR="$HOME/rpmbuild"

# Ensure rpmbuild directories exist
mkdir -p ${BUILD_DIR}/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Step 1: Build the application
echo "Building the application..."
cd ${SOURCE_DIR}
if [ -f "package.json" ]; then
    npm install
    npm run build
elif [ -f "go.mod" ]; then
    GOOS=linux GOARCH=amd64 go build -o ${APP_NAME}
else
    echo "Unknown project type. Exiting."
    exit 1
fi
cd ..

# Step 2: Copy the binaries to the SOURCES directory
echo "Copying binaries to SOURCES directory..."
if [ -d "${SOURCE_DIR}/dist" ]; then
    cp -r ${SOURCE_DIR}/dist ${BUILD_DIR}/SOURCES/${APP_NAME}-${VERSION}
else
    cp ${SOURCE_DIR}/${APP_NAME} ${BUILD_DIR}/SOURCES/
fi

# Step 3: Create source tarballs
echo "Creating source tarballs..."
if [ -d "${SOURCE_DIR}/dist" ]; then
    tar czvf ${BUILD_DIR}/SOURCES/${APP_NAME}-${VERSION}.tar.gz -C ${BUILD_DIR}/SOURCES ${APP_NAME}-${VERSION}
else
    tar czvf ${BUILD_DIR}/SOURCES/${APP_NAME}-${VERSION}.tar.gz -C ${SOURCE_DIR} ${APP_NAME}
fi

# Step 4: Copy the spec file to the SPECS directory
echo "Copying spec file..."
cp ${SPEC_FILE} ${BUILD_DIR}/SPECS/

# Step 5: Build the RPM package
echo "Building the RPM package..."
rpmbuild -ba ${BUILD_DIR}/SPECS/$(basename ${SPEC_FILE})

# Check if the RPM was created successfully
if [ $? -eq 0 ]; then
    echo "RPM package created successfully."
else
    echo "Failed to create RPM package."
    exit 1
fi
