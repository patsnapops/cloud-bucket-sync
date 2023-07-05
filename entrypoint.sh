#/bin/bash
mkdir -p /opt/logs/apps

# 如果没有debug变量则默认为false
if [ -z "$DEBUG" ]; then
    DEBUG=false
fi

# 判断服务类型如果是manager则跑manager，如果是worker则启动worker
if [ "$SERVICE_TYPE" == "manager" ]; then
    cbs manager start --debug=$DEBUG --log /opt/logs/apps/app.log -c ~/.cbs/ -p 8080
    else
    cbs worker start --debug=$DEBUG --cloud=$CLOUD --region=$REGION --log /opt/logs/apps/app.log  -c ~/.cbs/ --thread 8
fi
