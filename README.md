# report
report是一个用golang实现的数据报表发送程序，通过sql查询数据库并将数据写入excel表格，最终通过邮件发送给用户。

支持数据库类型为oracle、mysql、postgresql。

支持容器化部署，支持k8s cronjob控制器部署。

## 使用方法

1. 下载源码

```bash
git clone https://github.com/peterydd/report.git
```

2. 编译

```bash
# 下载依赖
go mod tidy

# 编译二进制文件
make build
# 或者跳过test，编译二进制文件
make build-skip-test
```

3. 配置文件

```bash
cp configs/config.yaml.example config.yaml
```

修改config.yaml中的配置信息

4. 运行

```bash
./report
```

## 配置文件说明
```yaml
database:
  driver: 0 # 0: oracle 1: mysql 2: postgressql
  source: "oracle://test:password@127.0.0.1:1521/FREEPDB1"
  # source: "test:password@tcp(localhost:3306)/test" # mysql
  # source: "postgres://test:password@127.0.0.1:5432/test?sslmode=disable" # postgresql
smtp: # 发件服务器配置信息
  host: "smtp.example.com"
  port: "25"
  username: "test@example.com"
  password: "password"
reports:
  - name: "报表名称" # 任务名称
    workBook:
      prefix: "报表名称_" # 报表文件名称前缀
      dateFormat: "20060102150405" # 报表文件名时间格式，采用golang的时间格式
      suffix: ".xlsx" # 报表文件名称后缀
    sheets:
      - name: "sheet页1" # sheet页名称
        sql: "select col1,col2,col3,col4,col5 from table1" # sheet页内容查询sql
        column: "字段1,字段2,字段3,字段4,字段5" # sheet页内容字段名称
        isSum: false # sheet页内容是否需要合计
        sumBeginColumn: 0 # sheet页内容合计开始列
      - name: "sheet页2" # sheet页名称
        sql: "select col1,col2,col3,col4,col5 from table2" # sheet页内容查询sql
        column: "字段1,字段2,字段3,字段4,字段5" # sheet页内容字段名称
        isSum: true # sheet页内容是否需要合计
        sumBeginColumn: 3 # sheet页内容合计开始列
    message:
      from: "test@example.com" # 发件人
      to: ["test@outlook.com", "test@qq.com"] # 收件人列表
      cc: ["test@gmail.com"] # 抄送列表
      bcc: ["test@189.cn"] # 密送人列表
      subject: "test主题" # 邮件主题
      body: |
        test正文
            测试邮件，请查收附件！
      contentType: "text/plain;charset=utf-8" # 邮件内容类型及字符编码
      attachment:
        contentType: "text/plain;charset=utf-8" # 附件内容类型及字符编码
        withFile: true # 是否携带附件
```

## 编译docker镜像

```bash
make docker-build
```

## 部署

### k8s cronjob部署

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: report
  namespace: default
  labels:
    app: report
    version: v1
data:
  report-config: |
    database:
      driver: 0
      source: "oracle://test:password@127.0.0.1:1521/FREEPDB1"
    smtp:
      host: "smtp.example.com"
      port: "25"
      username: "test@example.com"
      password: "password"
    reports:
      - name: "报表名称"
        workBook:
          prefix: "报表名称_"
          dateFormat: "20060102150405"
          suffix: ".xlsx"
        sheets:
          - name: "sheet页1"
            sql: "select col1,col2,col3,col4,col5 from table1"
            column: "字段1,字段2,字段3,字段4,字段5"
            isSum: false
            sumBeginColumn: 0
          - name: "sheet页2"
            sql: "select col1,col2,col3,col4,col5 from table2"
            column: "字段1,字段2,字段3,字段4,字段5"
            isSum: true
            sumBeginColumn: 3
        message:
          from: "test@example.com"
          to: ["test@outlook.com", "test@qq.com"]
          cc: ["test@gmail.com"]
          bcc: ["test@189.cn"]
          subject: "test主题"
          body: |
            test正文
                测试邮件，请查收附件！
          contentType: "text/plain;charset=utf-8"
          attachment:
            contentType: "text/plain;charset=utf-8"
            withFile: true
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: report
  namespace: default
  labels:
    app: report
spec:
  schedule: "*/30 * * * *"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          volumes:
            - name: config
              configMap:
                name: report
                items:
                  - key: report-config
                    path: config.yaml
          containers:
            - name: report
              image: peterydd/report:v1.0.3
              imagePullPolicy: IfNotPresent
              env:
                - name: TZ
                  value: Asia/Shanghai
              resources:
                requests:
                  cpu: 1000m
                  memory: 256Mi
                limits:
                  cpu: 2000m
                  memory: 512Mi
              volumeMounts:
                - name: config
                  mountPath: "/config.yaml"
                  subPath: config.yaml
          restartPolicy: OnFailure
```