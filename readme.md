## 简介
cbs 是一个云原生的用途不同云对象存储的同步转移的工具。目前支持S3协议类型的对象存储同步。附带的还做了cli工具，可以用来查看同步状态，同步进度以及对象操作等。
## 架构
![架构图](）
## 功能介绍
### 1.task管理
- 支持任务提交`cbs task apply -f task.json`
- 支持任务执行`cbs task exec {task_id}`
- 支持任务查看 `cbs task show {task_id}`

更多信息： [cbs task command](./docs/cbs-task.md)
### 2.对象管理

- 支持对象查询`cbs bucket ls {s3_url} -l {limit}`
- 支持对象删除`cbs bucket rm {s3_url}`

更多信息： [cbs bucket command](./docs/cbs-bucket.md)
### 3.manager功能
- 提供API接口
- 提供自修复任务
- 提供任务状态钉钉机器人告警

更多信息：[cbs manager command](./docs/cbs-manager.md)

### 4.worker功能（开发中。。。）
- 提供任务执行，自身带上cloud和region信息，用来跑时候自身的任务（出于流量费用，网络速度的考虑）

更多信息：[cbs worker command](./docs/cbs-worker.md)
