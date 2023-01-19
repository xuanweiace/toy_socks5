package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	server, err := net.Listen("tcp", "127.0.0.1:9000")
	log.Printf("服务端已启动，监听 127.0.0.1:9000")
	if err != nil {
		panic(err)
	}
	for {
		if cli, err := server.Accept(); err == nil {
			log.Printf("收到客户端 %v 的连接", cli.RemoteAddr())
			go process(cli)
		} else {
			log.Printf("accept failed %v", err)
		}

	}
}

// 处理这个conn，这个连接的生命周期等价于函数的生命周期，所以别忘记关掉连接
func process(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn) // 基于连接创建一个只读的带缓冲的流
	for {
		b := make([]byte, 10)
		if n, err := reader.Read(b); err == nil {
			fmt.Println("字节数：", n)
			fmt.Println(b)
			conn.Write(b)
		} else {
			log.Printf("process failed: %v", err)
			break
		}

		// 由于reader是一个带缓冲的流，所以这里看似是读一个字节，其实是每次从内核拷贝defaultBufSize = 4096个字节的数据。详见源码
		//if b, err := reader.ReadByte();err == nil {
		//	fmt.Println("读取字节:", b)
		//	conn.Write([]byte{b})
		//} else {
		//	log.Printf("process failed: %v", err)
		//	break
		//}
	}
}
