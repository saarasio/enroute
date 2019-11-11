#!/bin/bash

export PROJECT_DIR=my-project-7; $PWD/hasura-cli init --directory $PROJECT_DIR --endpoint http://localhost:8888 && cd $PROJECT_DIR && ../hasura-cli migrate status && ../hasura-cli migrate create --endpoint http://localhost:8888 --from-server --schema saaras_db --log-level DEBUG .
