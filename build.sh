#!/bin/bash
# Build script for the project
go build
cp cbs /usr/local/bin/cbs
/usr/local/bin/cbs -v
if [ $? -eq 0 ]; then
    echo "Build successful"
    cbs b sync /usr/local/bin/cbs s3://ops-9554/bin/
    cbs b ls s3://ops-9554/bin/cbs
else
    echo "Build failed"
fi
