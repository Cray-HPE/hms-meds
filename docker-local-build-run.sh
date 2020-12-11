#! /bin/bash
docker run $(docker build . | tee /dev/tty | tail -n 1 | awk '{print $3}')