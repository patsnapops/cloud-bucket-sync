#/bin/bash
# 环境变量获取debug模式
if [ "$DEBUG" == "true" ]; then
    cbs manager start --debug
    else
    cbs manager start
fi
