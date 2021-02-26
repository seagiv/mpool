#!/bin/sh

ssh -fN -L 3306:127.0.0.1:3306 root@admin-test.minerall.io
/root/dbgw/dbgw

