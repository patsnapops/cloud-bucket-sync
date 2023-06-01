## Config 配置
### cli 配置
```yaml
profiles:
  - name: default
    ak: YOUR_AK
    sk: YOUR_SK
    region: cn-northwest-1
    endpoint: http://your_proxy_host:your_port

  - name: us0066
    ak: AKIAJU********
    sk: 4QY************
    region: us-east-1
    endpoint: https://s3.us-east-1.amazonaws.com

manager:
  endpoint: "http://localhost:8012"
  api_version: "api/v1/"
```

## 最佳实践

### 1. 提交一个或多个任务
  
```bash
[root@zhoushoujianworkspace cloud-bucket-sync]# go run main.go task apply -f tests/task.json --dry-run 
taskID: e549bcef-e4b5-4348-b8d6-8db99ae2037d
```
[task.json](../tests/task.json)

### 2.查询task任务

```bash
[root@zhoushoujianworkspace cloud-bucket-sync]# cbs -c ./config/ task show
                   ID                  |                   NAME                   | WORKERTAG | SYNCMODE |  SUBMITTER   |                     RECORDS                      
---------------------------------------+------------------------------------------+-----------+----------+--------------+--------------------------------------------------
  5722eca6-1951-41c6-bb56-f4f179affe89 | patent_ans_map_snap                      | aws-us    | syncOnce | fanlipeng    | pending:0,running:0,success:0,failed:0,cancel:0  
  5773d3d5-597c-45c3-9eda-64495d61f7bd | bio-tidb-solr-mon                        | aws-us    | syncOnce | chenkele     | pending:0,running:0,success:0,failed:0,cancel:0  
  1c1d84ae-a39f-41e4-8887-66cc1b5d0e2f | 测试定时执行任务                         | aws-cn    | syncOnce | zhoushoujian | pending:0,running:0,success:0,failed:0,cancel:0  
  22a2eaaf-842a-4d2c-9a40-b1d480c17dee | rdSyncTaskJson                           | aws-us    | syncOnce | lushijie     | pending:0,running:0,success:0,failed:0,cancel:0  
  b2bc6a20-dbad-471b-9a86-e2de73387d8b | entity_logo128                           | tx-cn     | keepSync | taozhixue    | pending:0,running:0,success:0,failed:0,cancel:0  
  37d9a17f-b1c4-44d0-9fb2-98121463921f | rdSyncTask                               | aws-us    | keepSync | lushijie     | pending:0,running:0,success:0,failed:0,cancel:0  
---------------------------------------+------------------------------------------+-----------+----------+--------------+--------------------------------------------------
                                                                                                              COUNT     |                        6                         
                                                                                                         ---------------+--------------------------------------------------
```

### 3.查询task任务详情

```bash
[root@zhoushoujianworkspace cloud-bucket-sync]# cbs -c ./config/ task show 1c1d84ae-a39f-41e4-8887-66cc1b5d0e2f
{
   "id": "1c1d84ae-a39f-41e4-8887-66cc1b5d0e2f",
   "created_at": "0001-01-01T00:00:00Z",
   "updated_at": "0001-01-01T00:00:00Z",
   "is_deleted": false,
   "is_server_side": true,
   "worker_tag": "aws-cn", -- 这里要尽可能选择靠近目标桶的worker，因为如果你的节点流量走的nat，那么nat不管进出都收费，nat外部挂的EIP也会收费。所以我们尽可能在nat下只用流量进，因为EIP进流量是不收费。
   "name": "测试定时执行任务",
   "source_url": "s3://ops-9554/s3-proxy-test/",
   "target_url": "s3://ops-9554/zhoushoujiantest/popapi/",
   "source_profile": "cn9554",
   "target_profile": "cn9554",
   "sync_mode": "syncOnce",
   "submitter": "zhoushoujian",
   "corn": "",
   "keys_url": "",
   "is_silence": false,
   "is_overwrite": false,
   "time_before": "",
   "time_after": "",
   "include": "",
   "exclude": "",
   "storage_class": "STANDARD",
   "meta": "",
   "records": null
}
```


## 任务配置说明
### 1. 任务配置文件
```json
{
  "name": "测试定时执行任务",
  "source_url": "s3://ops-9554/s3-proxy-test/",
  "target_url": "s3://ops-9554/zhoushoujiantest/popapi/",
  "source_profile": "cn9554",
  "target_profile": "cn9554",
  "sync_mode": "syncOnce", // syncOnce（一次同步）, keepSync（保持同步）
  "submitter": "zhoushoujian",
  "worker_tag": "aws-cn",
  "corn": "", // 定时任务表达式 “分 时 日 月 周”
  "keys_url": "", // 指定需要同步的文件列表，文件内容为每行一个文件路径
  "is_silence": false, // 是否静默执行,钉钉通知会检验此字段
  "is_server_side": true, // 标记任务是否可以使用serverCopy 即后台copy，流量不需要走本地，比如 同一个地域下的桶。如果是跨地域的桶，需要设置为false，要走公网下载后再上传。
  "is_overwrite": false, // 是否强制覆盖源文件，如果为false，会跳过已经存在的文件，如果为true，默认还会校验文件的md5值，如果不一致，会覆盖
  "time_before": "", // 指定同步文件的最后修改时间，格式为：2006-01-02 15:04:05
  "time_after": "", // 指定同步文件的最后修改时间，格式为：2006-01-02 15:04:05
  "include": "", // 指定需要同步的文件，支持通配符，多个文件以逗号分隔
  "exclude": "", // 指定不需要同步的文件，支持通配符，多个文件以逗号分隔
  "storage_class": "STANDARD", // 指定目标存储类型，支持：STANDARD（标准存储）、STANDARD_IA（低频访问）、GLACIER（归档存储）
  "meta": "" // 指定目标文件的元数据，格式为：key1=value1,key2=value2（暂未实际使用到）
}
```