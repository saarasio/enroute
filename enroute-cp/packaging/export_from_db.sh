#!/bin/bash

export PROJECT_DIR=my-project-7; hasura init --directory $PROJECT_DIR --endpoint http://localhost:8081 && cd $PROJECT_DIR && hasura migrate status && hasura migrate create --endpoint http://localhost:8081 --from-server --schema saaras_db --log-level DEBUG .
