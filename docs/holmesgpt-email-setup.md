# HolmesGPT 邮件通知配置指南

本文档详细说明如何为 HolmesGPT 配置邮件通知功能。

## 📧 邮件配置概述

HolmesGPT 可以通过 Robusta 的邮件 sink 发送分析结果到指定邮箱。支持两种配置方式：
1. **SMTP 配置** - 使用标准 SMTP 服务器
2. **Amazon SES 配置** - 使用 AWS SES 服务

## 🔧 SMTP 配置

### Gmail 配置示例

```yaml
sinksConfig:
  - mail_sink:
      name: holmes_email_sink
      mailto: "mailtos://your-gmail@gmail.com:your-app-password@smtp.gmail.com:587?from=robusta-alerts@yourdomain.com&to=heningyu21@126.com"
      with_header: true
```

**Gmail 应用密码设置步骤：**
1. 登录 Gmail 账户
2. 进入 Google 账户设置
3. 启用两步验证
4. 生成应用专用密码
5. 使用应用密码替换 `your-app-password`

### 163 邮箱配置示例

```yaml
sinksConfig:
  - mail_sink:
      name: holmes_email_sink
      mailto: "mailtos://your-163-email@163.com:your-password@smtp.163.com:465?from=your-163-email@163.com&to=heningyu21@126.com"
      with_header: true
```

### QQ 邮箱配置示例

```yaml
sinksConfig:
  - mail_sink:
      name: holmes_email_sink
      mailto: "mailtos://your-qq-email@qq.com:your-auth-code@smtp.qq.com:587?from=your-qq-email@qq.com&to=heningyu21@126.com"
      with_header: true
```

**QQ 邮箱授权码获取：**
1. 登录 QQ 邮箱
2. 设置 → 账户
3. 开启 SMTP 服务
4. 生成授权码

### 企业邮箱配置示例

```yaml
sinksConfig:
  - mail_sink:
      name: holmes_email_sink
      mailto: "mailtos://username:password@mail.company.com:587?from=alerts@company.com&to=heningyu21@126.com"
      with_header: true
```

## 🔐 使用 Kubernetes Secret 存储敏感信息

为了安全起见，建议将邮件密码存储在 Kubernetes Secret 中：

### 1. 创建 Secret

```bash
kubectl create secret generic mail-secrets \
  --from-literal=smtp-password='your-email-password' \
  --namespace=robusta
```

### 2. 在配置中引用 Secret

```yaml
sinksConfig:
  - mail_sink:
      name: holmes_email_sink
      mailto: "mailtos://your-email@gmail.com:{{env.SMTP_PASSWORD}}@smtp.gmail.com:587?from=robusta-alerts@yourdomain.com&to=heningyu21@126.com"
      with_header: true

holmes:
  additionalEnvVars:
    - name: SMTP_PASSWORD
      valueFrom:
        secretKeyRef:
          name: mail-secrets
          key: smtp-password
```

## ☁️ Amazon SES 配置

如果您使用 AWS SES，可以使用以下配置：

```yaml
sinksConfig:
  - mail_sink:
      name: holmes_email_sink
      mailto: "mailtos://heningyu21@126.com"
      use_ses: true
      aws_region: "us-east-1"
      from_email: "robusta-alerts@yourdomain.com"
      aws_access_key_id: "{{env.AWS_ACCESS_KEY_ID}}"
      aws_secret_access_key: "{{env.AWS_SECRET_ACCESS_KEY}}"
      with_header: true

holmes:
  additionalEnvVars:
    - name: AWS_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: aws-secrets
          key: access-key-id
    - name: AWS_SECRET_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: aws-secrets
          key: secret-access-key
```

## 📝 邮件模板自定义

您可以自定义邮件内容模板：

```yaml
sinksConfig:
  - mail_sink:
      name: holmes_email_sink
      mailto: "mailtos://your-email@gmail.com:password@smtp.gmail.com:587?from=alerts@company.com&to=heningyu21@126.com"
      with_header: true
      # 自定义邮件主题和内容
      subject_template: "🚨 Kubernetes 告警 - {{alert_name}} ({{cluster_name}})"
      body_template: |
        集群告警通知
        
        告警详情：
        - 告警名称: {{alert_name}}
        - 集群名称: {{cluster_name}}
        - 命名空间: {{namespace}}
        - 严重程度: {{severity}}
        - 发生时间: {{alert_time}}
        
        HolmesGPT 分析结果：
        {{holmes_analysis}}
        
        请及时处理此告警。
```

## 🧪 测试邮件配置

### 1. 部署测试 Pod

```bash
kubectl apply -f https://raw.githubusercontent.com/robusta-dev/kubernetes-demos/main/crashpod/broken.yaml
```

### 2. 检查邮件发送日志

```bash
kubectl logs -n robusta deployment/robusta-runner | grep -i mail
```

### 3. 手动触发测试

```bash
kubectl create job test-alert --from=cronjob/nonexistent-cronjob || echo "This should trigger an alert"
```

## 🔧 故障排除

### 常见问题

1. **邮件发送失败**
   - 检查 SMTP 服务器地址和端口
   - 验证用户名和密码
   - 确认防火墙设置

2. **Gmail 认证失败**
   - 确保启用了两步验证
   - 使用应用专用密码而不是账户密码
   - 检查 Google 账户安全设置

3. **邮件格式问题**
   - 检查 `mailto` URL 格式
   - 验证发件人和收件人邮箱地址
   - 确认 `with_header` 设置

### 调试命令

```bash
# 查看 Robusta 日志
kubectl logs -n robusta deployment/robusta-runner -f

# 查看邮件 sink 配置
kubectl get configmap -n robusta robusta-config -o yaml

# 测试网络连接
kubectl run test-pod --image=busybox --rm -it -- nslookup smtp.gmail.com
```

## 📚 参考资料

- [Robusta 邮件 Sink 官方文档](https://docs.robusta.dev/master/configuration/sinks/mail.html)
- [HolmesGPT 配置指南](https://docs.robusta.dev/master/configuration/holmesgpt/index.html)
- [Kubernetes Secret 管理](https://kubernetes.io/docs/concepts/configuration/secret/)

## 🎯 下一步

配置完成后，您可以：
1. 监控邮件通知是否正常发送
2. 根据需要调整分析触发条件
3. 自定义邮件模板和内容
4. 配置更多的告警场景分析
