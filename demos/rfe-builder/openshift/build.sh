#!/bin/bash

# Build Script for RHOAI AI Feature Sizing Platform
# Usage: IMAGE_FULL_NAME=quay.io/my/image:latest ./build.sh
# Or use default: ./build.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEFAULT_IMAGE="quay.io/gkrumbach07/llama-index-demo/rhoai-ai-feature-sizing:latest"

# Use IMAGE_FULL_NAME environment variable or default
IMAGE_FULL_NAME="${IMAGE_FULL_NAME:-$DEFAULT_IMAGE}"

echo -e "${BLUE}🔨 RHOAI AI Feature Sizing - Container Build${NC}"
echo -e "${BLUE}===========================================${NC}"
echo -e "Image: ${GREEN}${IMAGE_FULL_NAME}${NC}"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect container runtime
CONTAINER_RUNTIME=""
if command_exists podman; then
    CONTAINER_RUNTIME="podman"
    echo -e "${GREEN}✅ Using Podman as container runtime${NC}"
elif command_exists docker; then
    CONTAINER_RUNTIME="docker"
    echo -e "${GREEN}✅ Using Docker as container runtime${NC}"
else
    echo -e "${RED}❌ Neither Podman nor Docker found. Please install one of them first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"
echo ""

# Build container image
echo -e "${YELLOW}🔨 Building container image...${NC}"
${CONTAINER_RUNTIME} build -t "${IMAGE_FULL_NAME}" ..

echo -e "${YELLOW}📤 Pushing container image to registry...${NC}"
${CONTAINER_RUNTIME} push "${IMAGE_FULL_NAME}"
echo -e "${GREEN}✅ Image pushed successfully${NC}"
echo ""

echo -e "${GREEN}🎉 Build completed successfully!${NC}"
echo -e "${GREEN}==============================${NC}"
echo -e "Image: ${BLUE}${IMAGE_FULL_NAME}${NC}"
echo ""
echo -e "${YELLOW}📝 Usage examples:${NC}"
echo -e "1. Build with custom image name:"
echo -e "   ${BLUE}IMAGE_FULL_NAME=quay.io/my/image:latest ./build.sh${NC}"
echo -e "2. Build with defaults:"
echo -e "   ${BLUE}./build.sh${NC}"
echo ""
echo -e "${YELLOW}📝 Next steps:${NC}"
echo -e "Deploy the image with:"
echo -e "   ${BLUE}./openshift/deploy.sh${NC}"
echo ""
