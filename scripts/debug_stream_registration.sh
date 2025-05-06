#!/bin/bash

# Stream registration debugging script for StreamHive
# This script checks the stream registration status across services

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Default configuration
STREAM_ID=""
SIGNALING_HOST="localhost:8086"
WEBRTC_HOST="localhost:8085"
RTMP_HOST="localhost:8082"
VERBOSE=false

# Print usage information
function print_usage {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -s, --stream-id STREAM_ID   Stream ID to check (required)"
    echo "  --signaling-host HOST       WebSocket Signaling host (default: localhost:8006)"
    echo "  --webrtc-host HOST          WebRTC host (default: localhost:8005)"
    echo "  --rtmp-host HOST            RTMP Ingestor host (default: localhost:8002)"
    echo "  -v, --verbose               Enable verbose output"
    echo "  -h, --help                  Show this help message"
    exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -s|--stream-id)
            STREAM_ID="$2"
            shift
            shift
            ;;
        --signaling-host)
            SIGNALING_HOST="$2"
            shift
            shift
            ;;
        --webrtc-host)
            WEBRTC_HOST="$2"
            shift
            shift
            ;;
        --rtmp-host)
            RTMP_HOST="$2"
            shift
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            print_usage
            ;;
        *)
            echo "Unknown option: $1"
            print_usage
            ;;
    esac
done

# Check if stream ID is provided
if [ -z "$STREAM_ID" ]; then
    echo -e "${RED}Error: Stream ID is required${NC}"
    print_usage
fi

echo -e "${BLUE}=== StreamHive Stream Registration Debug ===${NC}"
echo -e "${BLUE}Stream ID:${NC} $STREAM_ID"
echo -e "${BLUE}Signaling Host:${NC} $SIGNALING_HOST"
echo -e "${BLUE}WebRTC Host:${NC} $WEBRTC_HOST"
echo -e "${BLUE}RTMP Host:${NC} $RTMP_HOST"
echo ""

# Function to check service health
function check_service_health {
    local service_name=$1
    local host=$2
    local endpoint=$3

    echo -e "${BLUE}Checking $service_name health...${NC}"

    local health_result
    health_result=$(curl -s "http://$host$endpoint")
    local exit_code=$?

    if [ $exit_code -ne 0 ]; then
        echo -e "${RED}Error: Could not connect to $service_name at http://$host$endpoint${NC}"
        return 1
    fi

    if [ "$VERBOSE" = true ]; then
        echo -e "${PURPLE}Response:${NC} $health_result"
    fi

    if [[ $health_result == *"\"status\":\"up\""* ]]; then
        echo -e "${GREEN}$service_name is healthy${NC}"
        return 0
    else
        echo -e "${YELLOW}$service_name might have issues${NC}"
        return 0
    fi
}

# Function to check stream status in signaling service
function check_signaling_stream {
    echo -e "${BLUE}Checking stream in Signaling service...${NC}"

    local stream_info
    stream_info=$(curl -s "http://$SIGNALING_HOST/stream?streamId=$STREAM_ID")
    local exit_code=$?

    if [ $exit_code -ne 0 ]; then
        echo -e "${RED}Error: Could not connect to Signaling service${NC}"
        return 1
    fi

    if [[ $stream_info == *"Stream not found"* ]]; then
        echo -e "${RED}Stream not found in Signaling service${NC}"
        return 1
    fi

    if [ "$VERBOSE" = true ]; then
        echo -e "${PURPLE}Response:${NC} $stream_info"
    fi

    local node_id
    node_id=$(echo $stream_info | grep -o '"node_id":"[^"]*"' | cut -d'"' -f4)

    echo -e "${GREEN}Stream is registered in Signaling service${NC}"
    echo -e "${BLUE}Node ID:${NC} $node_id"
    return 0
}

# Function to check stream status in WebRTC service
function check_webrtc_stream {
    echo -e "${BLUE}Checking stream in WebRTC service...${NC}"

    local stream_health
    stream_health=$(curl -s "http://$WEBRTC_HOST/stream/health?stream_id=$STREAM_ID")
    local exit_code=$?

    if [ $exit_code -ne 0 ]; then
        echo -e "${RED}Error: Could not connect to WebRTC service${NC}"
        return 1
    fi

    if [[ $stream_health == *"Stream not found"* ]]; then
        echo -e "${RED}Stream not found in WebRTC service${NC}"
        return 1
    fi

    if [ "$VERBOSE" = true ]; then
        echo -e "${PURPLE}Response:${NC} $stream_health"
    fi

    local active_connections
    active_connections=$(echo $stream_health | grep -o '"active_connections":[0-9]*' | cut -d':' -f2)

    local video_queue_depth
    video_queue_depth=$(echo $stream_health | grep -o '"video_queue_depth":[0-9]*' | cut -d':' -f2)

    local audio_queue_depth
    audio_queue_depth=$(echo $stream_health | grep -o '"audio_queue_depth":[0-9]*' | cut -d':' -f2)

    echo -e "${GREEN}Stream is active in WebRTC service${NC}"
    echo -e "${BLUE}Active connections:${NC} $active_connections"
    echo -e "${BLUE}Video queue depth:${NC} $video_queue_depth"
    echo -e "${BLUE}Audio queue depth:${NC} $audio_queue_depth"
    return 0
}

