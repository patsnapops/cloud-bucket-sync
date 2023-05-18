## 简介
cbs 是一个云原生的用途不同云对象存储的同步转移的工具。目前支持S3协议类型的对象存储同步。附带的还做了cli工具，可以用来查看同步状态，同步进度以及对象操作等。
## 架构
![架构图](）
## 使用
- [cbs bucket command](./docs/cbs-bucket.md)
- [cbs manager command](./docs/cbs-manager.md)
```bash
docker build -t cbs .
docker-compose up -d
```