---
name: aliyunpan-web
description: 通过 aliyunpan-api 的网页版 Web 接口操作阿里云盘。当用户需要查看、搜索、上传、下载、移动、重命名、收藏、删除、还原文件，或生成下载直链、管理分享链接时使用。This skill should be used when the user wants to browse, search, upload, download, move, rename, star, delete, restore, or share files on Aliyun Drive through the project's Web API.
---

# 阿里云盘 Web 接口

封装 `aliyunpan-api` 的 `aliyunpan_web` 包，提供命令行工具 `aliyunpan-web/cmd/aliyunpan-web`。
凭证保存在**技能目录**的 `.env`，整个技能目录可以直接移动后继续使用。

## 运行方式

以下以技能目录位于项目根目录的 `aliyunpan-web/` 为例；如果移动到 `~/.codebuddy/skills/aliyunpan-web`，把命令中的前缀替换成该目录即可。`.env` 和 `bin/` 会跟随技能目录一起移动。

优先使用技能自带的二进制（无需进入项目目录）：

```powershell
powershell -NoProfile -File aliyunpan-web/scripts/aliyunpan-web.ps1 <命令> [参数...]
```

或直接执行：

```powershell
aliyunpan-web/bin/aliyunpan-web.exe <命令> [参数...]
```

二进制缺失或源码更新后，重新构建：

```powershell
powershell -NoProfile -File aliyunpan-web/scripts/build.ps1
```

技能目录移出项目后，构建时用 `-Repo` 指定 aliyunpan-api 仓库路径：

```powershell
powershell -NoProfile -File aliyunpan-web/scripts/build.ps1 -Repo D:\path\to\aliyunpan-api
```

也可以在项目根目录直接跑源码：

```powershell
go run ./aliyunpan-web/cmd/aliyunpan-web <命令> [参数...]
```

## 凭证

- 默认读取技能目录 `.env`；通过 `--env <path>` 或环境变量 `ALIYUNPAN_ENV_FILE` 可覆盖。
- 关键字段：`ALIYUNPAN_ACCESS_TOKEN`、`ALIYUNPAN_REFRESH_TOKEN`、`ALIYUNPAN_EXPIRE_TIME`、`ALIYUNPAN_APP_ID`、`ALIYUNPAN_DEVICE_ID`。
- `access_token` 过期或格式无效时，自动用 `refresh_token` 刷新，并把新令牌写回 `.env`。刷新信息输出到 stderr。
- 不要提交 `.env`；不要打印完整 token，除非用户明确要求。

## 常用命令

| 目的 | 命令 |
| --- | --- |
| 查看账号与网盘 ID | `whoami --json` |
| 强制刷新令牌 | `token --json` |
| 列出目录 | `ls "/我的文档" --json` |
| 递归列出 | `ls --all "/照片" --json` |
| 查看详情 | `stat "/我的文档/a.pdf" --json` |
| 通配符搜索 | `search "/照片/2026*" --json` |
| 创建目录 | `mkdir "/备份/2026"` |
| 删除到回收站 | `rm "/备份/a.zip"` |
| 移动 | `mv "/备份/a.zip" "/备份/2026"` |
| 重命名 | `rename "/备份/a.zip" "b.zip"` |
| 收藏 / 取消 | `star "/备份/a.zip"`、`unstar "/备份/a.zip"` |
| 下载直链 | `url "/备份/a.zip" --json` |
| 下载 | `dl "/备份/a.zip" ./a.zip --force` |
| 上传 | `upload ./a.zip "/备份"` |
| 分享 | `share create "/备份/a.zip" --pwd 1234 --expire "2026-12-31 23:59:59"` |
| 分享列表 / 取消 | `share list --json`、`share cancel <分享ID>` |
| 回收站 | `recycle list --json`、`recycle restore <文件ID>`、`recycle rm --force <文件ID>`、`recycle clear --force` |

## 智能体执行规则

1. 调用时加 `--json`，只解析 stdout；忽略 stderr 中的进度和令牌刷新日志。
2. 先 `ls` / `stat` / `search` 确认路径，再执行 `rm`、`mv`、`rename` 等写操作。
3. 远程路径使用以 `/` 开头的绝对路径；默认网盘是备份盘，资源库/相册/保险箱用 `--drive resource|album|safe`。
4. `recycle rm`、`recycle clear` 不可恢复，执行前必须向用户确认，确认后才加 `--force`。
5. `dl` 覆盖本地文件需要 `--force`；上传重名由服务端自动重命名。
6. 一次只执行一个写操作，失败时保留错误原文，不要静默重试删除类操作。

## 已知限制

- `share create` 可能被服务端拒绝，返回“请升级至最新版本使用此功能”，属于接口侧限制；遇到时告知用户，不要反复重试。
- 上传按 4MB 分片，单文件最多 10000 片；超大文件会自动增大分片。
- 目录下载未实现，先用 `ls --all` 找到文件再逐个 `dl`。
- 接口限流时返回“请求太频繁”，稍后再试。

详细参数、返回结构和排错说明见 `references/commands.md`。
