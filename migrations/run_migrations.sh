#!/bin/bash
set -e
PGDSN="${POSTGRES_DSN:-postgres://bench:bench@localhost:5432/bench}"
for f in $(ls migrations/*.sql | sort); do
    echo "Applying $f..."
    psql "$PGDSN" -f "$f"
done
echo "All migrations applied."
