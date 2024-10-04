#!/bin/bash

SPEC_FILE=$1
RELEASE_NOTES_FILE=$2

# Extract all release notes
ALL_NOTES=$(awk '/## \[/{flag=1} /## \[/{if (flag) print ""; flag=1} flag' $RELEASE_NOTES_FILE)

# Get the current date
CURRENT_DATE=$(date +"%a %b %d %Y")

# Get the packager's name and email
PACKAGER_NAME="Anar Manafov"
PACKAGER_EMAIL="Anar.Manafov@gmail.com"

# Initialize the changelog entry
CHANGELOG_ENTRY=""

# Function to convert date to the correct format
convert_date() {
    local input_date=$1
    if [[ $OSTYPE == 'darwin'* ]]; then
        # macOS date format conversion
        date -j -f "%Y-%m-%d" "$input_date" +"%a %b %d %Y"
    else
        # GNU date format conversion
        date -d "$input_date" +"%a %b %d %Y"
    fi
}

# Process each version section
while IFS= read -r line; do
    if [[ $line =~ ^##\ \[(.*)\]\ -\ (.*)$ ]]; then
        VERSION=${BASH_REMATCH[1]}
        DATE=${BASH_REMATCH[2]}
        if [[ $DATE == "NOT YET RELEASED" ]]; then
            DATE=$CURRENT_DATE
        else
            # Convert date to the correct format
            DATE=$(convert_date "$DATE")
        fi
        CHANGELOG_ENTRY+="* $DATE $PACKAGER_NAME <$PACKAGER_EMAIL> - $VERSION-1\n"
    else
        CHANGELOG_ENTRY+="$line\n"
    fi
done <<< "$ALL_NOTES"

# Create a temporary file for the changelog entry
TEMP_CHANGELOG=$(mktemp)
echo -e "$CHANGELOG_ENTRY" > $TEMP_CHANGELOG

# Append the changelog entry to the spec file
awk '/^%changelog/{print;system("cat '$TEMP_CHANGELOG'");next}1' $SPEC_FILE > temp && mv temp $SPEC_FILE

# Clean up the temporary file
rm $TEMP_CHANGELOG
