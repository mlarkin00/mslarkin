#!/bin/bash
set -e

# Configuration
REGION="${REGION:-us-west1}"
CLUSTER_NAME="${CLUSTER_NAME:-onlineboutique-cluster}"
INSTANCE_NAME="${INSTANCE_NAME:-onlineboutique-instance}"
ALLOYDB_CARTS_DATABASE_NAME="carts"
ALLOYDB_CARTS_TABLE_NAME="cart_items"
ALLOYDB_PRODUCTS_DATABASE_NAME="products"
ALLOYDB_PRODUCTS_TABLE_NAME="catalog_items"
MODEL="text-embedding-004"

# Path to scripts
SCRIPTS_DIR="./kustomize/components/shopping-assistant/scripts"

# Check dependencies
if ! command -v psql &> /dev/null; then
    echo "Error: 'psql' is not installed. Please install postgresql-client."
    exit 1
fi

if ! command -v python3 &> /dev/null; then
    echo "Error: 'python3' is not installed."
    exit 1
fi

# 1. Discover AlloyDB Primary IP
echo "Discovering AlloyDB Primary IP..."
# Try to get IP from gcloud if authenticated and api enabled
if command -v gcloud &> /dev/null; then
    ALLOYDB_PRIMARY_IP=$(gcloud alloydb instances list --region="${REGION}" --cluster="${CLUSTER_NAME}" --filter="INSTANCE_TYPE:PRIMARY" --format="value(ipAddress)")
fi

# Fallback or Override
if [ -z "$ALLOYDB_PRIMARY_IP" ]; then
    echo "Could not auto-detect AlloyDB IP. Please enter it manually:"
    read -p "AlloyDB Primary IP: " ALLOYDB_PRIMARY_IP
fi

if [ -z "$ALLOYDB_PRIMARY_IP" ]; then
    echo "Error: No IP provided."
    exit 1
fi

echo "Using AlloyDB IP: ${ALLOYDB_PRIMARY_IP}"
echo "Note: Ensure you are running this script from a machine with VPC access to this IP."

# 2. Database Initialization
export PGPASSWORD=${PGPASSWORD}
if [ -z "$PGPASSWORD" ]; then
    echo "Enter AlloyDB Password (postgres user):"
    read -s PGPASSWORD
    export PGPASSWORD
fi

echo "Creating databases..."
# Ignore errors if DB exists
psql -h "${ALLOYDB_PRIMARY_IP}" -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = '${ALLOYDB_CARTS_DATABASE_NAME}'" | grep -q 1 || \
    psql -h "${ALLOYDB_PRIMARY_IP}" -U postgres -c "CREATE DATABASE ${ALLOYDB_CARTS_DATABASE_NAME}"

psql -h "${ALLOYDB_PRIMARY_IP}" -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = '${ALLOYDB_PRODUCTS_DATABASE_NAME}'" | grep -q 1 || \
    psql -h "${ALLOYDB_PRIMARY_IP}" -U postgres -c "CREATE DATABASE ${ALLOYDB_PRODUCTS_DATABASE_NAME}"

echo "Creating tables..."
# Carts
psql -h "${ALLOYDB_PRIMARY_IP}" -U postgres -d "${ALLOYDB_CARTS_DATABASE_NAME}" -c "
    CREATE TABLE IF NOT EXISTS ${ALLOYDB_CARTS_TABLE_NAME} (userId text, productId text, quantity int, PRIMARY KEY(userId, productId));
    CREATE INDEX IF NOT EXISTS cartItemsByUserId ON ${ALLOYDB_CARTS_TABLE_NAME}(userId);
"

# Products - Extensions
psql -h "${ALLOYDB_PRIMARY_IP}" -U postgres -d "${ALLOYDB_PRODUCTS_DATABASE_NAME}" -c "
    CREATE EXTENSION IF NOT EXISTS vector;
    CREATE EXTENSION IF NOT EXISTS google_ml_integration CASCADE;
    GRANT EXECUTE ON FUNCTION embedding TO postgres;
"

# Products - Table
psql -h "${ALLOYDB_PRIMARY_IP}" -U postgres -d "${ALLOYDB_PRODUCTS_DATABASE_NAME}" -c "
    CREATE TABLE IF NOT EXISTS ${ALLOYDB_PRODUCTS_TABLE_NAME} (
        id TEXT PRIMARY KEY,
        name TEXT,
        description TEXT,
        picture TEXT,
        price_usd_currency_code TEXT,
        price_usd_units INTEGER,
        price_usd_nanos BIGINT,
        categories TEXT,
        product_embedding VECTOR(768),
        embed_model TEXT
    );
"

# 3. Data Population
echo "Generating and importing product data..."
pushd "${SCRIPTS_DIR}" > /dev/null

if [ ! -f "products.json" ]; then
    echo "Error: products.json not found in ${SCRIPTS_DIR}"
    popd > /dev/null
    exit 1
fi

python3 ./generate_sql_from_products.py > products.sql
psql -h "${ALLOYDB_PRIMARY_IP}" -U postgres -d "${ALLOYDB_PRODUCTS_DATABASE_NAME}" -f products.sql
rm products.sql

popd > /dev/null

# 4. Generate Embeddings
echo "Generating vector embeddings (via AlloyDB AI)..."
psql -h "${ALLOYDB_PRIMARY_IP}" -U postgres -d "${ALLOYDB_PRODUCTS_DATABASE_NAME}" -c "
    UPDATE ${ALLOYDB_PRODUCTS_TABLE_NAME}
    SET product_embedding = embedding('${MODEL}', description), embed_model='${MODEL}'
    WHERE product_embedding IS NULL;
"

echo "Database Setup Complete!"
