# 杩愮淮鎸囧崡

> 閫傜敤鐗堟湰锛歷1.0.x  
> 闈㈠悜锛歋RE銆佽繍缁淬€佸€肩彮宸ョ▼甯?
## 1. 閮ㄧ讲褰㈡€?
| 褰㈡€?| 鍦烘櫙 | 鍏抽敭鐐?|
|------|------|--------|
| 瑁告満 / VM `systemd` | 鑷缓鏈烘埧 | 鍗曚竴 cron 浠诲姟 |
| Docker | 娴嬭瘯銆佸皬瑙勬ā | 鎸傝浇 `config.yaml` |
| Kubernetes CronJob | 浜戝師鐢?| 鎺ㄨ崘鐢熶骇 |

## 2. 浜岃繘鍒堕儴缃?
### 2.1 鏋勫缓

```bash
make build VERSION=v1.1.0
# 浜х墿: ./report (Linux/macOS) 鎴?report.exe (Windows)
```

### 2.2 鐩綍缁撴瀯锛堝缓璁級

```
/opt/report/
鈹溾攢鈹€ bin/report                 # 鍙墽琛?鈹溾攢鈹€ config/config.yaml         # 閰嶇疆锛堥檺 600 鏉冮檺锛?鈹溾攢鈹€ output/                    # xlsx 杈撳嚭锛堝彲閫夛級
鈹斺攢鈹€ logs/                      # 鏃ュ織锛堝彲閫夛級
```

### 2.3 systemd 鍗曞厓

```ini
# /etc/systemd/system/report@daily.service
[Unit]
Description=Report Generator
After=network.target

[Service]
Type=oneshot
User=report
WorkingDirectory=/opt/report
ExecStart=/opt/report/bin/report
EnvironmentFile=/opt/report/config/report.env
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### 2.4 瀹氭椂

```bash
# /etc/cron.d/report
# m h dom mon dow user  command
0 9 * * *  report  systemctl start report@daily.service
```

## 3. Docker 閮ㄧ讲

### 3.1 鏋勫缓

```bash
make docker-build
# 鎴栧甫鐗堟湰锛歮ake docker-build VERSION=v1.1.0
```

### 3.2 杩愯

```bash
docker run -d \
  --name report \
  --restart on-failure:3 \
  -v /opt/report/config.yaml:/config.yaml:ro \
  peterydd/report:latest
```

### 3.3 璋冭瘯

```bash
docker logs -f report
docker exec -it report /bin/sh
```

## 4. Kubernetes CronJob

### 4.1 瀹屾暣绀轰緥

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: report-config
data:
  config.yaml: |
    database:
      driver: mysql
      source: "user:pass@tcp(mysql:3306)/db"
    smtp:
      host: smtp.example.com
      port: "587"
      username: report@example.com
      password: REPLACE_ME   # 瀹為檯浠?Secret 娉ㄥ叆
    reports:
      - { ... }
---
apiVersion: v1
kind: Secret
metadata:
  name: report-secret
type: Opaque
stringData:
  smtp-password: your-password
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-report
spec:
  schedule: "0 9 * * *"
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  startingDeadlineSeconds: 600
  jobTemplate:
    spec:
      backoffLimit: 2
      template:
        metadata:
          labels: { app: report }
        spec:
          restartPolicy: OnFailure
          containers:
            - name: report
              image: peterydd/report:v1.1.0
              imagePullPolicy: IfNotPresent
              env:
                - name: CONFIG_PATH
                  value: /etc/report
              volumeMounts:
                - name: cfg
                  mountPath: /etc/report
                  readOnly: true
                - name: out
                  mountPath: /tmp
              resources:
                requests: { cpu: 250m, memory: 256Mi }
                limits:   { cpu: 1000m, memory: 2Gi }
              securityContext:
                runAsNonRoot: true
                runAsUser: 65532
                allowPrivilegeEscalation: false
                capabilities: { drop: [ALL] }
                readOnlyRootFilesystem: true
          volumes:
            - name: cfg
              configMap: { name: report-config }
            - name: out
              emptyDir: {}
```

