[supervisord]
logfile=/supervisord/supervisord.log            ; supervisord log file
logfile_maxbytes=5MB                            ; maximum size of logfile before rotation
logfile_backups=10                              ; number of backed up logfiles
loglevel=trace                                  ; info, debug, warn, trace
pidfile=/supervisord/supervisord.pid            ; pidfile location
nodaemon=true									; do not run supervisord as a daemon
minfds=1024                                     ; number of startup file descriptors
minprocs=200                                    ; number of process descriptors
user=postgres                                   ; default user
childlogdir=/supervisord                        ; where child log files will live

[unix_http_server]
file = /supervisord/supervisord.sock
username = postgres
password = postgres

[program:postgres-prep]
command=/bin/run_pg_prep.sh
user=postgres
autorestart=false
startretries=0
redirect_stderr=false
priority=800

[program:run-postgresql]
command=/usr/lib/postgresql/11/bin/postgres -D /var/lib/postgresql/11/main -c config_file=/etc/postgresql/11/main/postgresql.conf
user=postgres
autorestart=true
nodaemon=false
priority=900

; 3.2 run migrations, start hasura, start enroute
[program:run-migrations-start-hasura-enroute]
command=/bin/run_migrations.sh
user=postgres
autorestart=false
priority=930
