#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "${GREEN}Initializing A Eso Voy development environment...${NC}"

# 1. Setup Environment Variables
if [ ! -f .env ]; then
    echo "Creating .env file from defaults..."
    cat > .env <<EOL
DB_NAME=aesovoy
DB_USER=postgres
DB_PASSWORD=password
DB_HOST=localhost
DB_PORT=5432
LOG_FILE=server.log

# Test DB settings
DB_TEST_NAME=aesovoy_test
DB_TEST_USER=postgres
DB_TEST_PASSWORD=password
DB_TEST_HOST=localhost
DB_TEST_PORT=5433

# Mail settings (Placeholder)
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USERNAME=user
SMTP_PASSWORD=pass
SMTP_FROM=no-reply@aesovoy.com
EOL
else
    echo ".env file already exists. Using existing configuration."
fi

# 2. Start Database Container
echo -e "${GREEN}Starting Database container...${NC}"
# We only need the 'db' service for local Go development, 
# but we might want 'test_db' if we run tests later.
docker compose up -d db

# 3. Wait for Database to be ready
echo "Waiting for database to be ready..."
until docker compose exec -T db pg_isready -U postgres; do
  echo "Database is unavailable - sleeping"
  sleep 2
done
echo -e "${GREEN}Database is up and running!${NC}"

# 4. Run the Application
echo -e "${GREEN}Starting Go Application...${NC}"
echo "Server will listen on port 8080"
echo "Press Ctrl+C to stop"

go run .
