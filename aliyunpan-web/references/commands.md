# aliyunpan-web 命令参考

## 全局选项

| 选项 | 说明 |
| --- | --- |
| `--json`, `-j` | 以 JSON 输出，推荐智能体使用 |
| `--drive <id\|file\|resource\|album\|safe>` | 选择网盘，默认 `file`（备份盘） |
| `--env <path>` | 指定 `.env`，默认技能目录 `.env` |
| `--all`, `-a` | `ls` 递归列出所有层级 |
| `--force`, `-f` | 覆盖本地文件 / 执行不可恢复操作 |
| `--out <path>` | `dl` 的本地保存路径 |
| `--pwd <4位>` | `share create` 的提取码 |
| `--expire <时间>` | `share create` 的过期时间 |
| `--timeout <秒\|时长>` | HTTP 超时，默认 120s |
| `--verbose`, `-v` | 输出调试信息 |
| `-h`, `--help` | 帮助 |

## .env 字段

```dotenv
ALIYUNPAN_ACCESS_TOKEN=...
ALIYUNPAN_REFRESH_TOKEN=...
ALIYUNPAN_EXPIRES_IN=7200
ALIYUNPAN_EXPIRE_TIME=2026-10-19 12:41:55
ALIYUNPAN_APP_ID=25dzX3vbYqktVxyX
ALIYUNPAN_DEVICE_ID=<32位随机字符串>
ALIYUNPAN_DEVICE_NAME=Chrome浏览器
ALIYUNPAN_MODEL_NAME=Windows网页版
# 可选：固定默认网盘
ALIYUNPAN_DRIVE_ID=
```

- 查找顺序：`--env` > `ALIYUNPAN_ENV_FILE` > 可执行文件所在技能目录 `.env` > 项目根目录 `.env`。
- 刷新令牌后自动回写 `ALIYUNPAN_ACCESS_TOKEN`、`ALIYUNPAN_REFRESH_TOKEN`、`ALIYUNPAN_EXPIRES_IN`、`ALIYUNPAN_EXPIRE_TIME`，保留其他字段和注释。
- `expire_time` 支持 RFC3339（`2026-10-19T04:34:13Z`）和本地时间（`2026-10-19 12:41:55`）。

## 命令

### whoami

```powershell
aliyunpan-web --json whoami
```

返回 `UserInfo`，包含 `fileDriveId`、`resourceDriveId`、`albumDriveId`、`safeBoxDriveId`、`userId`、空间用量等。

### token

```powershell
aliyunpan-web --json token
```

强制用 `refresh_token` 换新令牌并写回 `.env`。返回 `accessToken`、`refreshToken`、`expiresIn`、`expireTime`。

### ls

```powershell
aliyunpan-web ls "/我的文档"
aliyunpan-web --json ls "/我的文档"
aliyunpan-web --json ls --all "/照片"
```

- 路径不存在时退出码为 1，stderr 输出“获取文件信息失败”。
- JSON 输出 `FileEntity` 数组：`fileName`、`fileId`、`fileType`（`file`/`folder`）、`fileSize`、`path`、`updatedAt`、`parentFileId` 等。
- `--all` 会递归遍历所有子目录，文件多时较慢，可能触发限流。

### stat

```powershell
aliyunpan-web --json stat "/我的文档/a.pdf"
```

返回单个 `FileEntity`。

### search

```powershell
aliyunpan-web --json search "/照片/2026*"
aliyunpan-web --json search "/备份/*.zip"
```

支持 `*`、`?`、`[]` 通配符，路径必须是绝对路径。`*` 不跨越 `/`。

### mkdir

```powershell
aliyunpan-web mkdir "/备份/2026/10"
```

自动逐级创建，已存在的目录会跳过。返回 `fileId`。

### rm

```powershell
aliyunpan-web rm "/备份/a.zip"
aliyunpan-web rm "/备份/a.zip" "/备份/b.zip"
```

删除到回收站，可恢复。支持通配符路径。

### mv

```powershell
aliyunpan-web mv "/备份/a.zip" "/备份/2026"
aliyunpan-web mv "/备份/a.zip" "/备份/b.zip" "/归档"
```

最后一个参数是目标目录，其余为源路径。

### rename

```powershell
aliyunpan-web rename "/备份/a.zip" "b.zip"
```

只改名称，不支持跨目录。

### star / unstar

```powershell
aliyunpan-web star "/备份/a.zip"
aliyunpan-web unstar "/备份/a.zip"
```

支持通配符和多个路径。

### url

```powershell
aliyunpan-web --json url "/备份/a.zip"
```

返回 `url`、`expiration`、`size` 等。直链默认 4 小时有效。

### dl

```powershell
aliyunpan-web dl "/备份/a.zip" ./a.zip
aliyunpan-web dl "/备份/a.zip" ./a.zip --force
aliyunpan-web dl "/备份/a.zip" --out ./downloads/
```

- 仅支持文件，不支持目录。
- 本地已存在时默认报错，`--force` 覆盖。
- 下载地址被阿里云盘屏蔽时返回明确错误。

### upload

```powershell
aliyunpan-web upload ./a.zip "/备份"
aliyunpan-web --json upload ./a.zip "/备份"
```

- 目标目录不存在时先 `mkdir`。
- 同名文件由服务端自动重命名。
- 支持秒传；大文件按分片上传并显示进度到 stderr。
- JSON 输出含 `fileId`、`name`、`size`。

### share

```powershell
aliyunpan-web --json share create "/备份/a.zip" --pwd 1234 --expire "2026-12-31 23:59:59"
aliyunpan-web --json share list
aliyunpan-web share cancel <分享ID>
```

- `--pwd` 必须是 4 个字符；省略表示公开分享。
- `--expire` 省略表示永久有效。
- 若返回“请升级至最新版本使用此功能”，是服务端限制，停止重试并告知用户。

### recycle

```powershell
aliyunpan-web --json recycle list
aliyunpan-web recycle restore <文件ID>
aliyunpan-web recycle rm --force <文件ID>
aliyunpan-web recycle clear --force
```

- `restore` 还原到原位置。
- `rm` 与 `clear` 不可恢复，必须加 `--force`。
- 回收站条目没有路径，只能按 `fileId` 操作。

## 常见错误

| 错误 | 处理 |
| --- | --- |
| `缺少 refresh_token` | 在 `.env` 配置 `ALIYUNPAN_REFRESH_TOKEN` |
| `刷新 access_token 失败` | refresh_token 过期，需重新从浏览器获取 |
| `请求太频繁` | 触发限流，等待后再试，避免并发 |
| `你账号已超出最大登录设备数量` | 在阿里云盘客户端下线一台设备，或更换 `ALIYUNPAN_DEVICE_ID` |
| `请升级至最新版本使用此功能` | 分享接口被服务端限制 |
| `文件不存在` | 用 `ls` / `search` 确认路径和网盘（`--drive`） |
