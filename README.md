# toy_socks5
实现一个基础的不鉴权的socks5代理服务器。

## 步骤

包括协议解析，鉴权认证，代理服务器和目标服务器建立连接，流量转发双向通信。

## 效果

让tcp流量都通过代理服务器转发出去，并且在代理服务器输出目标域名+端口。


## 运行与测试

### v1版本

#### 运行

> go run main.go

#### 测试
> nc 127.0.0.1 9000

或者

> go run client.go
 
### 最终版本

#### 运行

> go run main.go

#### 测试

> curl --socks5 127.0.0.1:9000 -v http://www.baidu.com

或者 

使用浏览器插件