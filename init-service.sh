#!/bin/bash

# ==============================================================================
# OpenWiFi Go Microservice Scaffolding Initializer
# This script configures the template for a new service name and port bindings.
# ==============================================================================

set -euo pipefail

# Print usage information
usage() {
    echo "Usage: $0 <service-name> <public-port> <private-port> [target-directory]"
    echo ""
    echo "Arguments:"
    echo "  service-name      Alphanumeric lowercase name (e.g. parental-control)"
    echo "  public-port       Public API port (e.g. 16008)"
    echo "  private-port      Private/Internal API port (e.g. 17008)"
    echo "  target-directory  Optional destination folder. If omitted, runs in-place."
    echo ""
    echo "Example:"
    echo "  $0 mango-mdu-service 16010 17010 ./tmp_test"
    exit 1
}

# Check minimum argument count
if [ "$#" -lt 3 ]; then
    usage
fi

SERVICE_NAME=$1
PUBLIC_PORT=$2
PRIVATE_PORT=$3
TARGET_DIR=${4:-"."}

# Validate service-name matches alpha-numeric and hyphens in a clean DNS-compliant format
if [[ ! "$SERVICE_NAME" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
    echo "Error: Service name '$SERVICE_NAME' must only contain lowercase letters, numbers, and single hyphens. It cannot start or end with a hyphen, or contain consecutive hyphens."
    exit 1
fi

# Validate ports
validate_port() {
    local port=$1
    local port_type=$2
    if [[ ! "$port" =~ ^[0-9]+$ ]]; then
        echo "Error: $port_type port '$port' must be a positive integer."
        exit 1
    fi
    # Check range 1-65535
    if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        echo "Error: $port_type port '$port' must be in the range 1-65535."
        exit 1
    fi
    # Check reserved / privileged ports
    if [ "$port" -lt 1024 ]; then
        echo "Error: $port_type port '$port' is in the reserved system port range (below 1024). Please choose a port between 1024 and 65535."
        exit 1
    fi
}

validate_port "$PUBLIC_PORT" "Public"
validate_port "$PRIVATE_PORT" "Private"

if [ "$PUBLIC_PORT" -eq "$PRIVATE_PORT" ]; then
    echo "Error: Public port ($PUBLIC_PORT) and Private port ($PRIVATE_PORT) cannot be the same."
    exit 1
fi

# Locate the root of the template (where this script resides)
TEMPLATE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Resolve absolute path for Target Directory
mkdir -p "$TARGET_DIR"
TARGET_DIR="$( cd "$TARGET_DIR" && pwd )"

echo "=== Initializing New Go Service ==="
echo "Service Name: $SERVICE_NAME"
echo "Public Port:  $PUBLIC_PORT"
echo "Private Port: $PRIVATE_PORT"
echo "Target Dir:   $TARGET_DIR"
echo "==================================="

# 1. Copy template files to destination if not running in-place
if [ "$TEMPLATE_DIR" != "$TARGET_DIR" ]; then
    echo "Copying template files to target directory..."
    # Copy files excluding .git, binaries, and temporary files
    rsync -a \
      --exclude='.git' \
      --exclude='bin' \
      --exclude='tmp' \
      --exclude='*.log' \
      --exclude='docker-compose*_data' \
      "$TEMPLATE_DIR/" "$TARGET_DIR/"
fi

# 2. Perform string substitutions across all text files safely
echo "Applying string substitutions..."
cd "$TARGET_DIR"

find . -type f \( \
    -name "*.go" -o \
    -name "*.mod" -o \
    -name "Makefile" -o \
    -name "Dockerfile" -o \
    -name "*.env" -o \
    -name "*.yaml" -o \
    -name "*.yml" -o \
    -name "*.md" -o \
    -name "*.sql" \
    \) -not -path "*/.git/*" -not -name "init-service.sh" -print0 | while IFS= read -r -d '' file; do
    # Replace foundation module path with the new module path
    sed -i "s|github.com/routerarchitects/mango-mdu-service|github.com/routerarchitects/$SERVICE_NAME|g" "$file"
    
    # Replace service name placeholder
    sed -i "s|owmdu|$SERVICE_NAME|g" "$file"
    sed -i "s|mango-mdu-service|$SERVICE_NAME|g" "$file"
    
    # Replace ports placeholders
    sed -i "s|16010|$PUBLIC_PORT|g" "$file"
    sed -i "s|17010|$PRIVATE_PORT|g" "$file"
done

# 3. Rename environment files to match the service name
echo "Renaming environment files..."
if [ -f "deployments/docker-compose/docker-compose.env" ]; then
    mv "deployments/docker-compose/docker-compose.env" "deployments/docker-compose/$SERVICE_NAME.env"
fi

# 4. Initialize dynamic dependencies (go mod tidy)
echo "Resolving dependencies (go mod tidy)..."
if command -v go &> /dev/null; then
    go mod tidy
else
    echo "Warning: 'go' binary not found. Please run 'go mod tidy' manually in the target folder."
fi

# Ask user for deployment directory path
echo ""
echo "=== Deployment Integration ==="
read -p "Enter Mango Cloud docker-compose directory path (e.g. /path_to/mango-cloud-deployment/docker-compose): " DEPLOY_DIR

# Expand tilde ~ if present
DEPLOY_DIR="${DEPLOY_DIR/#\~/$HOME}"

# Resolve absolute path for deploy dir
if [ -d "$DEPLOY_DIR" ]; then
    DEPLOY_DIR="$( cd "$DEPLOY_DIR" && pwd )"
fi

RELATIVE_PATH=""
if [ "$TEMPLATE_DIR" != "$TARGET_DIR" ]; then
    RELATIVE_PATH=$(realpath --relative-to="$TEMPLATE_DIR" "$TARGET_DIR")
fi

# 1. Automate copying env file
COPY_SUCCESS=false
if [ -n "$DEPLOY_DIR" ] && [ -d "$DEPLOY_DIR" ]; then
    cp "$TARGET_DIR/deployments/docker-compose/$SERVICE_NAME.env" "$DEPLOY_DIR/$SERVICE_NAME.env"
    COPY_SUCCESS=true
fi

echo ""
echo "Success! Service '$SERVICE_NAME' has been initialized."
echo ""
echo "Next Steps:"

if [ "$COPY_SUCCESS" = true ]; then
    echo "  1. Environment file successfully copied to:"
    echo "     $DEPLOY_DIR/$SERVICE_NAME.env"
else
    echo "  1. Copy the environment file to your docker-compose directory:"
    echo "     $ cp $TARGET_DIR/deployments/docker-compose/$SERVICE_NAME.env /path_to/mango-cloud-deployment/docker-compose/"
fi

if [ -n "$DEPLOY_DIR" ]; then
    echo "  2. Append the following compose service snippet to '$DEPLOY_DIR/docker-compose.yml':"
else
    echo "  2. Append the following compose service snippet to '/path_to/mango-cloud-deployment/docker-compose/docker-compose.yml':"
fi
echo "--------------------------------------------------------------------------------"
sed -n '/services:/,$p' "$TARGET_DIR/deployments/docker-compose/docker-compose.yaml" | tail -n +2
echo "--------------------------------------------------------------------------------"

if [ -n "$DEPLOY_DIR" ]; then
    echo "  3. Go to $DEPLOY_DIR and run docker compose up."
else
    echo "  3. Go to /path_to/mango-cloud-deployment/docker-compose and run docker compose up."
fi

if [ -n "$RELATIVE_PATH" ]; then
    echo "  4. Delete the temporary initialization folder:"
    echo "     $ rm -rf $RELATIVE_PATH"
fi
echo ""