# Function to get all streams from signaling service
function get_all_streams {
    echo -e "${BLUE}Getting all streams from Signaling service...${NC}"

    local streams_info
    streams_info=$(curl -s "http://$SIGNALING_HOST/streams")
    local exit_code=$?

    if [ $exit_code -ne 0 ]; then
        echo -e "${RED}Error: Could not connect to Signaling service${NC}"
        return 1
    fi

    echo $streams_info | jq -r '.[] | "Stream ID: \(.id), Node: \(.node_id), Active: \(.active), Clients: \(.clients)"'

    return 0
}

# Function to try registering a WebSocket client
function test_websocket_connection {
    echo -e "${BLUE}Testing WebSocket connection to stream...${NC}"

    # This is a simple test using websocat if available
    if ! command -v websocat &> /dev/null; then
        echo -e "${YELLOW}Warning: websocat not installed, skipping WebSocket test${NC}"
        echo -e "${YELLOW}Install with: cargo install websocat${NC}"
        return 0
    fi

    local ws_url="ws://$SIGNALING_HOST/ws?streamId=$STREAM_ID&clientId=debug-script"
    echo -e "${BLUE}Connecting to:${NC} $ws_url"

    # Try to connect for 5 seconds
    timeout 5 websocat --no-close $ws_url 2>&1 | grep -v "Connection closed"
    local exit_code=${PIPESTATUS[0]}

    if [ $exit_code -eq 124 ]; then
        # Timeout is actually success here (connection stayed open)
        echo -e "${GREEN}WebSocket connection successful${NC}"
        return 0
    elif [ $exit_code -ne 0 ]; then
        echo -e "${RED}WebSocket connection failed${NC}"
        return 1
    else
        echo -e "${GREEN}WebSocket connection and disconnection successful${NC}"
        return 0
    fi
}

# Execute the checks
check_service_health "Signaling service" "$SIGNALING_HOST" "/health"
echo ""

check_service_health "WebRTC service" "$WEBRTC_HOST" "/health"
echo ""

# Check if the specific stream exists in signaling service
check_signaling_stream
signaling_result=$?
echo ""

# Check if the specific stream exists in WebRTC service
check_webrtc_stream
webrtc_result=$?
echo ""

# If stream not found in signaling but exists in WebRTC
if [ $signaling_result -ne 0 ] && [ $webrtc_result -eq 0 ]; then
    echo -e "${RED}Stream registration issue detected:${NC}"
    echo -e "${YELLOW}Stream exists in WebRTC service but not in Signaling service${NC}"
    echo -e "${YELLOW}This indicates a failed stream registration${NC}"
    echo ""
    echo -e "${BLUE}Possible solutions:${NC}"
    echo "1. Restart the WebRTC service to trigger stream re-registration"
    echo "2. Verify that the WebRTC service can reach the Signaling service"
    echo "3. Check for network issues between services"
    echo "4. Verify that services are using correct addresses"
    echo ""
fi

# If stream not found in WebRTC but exists in signaling
if [ $signaling_result -eq 0 ] && [ $webrtc_result -ne 0 ]; then
    echo -e "${RED}Stream registration issue detected:${NC}"
    echo -e "${YELLOW}Stream exists in Signaling service but not in WebRTC service${NC}"
    echo -e "${YELLOW}This indicates a stale stream registration${NC}"
    echo ""
    echo -e "${BLUE}Possible solutions:${NC}"
    echo "1. Restart the stream"
    echo "2. Check if the WebRTC node is healthy"
    echo "3. Verify that the RTMP ingestor is sending frames to the WebRTC service"
    echo ""
fi

# If stream is registered correctly, test WebSocket connection
if [ $signaling_result -eq 0 ] && [ $webrtc_result -eq 0 ]; then
    echo -e "${GREEN}Stream is properly registered in both services${NC}"
    echo ""
    test_websocket_connection
fi

# List all streams if verbose mode is enabled or if the specific stream was not found
if [ "$VERBOSE" = true ] || [ $signaling_result -ne 0 ]; then
    echo ""
    echo -e "${BLUE}Listing all registered streams:${NC}"
    get_all_streams
fi

echo ""
echo -e "${BLUE}=== Debug Complete ===${NC}"
