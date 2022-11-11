#!/bin/bash

echo "input first param:="$1
if test -z $1; then
   echo "not find first param.exit!!!"
   exit
fi

work_path=$(cd $(dirname $0); pwd)

bin=$1

`ulimit -Sc unlimited`

cur_pid=`cat ./cur.pid 2>/dev/null`
#echo "pid:=$cur_pid"

if test -n "$cur_pid"; then
   echo "kill -9 $cur_pid"
   `kill -9 $cur_pid 2>/dev/null`
fi

datetag=`date '+%m_%d_%k_%M'`

`cd ${work_path}`
ls ./*.trace  2>/dev/null | xargs -r tar -czf "./SERVER_TRACE.${datetag}.tar.gz"

ls ./*.trace 2>/dev/null | xargs -r rm -fr

ls ./*.log.*  2>/dev/null | xargs -r tar -czf "./LOG.${datetag}.tar.gz"

ls ./*.log.* 2>/dev/null | xargs -r rm -fr
ls ./core.*  2>/dev/null | xargs -r tar -czf "./CORE.${datetag}.tar.gz"

ls ./core* 2>/dev/null | xargs -r rm -fr

ls ./nohup.out 2>/dev/null | xargs -r rm -fr

#nohup ${work_path}/${bin} --log_dir=${work_path} --logbufsecs=0 --stderrthreshold=3 --vmodule=time_tracer=1 --cfg=${work_path}/cfg.xml --max_log_size=128 &
nohup ${work_path}/${bin} -log_dir=log -alsologtostderr -v=3 &

echo $! > cur.pid



