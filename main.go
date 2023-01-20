package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

const (
	cmdBind       = 0x01
	socks5Version = 0x05
	atypeIPV4     = 0x01
	atypeHOST     = 0x03

	AUTH_NO_REQUIRED = 0x00
	AUTH_UNAME_PWD   = 0x02
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
	if err := auth(conn); err != nil {
		log.Printf("[auth failed in %v] err=%v", conn.RemoteAddr(), err)
	}
	log.Printf("[auth success]--------------------")
	if err := connect(conn); err != nil {
		log.Printf("[connect failed in %v] err=%v", conn.RemoteAddr(), err)
	}
	log.Printf("[connect success]--------------------")
	//和真正的(ip,port)建立连接，双向转发数据

}

func auth(conn net.Conn) error {
	// conn其实就是个reader 这一行是为了让bufio接管reader，相当于是readerProxy
	reader := bufio.NewReader(conn)
	buf := make([]byte, 2)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return fmt.Errorf("[auth failed1] err=%w", err) //Go1.13版本为fmt.Errorf函数新加了一个%w占位符用来生成一个可以包裹Error的Wrapping Error
	}
	log.Printf("[auth result] version=%v", buf)
	// 仅支持socks5版本
	if buf[0] != socks5Version {
		return fmt.Errorf("[auth failed] version byte %v not support", int(buf[0]))
	}
	// 读取浏览器支持的认证方法
	method := make([]byte, int(buf[1]))
	if _, err := io.ReadFull(reader, method); err != nil {
		return fmt.Errorf("[auth failed2] err=%w", err)
	}
	log.Printf("[auth result] method=%v", method)
	// 仅支持 无鉴权方式
	if buf[1] != 0x01 || !bytes.Equal(method, []byte{AUTH_NO_REQUIRED}) {
		return fmt.Errorf("[auth failed] method length %v or byte %v not support", buf[1], method)
	}
	// 别忘了认证阶段结束之后，需要给浏览器一个回复
	if _, err := conn.Write([]byte{socks5Version, AUTH_NO_REQUIRED}); err != nil {
		return fmt.Errorf("[auth failed3] err=%w", err)
	}
	return nil
}

// 为了简化代码，就不做参数校验了
func connect(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	buf := make([]byte, 4)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return fmt.Errorf("[connect failed1] err=%w", err)
	}
	// cmd只支持connection请求，即让代理服务器和目标服务器建立连接
	ver, cmd, rsv, atype := buf[0], buf[1], buf[2], buf[3]
	log.Println("[connect result]", ver, cmd, rsv, atype)

	var addr string
	if atype == atypeIPV4 {
		//复用这个buf 读取ipv4地址
		if _, err := io.ReadFull(reader, buf); err != nil {
			return fmt.Errorf("[connect failed3] err=%w", err)
		}
		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
		log.Printf("[connect result] atype=ipv4 addr=%v", addr)
	} else if atype == atypeHOST { // 如果是域名的话，先读一个字节len，再读len个字节域名
		sz, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("[connect failed4] err=%w", err)
		}
		domain := make([]byte, sz)
		if _, err := io.ReadFull(reader, domain); err != nil {
			return fmt.Errorf("[connect failed5] err=%w", err)
		}
		addr = string(domain)
		log.Printf("[connect result] atype=host size=%v addr=%v", sz, addr)
	} else {
		return fmt.Errorf("[connect failed6] invalid atype [%v]", atype)
	}

	//读端口
	if _, err := io.ReadFull(reader, buf[:2]); err != nil {
		return fmt.Errorf("[connect failed7] err=%w", err)
	}
	port := binary.BigEndian.Uint16(buf[:2]) // byte转int的时候都需要这样

	log.Printf("[connect result] target addr=%v, port=%v, buf=%v 小端%v", addr, port, buf, binary.LittleEndian.Uint16(buf[:2]))

	//先和目标服务器建立好连接 再返回给浏览器connect成功
	dest, err := net.Dial("tcp", fmt.Sprintf("%v:%v", addr, port))
	if err != nil {
		return fmt.Errorf("dial dst failed:%w", err)
	}
	defer dest.Close()

	//由于BND.ADDR和BND.PORT 对于connect这种cmd 是可选字段，所以我们直接填0即可。
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return fmt.Errorf("[connect failed8] err=%w", err)
	}

	//relay转发阶段
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 这句没用 但多次cancel没关系 是幂等的

	go func() {
		_, _ = io.Copy(dest, reader)
		cancel()
	}()
	go func() {
		_, _ = io.Copy(conn, dest)
		cancel()
	}()

	<-ctx.Done()
	return nil
}
