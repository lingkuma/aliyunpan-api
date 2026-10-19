# aliyunpan-api
GO语言封装的 aliyunpan 阿里云盘官方OpenAPI接口和网页端Web接口。你可以基于该接口库实现对阿里云盘的二次开发。   
两种接口都可以实现对阿里云盘的文件访问，但是由于阿里OpenAPI目前接口开放有限，有部分功能只有web端接口具备，例如：分享、相册等。你可以根据需要自行选择，也可以两者融合使用。

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/tickstep/aliyunpan-api?tab=doc)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://raw.githubusercontent.com/modern-go/concurrent/master/LICENSE)

# 关于登录Token
阿里官方OpenAPI的登录和网页版Web登录token是不一样的，方式也不一样，本仓库并没有这部分代码，你需要自行实现。
1. 阿里官方OpenAPI的登录token，可以参考官方文档：https://www.yuque.com/aliyundrive/zpfszx/btw0tw
2. 网页版Web的登录token，可以通过使用浏览器获取到的RefreshToken进行获取，样例如下：
```javascript
(async () => {
  // 1) 自动寻找存 refresh_token 的 localStorage 项
  let refreshToken = "", fromKey = "";
  for (const k of Object.keys(localStorage)) {
    const v = localStorage.getItem(k);
    if (!v || !v.includes("refresh_token")) continue;
    try {
      const j = JSON.parse(v);
      if (j.refresh_token) { refreshToken = j.refresh_token; fromKey = k; break; }
    } catch (e) {}
  }
  if (!refreshToken) {
    console.error("没找到 refresh_token，请改用 Network 面板抓 account/token 的响应");
    return;
  }
  console.log("来源 key:", fromKey, "| token 长度:", refreshToken.length);

  // 2) 换取 access_token
  const res = await fetch("https://auth.aliyundrive.com/v2/account/token", {
    method: "POST",
    headers: {
      "content-type": "application/json;charset=UTF-8",
      "origin": "https://www.aliyundrive.com",
      "referer": "https://www.aliyundrive.com/"
    },
    body: JSON.stringify({
      refresh_token: refreshToken,
      api_id: "pJZInNHN2dZWk8qg",
      grant_type: "refresh_token"
    })
  });
  const data = await res.json();
  console.log("HTTP", res.status, data);

  if (data.access_token) {
    console.table({
      access_token:  data.access_token,
      refresh_token: data.refresh_token,
      expires_in:    data.expires_in,
      expire_time:   data.expire_time
    });
    console.warn("旧的 refresh_token 已作废，请把上面这个新的另行保存");
  }
})();

```

# 快速使用教程
## 阿里官方OpenAPI接口
导入包
```
import "github.com/tickstep/aliyunpan-api/aliyunpan"
import "github.com/tickstep/aliyunpan-api/aliyunpan_open"
```

使用授权登录后得到的AccessToken创建OpenPanClient实例
```
	openPanClient := aliyunpan_open.NewOpenPanClient(openapi.ApiConfig{
		TicketId:     "",
		UserId:       "",
		ClientId:     "",
		ClientSecret: "",
	}, openapi.ApiToken{
		AccessToken: "eyJraWQiOiJLcU8iLC...jIUeqP9mZGZDrFLN--h1utcyVc",
		ExpiredAt:   1709527182,
	}, nil)
```

调用OpenPanClient相关方法可以实现对阿里云盘的相关操作
```
	// get user info
	ui, err := openPanClient.GetUserInfo()
	if err != nil {
		fmt.Println("get user info error")
		return
	}
	fmt.Println("当前登录用户：" + ui.Nickname)

	// do some file operation
	fi, _ := openPanClient.FileInfoByPath(ui.FileDriveId, "/我的文档")
	fmt.Println("\n我的文档 信息：")
	fmt.Println(fi)
```

## 网页版Web端接口
导入包
```
import "github.com/tickstep/aliyunpan-api/aliyunpan"
import "github.com/tickstep/aliyunpan-api/aliyunpan_web"
```

使用浏览器获取到的RefreshToken创建WebPanClient实例
```
	// get access token
	refreshToken := "f34b54eba1...706f389"
	webToken, err := aliyunpan_web.GetAccessTokenFromRefreshToken(refreshToken)
	if err != nil {
		fmt.Println("get acccess token error")
		return
	}
	
	// web pan client
	appConfig := aliyunpan_web.AppConfig{
		AppId: "25dzX3vbYqktVxyX",
		DeviceId: "T6ZJyY7JqX6EN2cDzLCxMVYZ",
		UserId:    "",
		Nonce:     0,
		PublicKey: "",
	}
	webPanClient := aliyunpan_web.NewWebPanClient(*webToken, aliyunpan_web.AppLoginToken{}, appConfig, aliyunpan_web.SessionConfig{
		DeviceName: "Chrome浏览器",
		ModelName:  "Windows网页版",
	})

	// create session
	webPanClient.CreateSession(&aliyunpan_web.CreateSessionParam{
		DeviceName: "Chrome浏览器",
		ModelName:  "Windows网页版",
	})

```

调用WebPanClient相关方法可以实现对阿里云盘的相关操作
```
	// get user info
	ui, err := webPanClient.GetUserInfo()
	if err != nil {
		fmt.Println("get user info error")
		return
	}
	fmt.Println("当前登录用户：" + ui.Nickname)

	// do some file operation
	fi, _ := webPanClient.FileInfoByPath(ui.FileDriveId, "/我的文档")
	fmt.Println("\n我的文档 信息：")
	fmt.Println(fi)
```

# 链接
> [tickstep/aliyunpan](https://github.com/tickstep/aliyunpan)   
> [阿里OpenAPI文档](https://www.yuque.com/aliyundrive/zpfszx/btw0tw)   