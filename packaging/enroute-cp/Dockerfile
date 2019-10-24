#FROM golang:1.12.7 AS build
#FROM envoyproxy/envoy:latest

# (1) Copy/Install Artifacts
	# (1.1) Install postgresql on container
	# (1.2) Install supervisord on container
	# (1.3) Copy migrations to container
	# (1.4) Copy hasura binaries to container
   # (1.4.1) Make hasura binaries executable
   # (1.4.2) Copy hasura migrations script
	# (1.5) Copy supervisord config file
	# (1.6) Copy script that waits for hasura
	# (1.7) Copy enroute-cp
   
# (2) Setup environment variables
	# (2.1) Setup environment variables to run migrations script

# (2) Run supervisord
	# (3.1) Start postgresql
	# (3.2) Run migrations with autorestart to false - only runs it once
   	# (3.2.1) Migration script runs hasura temporarily and then kills it.
	# (3.3) Start hasura

FROM ubuntu:16.04

# 1.1
RUN apt-get update
RUN apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys B97B0AFCAA1A47F044F244A07FCC7D46ACCC4CF8
RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ xenial-pgdg main" > /etc/apt/sources.list.d/pgdg.list
RUN apt-get update && apt-get install -y python-software-properties software-properties-common postgresql-11 postgresql-client-11 postgresql-contrib-11

#1.2
RUN apt-get update && apt-get install -y supervisor vim netcat net-tools sed
RUN mkdir -p /var/log/supervisord

# 1.3
RUN ["mkdir", "-p", "/migrations"]
COPY migrations/* /migrations/

# 1.4
COPY hasura/* /bin/

# 1.4.1
RUN chmod +x /bin/hasura-cli

# 1.4.2
COPY run_migrations.sh /bin/
RUN chmod +x /bin/run_migrations.sh

COPY wait_for_postgres.sh /bin/
RUN chmod +x /bin/wait_for_postgres.sh

# 1.5
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# 1.6
COPY wait_for_hasura.sh  bin/
RUN chmod +x /bin/wait_for_hasura.sh

# 1.7 copy enroute-cp
COPY enroute-cp /bin/

# 2
ENV HASURA_GRAPHQL_CLI_ENVIRONMENT=server-on-docker
ENV HASURA_GRAPHQL_DATABASE_URL=postgres://postgres:@localhost:5432/postgres
ENV HASURA_GRAPHQL_ENABLE_CONSOLE="true"
ENV HASURA_GRAPHQL_MIGRATIONS_DIR=/migrations
ENV HASURA_GRAPHQL_SERVER_PORT=8888

# Only allow connections from localhost
RUN echo "host all  all    127.0.0.1/32  trust" >> /etc/postgresql/11/main/pg_hba.conf
# Only listen on localhost
# RUN echo "listen_addresses='*'" >> /etc/postgresql/11/main/postgresql.conf

# Allow other processes to reach posgresql on port 5432
EXPOSE 5432

# Allow other processes to reach hasura on port 8080
EXPOSE 8888

# Setup volume for postgresql
VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql", "/var/lib/postgresql/11/main"]

ENV WEBAPP_SECRET="treeseverywhere"

CMD ["/usr/bin/supervisord"]
