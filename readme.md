## 简介
`cbs` 是一个云原生的用来实现不同云对象存储的同步的工具。可以托管任务，也可以cli执行。目前支持S3协议类型的对象存储同步。
- 支持对象操作，包括查询、删除、下载、上传、复制、同步等（比aws cli的优势是支持不同云的对象，或者aws不同地域的桶同步）
- 支持将同步任务提交到后台执行，支持任务状态查询，支持任务状态钉钉机器人告警
- cbs自身的实现了任务管理功能，确保提交的任务能够被执行。任务支持周期性执行和实时目录同步
## manager架构
![](./docs/index.png)
## 功能介绍
### 1.task管理
- 支持任务提交`cbs task apply -f task.json`
- 支持任务执行`cbs task exec {task_id}`
- 支持任务查看 `cbs task show {task_id}`

更多信息： [cbs task command](./docs/cbs-task.md)
### 2.对象管理

- 支持对象查询`cbs bucket ls {s3_url} -l {limit}`
- 支持对象删除`cbs bucket rm {s3_url}`
- 支持对象下载`cbs bucket sync {s3_url} {local_path}`
- 支持对象上传`cbs bucket sync {local_path} {s3_url}`
- 支持对象复制、同步（支持跨不同账号，不同云厂商，需要支持s3协议）`cbs bucket sync {s3_url} {s3_url}`

更多信息： [cbs bucket command](./docs/cbs-bucket.md)
### 3.manager功能
- 提供API接口
- 提供自修复任务
- 提供任务状态钉钉机器人告警

更多信息：[cbs manager command](./docs/cbs-manager.md)

### 4.worker功能
- 提供任务执行，自身带上cloud和region信息，用来跑时候自身的任务（出于流量费用，网络速度的考虑）
- 支持任务并发执行

更多信息：[cbs worker command](./docs/cbs-worker.md)
