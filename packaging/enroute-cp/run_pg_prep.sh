#!/bin/bash

set -e
sed -i 's/\(^host.*all.*all.*\)md5/\1trust/' /etc/postgresql/11/main/pg_hba.conf
