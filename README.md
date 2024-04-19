# report
report是一个用golang实现的数据报表发送程序，通过sql查询数据库并将数据写入excel表格，最终通过邮件发送给用户。

支持数据库类型为oracle、mysql、postgresql。

## 使用方法
1. 下载源码
```shell
git clone 
```

2. 编译
```shell
# 下载依赖
go mod tidy

# 编译二进制文件
make build
# 或者跳过test，编译二进制文件
make build-skip-test
```

3. 配置文件
```shell
cp configs/config.yaml.example config.yaml
```
修改config.yaml中的配置信息

4. 运行
```shell
./report
```

## 配置文件说明
```toml
[database]
type = "mysql"
host = "

```
