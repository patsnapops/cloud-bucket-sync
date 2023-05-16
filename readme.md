## 最佳实践

### bucket 查询相关

```bash
# 递归查询一个大量的对象，可以使用chan特性，-q 100 指定chan的缓冲区大小，-r 递归查询， -l limit 指定查询的数量
cbs b ls s3://patent-familydata/ -q 100 -l 100 -r
```