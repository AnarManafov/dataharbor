#!/bin/bash

# How to use:
# * to build both backend and frontend packages: ./packaging/build_rpm.sh
#   or ./packaging/build_rpm.sh -b -f
# * to build the backend package only: ./packaging/build_rpm.sh -b
# * to build the frontend package only: ./packaging/build_rpm.sh -f

# Variables
VERSION_BACKEND="0.6.0"
APP_NAME_BACKEND="data-lake-ui-backend"
SOURCE_DIR_BACKEND="app"
SPEC_FILE_BACKEND="packaging/${APP_NAME_BACKEND}.spec"

VERSION_FRONTEND="0.6.0"
APP_NAME_FRONTEND="data-lake-ui-frontend"
SOURCE_DIR_FRONTEND="web"
SPEC_FILE_FRONTEND="packaging/${APP_NAME_FRONTEND}.spec"

# Default values
BUILD_BACKEND=false
BUILD_FRONTEND=false

# Parse arguments
while getopts "bf" opt; do
  case ${opt} in
    b )
      BUILD_BACKEND=true
      ;;
    f )
      BUILD_FRONTEND=true
      ;;
    \? )
      echo "Usage: cmd [-b] [-f]"
      exit 1
      ;;
  esac
done

# If no flags are provided, build both
if [ "$BUILD_BACKEND" = false ] && [ "$BUILD_FRONTEND" = false ]; then
  BUILD_BACKEND=true
  BUILD_FRONTEND=true
fi

# Build backend package if requested
if [ "$BUILD_BACKEND" = true ]; then
  ./packaging/build_rpm_base.sh ${APP_NAME_BACKEND} ${SOURCE_DIR_BACKEND} ${SPEC_FILE_BACKEND} ${VERSION_BACKEND}
fi

# Build frontend package if requested
if [ "$BUILD_FRONTEND" = true ]; then
  ./packaging/build_rpm_base.sh ${APP_NAME_FRONTEND} ${SOURCE_DIR_FRONTEND} ${SPEC_FILE_FRONTEND} ${VERSION_FRONTEND}
fi