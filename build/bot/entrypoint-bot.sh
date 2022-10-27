#!/bin/sh

# Start the first proce
touch /app/offsets.yaml
/app/file.d --config /app/config.yml &

# Start the second process
/app/bot 2>&1 | tee /app/log.txt &

# Wait for any process to exit
wait -n

# Exit with status of process that exited first
exit $?