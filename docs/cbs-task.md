## Config 配置
### cli 配置
```yaml
profiles:
  - name: default
    ak: YOUR_AK
    sk: YOUR_SK
    region: cn-northwest-1
    endpoint: http://local.s3-proxy.patsnap.info

  - name: us0066
    ak: AKIAJU********
    sk: 4QY************
    region: us-east-1
    endpoint: https://s3.us-east-1.amazonaws.com

manager:
  endpoint: "http://localhost:8080"
  api_version: "api/v1/"
```

## 最佳实践
### 查询task任务

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

### 查询task任务详情

```bash
[root@zhoushoujianworkspace cloud-bucket-sync]# cbs -c ./config/ task show 1c1d84ae-a39f-41e4-8887-66cc1b5d0e2f
{
   "id": "1c1d84ae-a39f-41e4-8887-66cc1b5d0e2f",
   "created_at": "0001-01-01T00:00:00Z",
   "updated_at": "0001-01-01T00:00:00Z",
   "is_deleted": false,
   "worker": "aws-cn",
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