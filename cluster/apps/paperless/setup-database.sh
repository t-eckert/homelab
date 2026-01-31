#!/bin/bash
# Script to create paperless database and user in PostgreSQL

kubectl exec -n postgres deployment/postgres -- psql -U postgres -d homelab -c "
CREATE DATABASE paperless;
GRANT ALL PRIVILEGES ON DATABASE paperless TO postgres;
"

echo "Paperless database created successfully"