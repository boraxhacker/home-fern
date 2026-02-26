#!/bin/bash

# Source the aws-fern.sh script to set up environment variables
source ./aws-fern.sh

# Create a hosted zone
echo "Creating hosted zone example.com..."
aws route53 create-hosted-zone --name example.com --caller-reference $(date +%s)

# List hosted zones by name
echo "Listing hosted zones by name (DNSName=example.com)..."
aws route53 list-hosted-zones-by-name --dns-name example.com

# List hosted zones by name with max items
echo "Listing hosted zones by name (DNSName=example.com, MaxItems=1)..."
aws route53 list-hosted-zones-by-name --dns-name example.com --max-items 1

DELETEME=$(aws route53 list-hosted-zones-by-name --dns-name example.com --max-items 1 \
  | jq -r '.HostedZones[0].Id')


# List hosted zones
echo "Listing hosted zones..."
aws route53 list-hosted-zones

# Clean up, delete ${DELETEME}
echo "Deleting hosted zone ${DELETEME}..."
aws route53 delete-hosted-zone --id ${DELETEME}
