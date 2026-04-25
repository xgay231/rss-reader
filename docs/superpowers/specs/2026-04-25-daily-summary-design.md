# 每日文章总结功能设计

## 概述

为 RSS 阅读器添加每日总结功能：每天定时将收到的新文章聚合为一份摘要，发送至用户邮箱。同时支持手动触发。

## 功能需求

- 用户可自定义每日总结发送时间
- 支持选择特定分组或订阅源（默认所有订阅源）
- 当天新文章（标题 + 链接）聚合后由 AI 生成一段合并摘要
- 通过 SMTP 发送 HTML 格式邮件
- 提供手动触发按钮便于测试

## 数据模型变更

### User 模型扩展

在 `backend/models/user.go` 中新增字段：

```go
DailySummaryEnabled bool   `json:"dailySummaryEnabled" bson:"dailySummaryEnabled"`
DailySummaryTime    string `json:"dailySummaryTime" bson:"dailySummaryTime"` // 格式 "HH:MM"
DailySummaryEmail   string `json:"dailySummaryEmail" bson:"dailySummaryEmail"` // 发送目标邮箱
SmtpPassword        string `json:"-" bson:"smtpPassword"` // SMTP 密码/授权码，不暴露给前端
```

## API 设计

### 新增接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/daily-summary/send` | POST | 手动触发发送每日总结 |
| `/api/daily-summary/settings` | GET | 获取每日总结设置 |
| `/api/daily-summary/settings` | PUT | 更新每日总结设置 |

### 设置接口扩展

`GET/PUT /api/settings` 响应/请求体新增字段：

```json
{
  "feedUpdateInterval": 15,
  "autoSummary": true,
  "dailySummaryEnabled": true,
  "dailySummaryTime": "09:00",
  "dailySummaryEmail": "user@example.com"
}
```

注意：`SmtpPassword` 仅在 PUT 时接收，不返回给前端。

## 核心流程

### 手动触发流程

1. 前端调用 `POST /api/daily-summary/send`
2. 后端鉴权（从 context 获取 userID）
3. 查询该用户当天（0:00 至当前时间）新增的文章
4. 将文章标题 + 链接组成列表，调用 AI 生成合并摘要
5. 将摘要组装成 HTML 邮件，通过 SMTP 发送
6. 返回发送结果（成功/失败信息）

### 定时任务流程

1. 服务器启动时启动一个定时调度器（`robfig/cron`）
2. 每分钟检查所有启用了每日总结的用户
3. 如果当前时间（精确到分钟）匹配用户的 `dailySummaryTime`，触发发送流程
4. 发送逻辑复用手动触发流程

### 文章查询范围

- 时间范围：当天 00:00:00 至 23:59:59
- 来源：用户选择的所有订阅源 / 分组（暂定默认所有订阅源）

## AI 摘要生成

调用现有 AI 摘要服务，将当天文章列表聚合成一段话。

输入示例：
```
请为用户生成一段今日文章摘要。

文章列表：
1. "AI 技术新突破" - 来源: 科技日报
2. "2026 年投资趋势分析" - 来源: 财经周刊
3. "如何提高工作效率" - 来源: 生活指南

请生成一段 100-200 字的合并摘要，概括今日文章的核心内容。
```

输出：一段简洁的合并摘要

## 邮件格式

HTML 格式，结构如下：

```html
<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
  <h2 style="color: #333;">📬 每日文章总结</h2>
  <p style="color: #666;">2026年4月25日</p>
  <hr style="border: 1px solid #eee;">

  <h3 style="color: #444;">今日概览</h3>
  <p>你今天收到了 <strong>5</strong> 篇文章，涉及 AI科技、投资理财 等主题。</p>

  <h3 style="color: #444;">智能摘要</h3>
  <p style="line-height: 1.6;">今日文章涵盖了 AI 技术的新进展...（AI 生成的合并摘要）</p>

  <h3 style="color: #444;">文章列表</h3>
  <ul style="line-height: 1.8;">
    <li><a href="https://..." style="color: #0066cc;">文章标题1</a> - 来源</li>
    <li><a href="https://..." style="color: #0066cc;">文章标题2</a> - 来源</li>
  </ul>
</div>
```

## 依赖

- `github.com/robfig/cron/v3` - 定时调度
- `github.com/go-mail/mail/v2` - SMTP 发送邮件

## 文件变更

- `backend/models/user.go` - 新增字段
- `backend/main.go` - 新增 API handlers、定时调度逻辑
- `frontend/` - 设置页面添加每日总结配置项、手动发送按钮

## 测试计划

1. 单元测试：每日总结查询、邮件组装
2. 手动测试：设置时间、手动触发、验证邮件内容
3. 定时测试：修改时间为当前时间 +1 分钟，验证是否准时发送
