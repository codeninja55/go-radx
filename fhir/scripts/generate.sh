#!/bin/bash

# Stop on error
set -e

# Variables
FHIR_VERSION="R4"
DOWNLOAD_URL="https://hl7.org/fhir/R4/definitions.json.zip"
ZIP_FILE="definitions.json.zip"
SCHEMA_DIR="fhir_schemas"
OUTPUT_DIR="fhir/r4/resources"
PACKAGE_NAME="resources"

# Clean up previous run
rm -rf $ZIP_FILE $SCHEMA_DIR

# Create output directory
mkdir -p $OUTPUT_DIR

# Download FHIR definitions
echo "Downloading FHIR $FHIR_VERSION definitions..."
curl -L -o $ZIP_FILE $DOWNLOAD_URL

# Unzip definitions
echo "Unzipping definitions..."
unzip -o $ZIP_FILE -d $SCHEMA_DIR
rm $ZIP_FILE

# Install go-jsonschema generator
echo "Installing go-jsonschema..."
go install github.com/atombender/go-jsonschema@latest

# Generate Go structs
echo "Generating Go structs..."
# We only generate a subset of resources for now as a proof of concept
RESOURCES=("Patient" "Observation" "Bundle" "Organization" "Practitioner")

for resource in "${RESOURCES[@]}"; do
    schema_file="$SCHEMA_DIR/${resource}.schema.json"
    output_file="$OUTPUT_DIR/$(echo $resource | tr '[:upper:]' '[:lower:]').go"
    echo "Generating $output_file from $schema_file"
    go-jsonschema -p $PACKAGE_NAME -o "$output_file" "$schema_file"
done

# Clean up schema directory
# rm -rf $SCHEMA_DIR

echo "FHIR $FHIR_VERSION Go structs generated successfully in $OUTPUT_DIR"
