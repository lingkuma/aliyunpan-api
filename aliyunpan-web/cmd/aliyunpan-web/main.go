package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type options struct {
	json    bool
	verbose bool
	all     bool
	force   bool
	help    bool

	envPath string
	drive   string
	out     string
	pwd     string
	expire  string
	timeout time.Duration
}

func main() {
	opts, args, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		fmt.Fprintln(os.Stderr, "使用 --help 查看用法")
		os.Exit(2)
	}
	if opts.help || len(args) == 0 {
		printUsage()
		return
	}

	cmd := args[0]
	rest := args[1:]
	if cmd == "help" {
		printUsage()
		return
	}

	a, err := newApp(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}

	switch cmd {
	case "whoami", "user", "me":
		err = a.runWhoami()
	case "token", "refresh":
		err = a.runToken()
	case "ls", "list":
		err = a.runLs(rest)
	case "stat", "info":
		err = a.runStat(rest)
	case "search", "find":
		err = a.runSearch(rest)
	case "mkdir":
		err = a.runMkdir(rest)
	case "rm", "del", "delete":
		err = a.runRm(rest)
	case "mv", "move":
		err = a.runMv(rest)
	case "rename", "ren":
		err = a.runRename(rest)
	case "star":
		err = a.runStar(rest, true)
	case "unstar":
		err = a.runStar(rest, false)
	case "url", "link":
		err = a.runURL(rest)
	case "dl", "download":
		err = a.runDownload(rest)
	case "upload", "up":
		err = a.runUpload(rest)
	case "share":
		err = a.runShare(rest)
	case "recycle", "rb", "trash":
		err = a.runRecycle(rest)
	default:
		printUsage()
		err = fmt.Errorf("未知命令: %s", cmd)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func (a *app) runWhoami() error {
	ui, err := a.userInfo()
	if err != nil {
		return err
	}
	if a.opts.json {
		return printJSON(ui)
	}
	fmt.Printf("昵称: %s\n用户名: %s\n用户ID: %s\n状态: %s\n总空间: %s\n已用空间: %s\n可用空间: %s\n",
		ui.Nickname, ui.UserName, ui.UserId, ui.Status,
		humanSize(int64(ui.TotalSize)), humanSize(int64(ui.UsedSize)), humanSize(int64(ui.TotalSize)-int64(ui.UsedSize)))
	fmt.Println("网盘ID:")
	fmt.Printf("  备份盘(file): %s\n", ui.FileDriveId)
	fmt.Printf("  资源库(resource): %s\n", ui.ResourceDriveId)
	fmt.Printf("  相册(album): %s\n", ui.AlbumDriveId)
	fmt.Printf("  保险箱(safe): %s\n", ui.SafeBoxDriveId)
	return nil
}

func (a *app) runToken() error {
	tok, err := a.refreshToken()
	if err != nil {
		return err
	}
	if a.opts.json {
		return printJSON(tok)
	}
	fmt.Printf("access_token: %s\nrefresh_token: %s\nexpires_in: %d\nexpire_time: %s\n",
		tok.AccessToken, tok.RefreshToken, tok.ExpiresIn, tok.ExpireTime)
	return nil
}

func parseArgs(args []string) (*options, []string, error) {
	opts := &options{timeout: 120 * time.Second}
	positional := make([]string, 0, len(args))

	takeValue := func(i *int, name, inline string) (string, error) {
		if inline != "" {
			return inline, nil
		}
		if *i+1 >= len(args) {
			return "", fmt.Errorf("参数 %s 缺少取值", name)
		}
		*i++
		return args[*i], nil
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		name := arg
		inline := ""
		if strings.HasPrefix(arg, "--") {
			if idx := strings.IndexByte(arg, '='); idx > 0 {
				name = arg[:idx]
				inline = arg[idx+1:]
			}
		}

		switch name {
		case "-h", "--help":
			opts.help = true
		case "-j", "--json":
			opts.json = true
		case "-v", "--verbose":
			opts.verbose = true
		case "-a", "--all":
			opts.all = true
		case "-f", "--force":
			opts.force = true
		case "--env":
			v, err := takeValue(&i, name, inline)
			if err != nil {
				return nil, nil, err
			}
			opts.envPath = v
		case "--drive":
			v, err := takeValue(&i, name, inline)
			if err != nil {
				return nil, nil, err
			}
			opts.drive = v
		case "--out":
			v, err := takeValue(&i, name, inline)
			if err != nil {
				return nil, nil, err
			}
			opts.out = v
		case "--pwd":
			v, err := takeValue(&i, name, inline)
			if err != nil {
				return nil, nil, err
			}
			opts.pwd = v
		case "--expire":
			v, err := takeValue(&i, name, inline)
			if err != nil {
				return nil, nil, err
			}
			opts.expire = v
		case "--timeout":
			v, err := takeValue(&i, name, inline)
			if err != nil {
				return nil, nil, err
			}
			if d, parseErr := time.ParseDuration(v); parseErr == nil {
				opts.timeout = d
			} else if sec, atoiErr := strconv.Atoi(v); atoiErr == nil {
				opts.timeout = time.Duration(sec) * time.Second
			} else {
				return nil, nil, fmt.Errorf("--timeout 取值无效: %s", v)
			}
		default:
			return nil, nil, fmt.Errorf("未知参数: %s", arg)
		}
	}
	return opts, positional, nil
}

func printUsage() {
	fmt.Print(`aliyunpan-web - 阿里云盘 Web 接口命令行工具

用法:
  go run ./aliyunpan-web/cmd/aliyunpan-web [全局选项] <命令> [参数...]

全局选项:
  --json, -j              以 JSON 输出（智能体推荐）
  --drive <id|file|resource|album|safe>  选择网盘，默认备份盘
  --env <path>            指定 .env 文件，默认读取项目根目录 .env
  --all, -a               ls 递归列出所有层级
  --force, -f             覆盖本地文件 / 执行不可恢复操作
  --out <path>            dl 的本地保存路径
  --pwd <4位密码>         share create 的提取码
  --expire <时间>         share create 的过期时间，如 "2026-12-31 23:59:59"
  --timeout <秒|时长>     HTTP 超时，默认 120s
  --verbose, -v           输出调试信息
  -h, --help              显示帮助

命令:
  whoami                       用户信息与各网盘ID
  token                        强制刷新 access_token 并写回 .env
  ls [远程路径]                列出目录（默认根目录）
  stat <远程路径>              查看文件/目录详情
  search <通配符路径>          按通配符搜索，如 "/照片/2026*"
  mkdir <远程路径>             创建目录（自动递归）
  rm <远程路径...>             删除到回收站
  mv <源路径...> <目标目录>    移动文件/目录
  rename <远程路径> <新名称>   重命名
  star <远程路径...>           收藏
  unstar <远程路径...>         取消收藏
  url <远程路径>               获取下载直链
  dl <远程路径> [本地路径]     下载文件
  upload <本地文件> [远程目录] 上传文件（默认根目录）
  share create <路径...>       创建分享
  share list                   分享列表
  share cancel <分享ID...>     取消分享
  recycle list                 回收站列表
  recycle restore <文件ID...>  从回收站还原
  recycle rm --force <文件ID...>  彻底删除
  recycle clear --force        清空回收站

示例:
  go run ./aliyunpan-web/cmd/aliyunpan-web --json whoami
  go run ./aliyunpan-web/cmd/aliyunpan-web ls "/我的文档"
  go run ./aliyunpan-web/cmd/aliyunpan-web --json ls --all "/照片"
  go run ./aliyunpan-web/cmd/aliyunpan-web upload ./demo.zip "/备份"
  go run ./aliyunpan-web/cmd/aliyunpan-web dl "/备份/demo.zip" ./demo.zip --force
  go run ./aliyunpan-web/cmd/aliyunpan-web share create "/我的文档/a.pdf" --pwd 1234

Token:
  默认读取项目根目录 .env 中的 ALIYUNPAN_ACCESS_TOKEN / ALIYUNPAN_REFRESH_TOKEN。
  access_token 过期或格式无效时，会自动用 refresh_token 刷新并写回 .env。
`)
}
