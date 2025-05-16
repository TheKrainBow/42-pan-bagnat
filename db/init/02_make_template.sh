#!/usr/bin/env bash
set -e

DB="panbagnat"
TEMPLATE_DB="schema_template"

# Check if the template already exists
exists=$(psql -U admin -d "$DB" -tAc "SELECT 1 FROM pg_database WHERE datname = '$TEMPLATE_DB';")

if [ "$exists" != "1" ]; then
  echo "Creating template database $TEMPLATE_DB from $DB…"
  psql -U admin -d "$DB" -c "CREATE DATABASE $TEMPLATE_DB WITH TEMPLATE = '$DB';"
else
  echo "Template database $TEMPLATE_DB already exists, skipping."
fi
