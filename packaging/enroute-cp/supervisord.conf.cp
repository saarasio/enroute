[supervisord]
logfile=/var/log/supervisord/supervisord.log    ; supervisord log file
logfile_maxbytes=5MB                            ; maximum size of logfile before rotation
logfile_backups=10                              ; number of backed up logfiles
loglevel=trace                                  ; info, debug, warn, trace
pidfile=/var/run/supervisord.pid                ; pidfile location
nodaemon=true									; do not run supervisord as a daemon
minfds=1024                                     ; number of startup file descriptors
minprocs=200                                    ; number of process descriptors
user=root                                       ; default user
childlogdir=/var/log/supervisord/               ; where child log files will live

[program:fix-access-for-postgres]
command=sed -i 's/\(^host.*all.*all.*\)md5/\1trust/' /etc/postgresql/11/main/pg_hba.conf
user=postgres
autorestart=false
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
user=root
autorestart=false
priority=930
