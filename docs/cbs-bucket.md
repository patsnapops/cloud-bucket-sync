## Config 配置
### cli 配置
```yaml
profiles:
  - name: default
    ak: YOUR_AK
    sk: YOUR_SK
    region: cn-northwest-1
    endpoint: http://local.s3-proxy.patsnap.info
```

## 最佳实践

### bucket 查询相关

```bash
# 递归查询一个大量的对象，可以使用chan特性，-r 递归查询， -l limit 指定查询的数量
cbs b ls s3://patent-familydata/ -l 100 -r
# 如果对象load时间很长，-q 100 指定chan的缓冲区大小，来及时处理已经加载到内存的对象而不需要等待全部加载完毕
cbs b ls s3://patent-familydata/ -l 100 -r -q 100
```
### bucket 删除相关
```bash
# 删除对象 -p 指定profile， -f 强制删除， --thread-num 指定并发数，删除默认指定了-q 1000来及时处理已经加载到内存的对象而不需要等待全部加载完毕
cbs b rm s3://patent-familydata/ -p us0066 -r --limit 100 -f --thread-num 30
```
```bash
# 删除对象来源于指定文件 .txt txt格式必须是每个key为一行（兼容空格，空行）
cbs b rm s3://patent-familydata/ -c ./config/ -p us0066 -d --dry-run --file keys.txt 
# 删除对象来源于指定文件 .csv
cbs b rm s3://patent-familydata/ -c ./config/ -p us0066 -f --dry-run --file efc35b3c-2453-4aba-9935-1b28f331ccad.csv 
# 删除对象来源于指定目录，处理目录下所有 txt和csv结尾的文件
cbs b rm s3://patent-familydata/ -c ./config/ -p us0066 -f --dry-run --dir ./ 
```

#### 性能相关
```bash
# 单线程删除 100个对象，耗时 32秒
cbs b rm s3://patent-familydata/ -p us0066 -r --limit 100 -f -c ./config/
···
delete patent-familydata/10/00/41/05/10004105.XML success
delete patent-familydata/10/00/41/13/10004113.XML success

Total Objects: 100, Total Size: 579.22 KB, Cost Time: 32.012681066s
```
```bash
# 10线程删除 100个对象，耗时 6秒
cbs b rm s3://patent-familydata/ -p us0066 -r --limit 100 -f -c ./config/ --thread-num 10
···
delete patent-familydata/10/00/80/94/10008094.XML success
delete patent-familydata/10/00/81/33/10008133.XML success
delete patent-familydata/10/00/81/26/10008126.XML success

Total Objects: 100, Total Size: 578.13 KB, Cost Time: 5.98809868s
```

```bash
# 10线程删除 10000个对象，耗时 5分4秒
cbs b rm s3://patent-familydata/ -p us0066 -r --limit 10000 -f -c ./config/ --thread-num 10
···
delete patent-familydata/10/54/35/33/10543533.XML success
delete patent-familydata/10/54/35/34/10543534.XML success
delete patent-familydata/10/54/35/38/10543538.XML success
delete patent-familydata/10/54/35/43/10543543.XML success

Total Objects: 10000, Total Size: 51.94 MB, Cost Time: 5m4.816449186s
```
```bash
# 100线程删除 10000个对象，耗时 45秒
cbs b rm s3://patent-familydata/ -p us0066 -r --limit 10000 -f -c ./config/ --thread-num 100
···
delete patent-familydata/10/57/31/00/10573100.XML success
delete patent-familydata/10/57/30/91/10573091.XML success
delete patent-familydata/10/57/31/07/10573107.XML success

Total Objects: 10000, Total Size: 21.10 MB, Cost Time: 45.873002895s
```

thread-num | 100 | 10000 |640万| 1秒对象|失败个数
---|---|---|---|---|---
1 | 32.012681066s | - |- | 3.125个/s|-
10 | 5.98809868s | 5m4.816449186s | - |16.666个/s 32.89个/s|-
100 | - | 45.873002895s - |-| 217.39个/s|-
150 | - | - | 5h37m57.563273064s|  326.08个/s|1700