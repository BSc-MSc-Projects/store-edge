#!/bin/bash

# Remove old garbage
docker image rm -f edge:latest
cd edge_deploy
docker-compose down
cd ..
cp -r ~/.aws/ .	# copy the AWS credentials 

# Build the new images 
docker build -t edge -f Dockerfile.edge .

# Remove the credentials
rm -r .aws

# Now, run the actual compose file
cd edge_deploy
docker-compose up

