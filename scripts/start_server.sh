#!/bin/bash

go build . && sudo ./fgtun -s flee://dummy:hello@127.0.0.1:9997