### 4.2 鍏抽敭鐐?
- `concurrencyPolicy: Forbid` 闃叉涓婃鏈畬鎴愭椂鏂颁竴娆″惎鍔?- `backoffLimit: 2` 澶辫触鍚庨噸璇?2 娆?- `securityContext` 浠ラ潪 root 鐢ㄦ埛杩愯銆佸彧璇绘牴鏂囦欢绯荤粺
- 閰嶇疆鏂囦欢鐢?ConfigMap锛涙晱鎰熷瓧娈电敤 Secret锛堟洿浣冲疄璺垫槸鐢?`external-secrets-operator`锛?
## 5. 鏃ュ織

### 5.1 杈撳嚭

宸ュ叿浣跨敤 Go 鏍囧噯 `log` 鍖咃紝杈撳嚭鍒?**stderr**銆?
鏍蜂緥锛?
```
2024/05/01 09:00:00 configuration loaded successfully
2024/05/01 09:00:00 sheet 璁㈠崟鏄庣粏 using streaming mode, batch size: 20000
2024/05/01 09:00:01 sheet 璁㈠崟鏄庣粏 streaming query completed, 12345 rows processed
2024/05/01 09:00:02 report 閿€鍞姤琛╛20240501_090002.xlsx generated successfully
2024/05/01 09:00:03 email 姣忔棩閿€鍞姤琛?sent successfully
```

### 5.2 鍏抽敭浜嬩欢娓呭崟

| 鍏抽敭璇?| 鍚箟 |
|--------|------|
| `configuration loaded` | 鍚姩鎴愬姛 |
| `using streaming mode` | sheet 璧版祦寮?|
| `query completed` | 鍗?sheet 瀹屾垚 |
| `generated successfully` | xlsx 钀界洏 |
| `sent successfully` | 閭欢宸插彂鍑?|
| `failed:` | 澶辫触锛堝繀鏈夊師鍥狅級 |

### 5.3 闆嗕腑鏃ュ織

鐢熶骇寤鸿锛?
- **Loki / ELK**锛氱敤 vector / fluentbit 閲囬泦 stderr
- **JSON 杈撳嚭**锛坴1.1锛夛細鏇挎崲 `log` 涓?`zap` / `zerolog`
- **PII 鑴辨晱**锛氫笉瑕佸湪鏃ュ織涓緭鍑哄瘑鐮?/ DSN

### 5.4 鏃ュ織杞浆

- systemd journal锛氳嚜鍔?- Docker锛歚--log-opt max-size=10m --log-opt max-file=3`
- 瑁告満锛歭ogrotate

## 6. 鐩戞帶

### 6.1 鍋ュ悍妫€鏌?
CronJob 妯″紡娌℃湁甯搁┗杩涚▼锛岀洃鎺у簲鑱氱劍 **Job 瀹屾垚鐘舵€?*锛?
```yaml
# PrometheusRule 绀轰緥
- alert: ReportJobFailed
  expr: kube_job_status_failed{job_name=~"daily-report-.*"} > 0
  for: 1m
  annotations:
    summary: "Report Job 澶辫触"
```

### 6.2 鑷畾涔夋寚鏍囷紙v1.2 璁″垝锛?
v1.2 灏嗗鍑?Prometheus 鎸囨爣锛?
- `report_query_duration_seconds{report, sheet}`
- `report_excel_size_bytes{report}`
- `report_email_send_total{status}`
- `report_active_sheet_goroutines`

### 6.3 涓氬姟鐩戞帶

- 鏀朵欢浜烘敹鍒伴偖浠?鈫?閫氳繃 SMTP 鎶曢€掓棩蹇楋紙澶栭儴锛?- xlsx 鏂囦欢澶у皬 鈫?寮傚父灏忥紙濡?< 1KB锛夊彲鑳芥槸绌烘煡璇?
## 7. 澶囦唤涓庢仮澶?
### 7.1 澶囦唤

- 閰嶇疆鏂囦欢锛氱敤 Git 绠＄悊
- 鐢熸垚鐨?xlsx锛氬彲閫夋寔涔呭嵎 / 瀵硅薄瀛樺偍

### 7.2 鎭㈠

- 閲嶆柊閮ㄧ讲 CronJob 鍗冲彲锛屽伐鍏锋棤鐘舵€?
## 8. 鍗囩骇

```bash
# 1. 鎷夊彇鏂扮増鏈?git pull
make build VERSION=v1.1.0

