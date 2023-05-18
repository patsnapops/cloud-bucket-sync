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
# 递归查询一个大量的对象，可以使用chan特性，-q 100 指定chan的缓冲区大小，-r 递归查询， -l limit 指定查询的数量
cbs b ls s3://patent-familydata/ -q 100 -l 100 -r
```

```bash
# 删除对象 -p 指定profile， -f 强制删除， --thread-num 指定并发数
cbs b rm s3://zhoushoujiantest/ -p us0066 -r --limit 100 -f --thread-num 30
```

```bash