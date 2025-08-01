#!/bin/bash

# Configuration
URL="https://www.ideamans.com/"
PORT=8080
INVENTORY=./inventory
LIGHTHOUSE=./lighthouse

# Setup function
setup() {
    echo "Setting up environment..."
    mkdir -p $LIGHTHOUSE
    make clean && make build
    rm -rf $INVENTORY
    yarn
    echo "Setup completed."
}

# Baseline test function
baseline() {
    echo "Running baseline Lighthouse test..."
    yarn lighthouse $URL \
      --only-categories=performance \
      --output=html \
      --output-path=$LIGHTHOUSE/lighthouse-baseline.html \
      --view
    echo "Baseline test completed."
}

# Recording test function
recording() {
    echo "Running recording mode test..."
    
    # Start recording proxy
    ./http-playback-proxy recording --port $PORT --no-beautify $URL &
    RECORD_PID=$!
    sleep 2
    
    # Run Lighthouse through proxy
    yarn lighthouse $URL \
      --chrome-flags="--proxy-server=127.0.0.1:$PORT --ignore-certificate-errors --disable-web-security" \
      --only-categories=performance \
      --output=html \
      --output-path=$LIGHTHOUSE/lighthouse-recording.html \
      --view
    
    # Stop proxy
    kill $RECORD_PID 2>/dev/null || true
    wait $RECORD_PID 2>/dev/null || true
    sleep 2
    
    # Show inventory stats
    if [ -f "$INVENTORY/inventory.json" ]; then
        resource_count=$(jq '.resources | length' "$INVENTORY/inventory.json" 2>/dev/null || echo "unknown")
        domain_count=$(jq '.domains | length' "$INVENTORY/inventory.json" 2>/dev/null || echo "unknown")
        echo "Recorded $resource_count resources and $domain_count domains"
    fi
    
    echo "Recording test completed."
}

# Playback test function
playback() {
    echo "Running playback mode test..."
    
    # Check inventory exists
    if [ ! -f "$INVENTORY/inventory.json" ]; then
        echo "ERROR: No inventory found. Run recording first."
        return 1
    fi
    
    # Start playback proxy
    ./http-playback-proxy playback --port $PORT &
    PLAYBACK_PID=$!
    sleep 2
    
    # Run Lighthouse through proxy
    yarn lighthouse $URL \
      --chrome-flags="--proxy-server=127.0.0.1:$PORT --ignore-certificate-errors --disable-web-security" \
      --only-categories=performance \
      --output=html \
      --output-path=$LIGHTHOUSE/lighthouse-playback.html \
      --view
    
    # Stop proxy
    kill $PLAYBACK_PID 2>/dev/null || true
    wait $PLAYBACK_PID 2>/dev/null || true
    
    echo "Playback test completed."
}

# Cleanup function
cleanup() {
    pkill -f "http-playback-proxy" 2>/dev/null || true
}

# Set trap for cleanup
trap cleanup EXIT

# Main execution
setup
baseline    # Comment out to skip baseline
recording   # Comment out to skip recording  
playback    # Comment out to skip playback

echo "All tests completed!"
echo "Reports saved in: $LIGHTHOUSE/"
echo "Inventory saved in: $INVENTORY/"