# 2. 鐏板害涓€鍙版満鍣?systemctl start report@daily.service  # 瑙傚療鏃ュ織

# 3. 鎺ㄩ€侀暅鍍忥紙K8s 妯″紡锛?docker push peterydd/report:v1.1.0
kubectl set image cronjob/daily-report report=peterydd/report:v1.1.0
```

璇︾粏杩佺Щ娉ㄦ剰浜嬮」瑙?[docs/migration.md](migration.md)銆?
## 9. 鏁呴殰鎺掓煡

### 9.1 Job 涓€鐩?Pending

- 妫€鏌?imagePullSecrets
- 妫€鏌?nodeSelector / tolerations
- 妫€鏌ヨ祫婧愰厤棰?
### 9.2 Job Failed

```bash
kubectl describe job <name>
kubectl logs <pod>
```

甯歌鍘熷洜锛?- `failed to read configuration file` 鈫?ConfigMap 鏈寕杞芥垨璺緞閿?- `unsupported database driver` 鈫?`driver` 鍊奸敊
- DB 杩炴帴瓒呮椂 鈫?闃茬伀澧?/ DSN
- SMTP 璁よ瘉澶辫触 鈫?瀵嗙爜閿欒

### 9.3 閭欢鏈敹鍒?
1. 鏌ョ湅 Job 鏃ュ織涓?`email ... sent successfully`
2. 鏀朵欢浜烘煡鍨冨溇閭欢
3. SMTP 鏈嶅姟鍣ㄦ姇閫掓棩蹇楋紙Postfix锛歚/var/log/maillog`锛?4. SPF / DKIM 閰嶇疆

### 9.4 鎶ヨ〃涓虹┖浣嗘棤閿欒

- SQL 鏈韩鏃犵粨鏋?- Sheet 鍚嶅寘鍚壒娈婂瓧绗﹁鏇挎崲
- 鍐欏叆鏃舵病璁?`column`锛屽垪鏁板涓嶄笂

### 9.5 OOM Killed

- 澶ф暟鎹噺鏈惎鐢?stream
- 鍑忓皬 `batchSize`
- 鎻愰珮鍐呭瓨 limit

### 9.6 鏁版嵁绔炰簤

- `go test -race ./...` 澶嶇幇
- 鍏变韩鍙橀噺鏈姞閿?
## 10. 瀹归噺瑙勫垝

| 鎶ヨ〃瑙勬ā | CPU | 鍐呭瓨 | 纾佺洏 | 缃戠粶 |
|----------|-----|------|------|------|
| 灏忥紙< 1 涓囪锛?| 250m | 256Mi | 100MB锛堝惈 xlsx锛?| 浣?|
| 涓紙10 涓囪锛?| 500m | 512Mi | 200MB | 涓?|
| 澶э紙鐧句竾琛岋級 | 1000m | 2Gi | 1GB+ | 楂橈紙DB鈫扐pp鈫扴MTP锛?|

> 閭欢闄勪欢 > 20MB 寤鸿鏀圭敤瀵硅薄瀛樺偍 + 閾炬帴銆?
## 11. 鍏宠仈閾炬帴

- [閰嶇疆鍙傝€僝(configuration.md)
- [瀹夊叏瀹炶返](security.md)
- [寮€鍙戞寚鍗梋(development.md)
- [鏋舵瀯鏂囨。](architecture.md)
