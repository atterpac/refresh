#!/bin/bash

# Function to display script usage
usage() {
    echo "Usage: $0 <count>"
    exit 1
}

# Check if count is provided
if [ -z "$1" ]; then
    echo "Count not provided."
    usage
fi

# Parse count from the command line
count="$1"

# Main loop to log count every second
for ((i=1; i<=$count; i++)); do
    sleep 1
done


