#!/bin/bash

go build . && sudo ./fgtun -c flee://dummy:hello@172.0.0.1:9997
