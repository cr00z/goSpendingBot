#!/bin/bash

DBSTRING="host=$POSTGRES_HOST port=5432 user=$POSTGRES_USER password=$POSTGRES_PASSWORD sslmode=$POSTGRES_SSLMODE"

goose postgres "$DBSTRING" up