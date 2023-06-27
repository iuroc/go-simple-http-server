package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/url"
	"os"
	"path/filepath"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer listener.Close()
	log.Println("http://127.0.0.1:8080")
	for {
		client, err := listener.Accept()
		if err != nil {
			log.Println(err.Error())
		}
		go handleClient(client)
	}
}

// 处理客户端请求
func handleClient(client net.Conn) {
	request := getRequest(client)
	serveDir := "file"
	path, err := url.QueryUnescape(request.path)
	if err == nil {
		path = serveDir + path
	}
	header := make(map[string]string)
	var body bytes.Buffer
	var code int
	if _, err := os.Stat(path); err == nil {
		if content, err := ioutil.ReadFile(path); err == nil {
			code = 200
			contentType := mime.TypeByExtension(filepath.Ext(path))
			header["Content-Type"] = contentType
			body.Write(content)
		} else {
			code = 502
			body.WriteString("文件读取错误")
		}
	} else if os.IsNotExist(err) {
		code = 404
		body.WriteString("文件未找到")
	} else {
		code = 502
		body.WriteString("系统错误")
	}
	message := makeResponse(header, code, body.Bytes())
	client.Write(message)
	log.Println(code, request.path)
}

// 生成响应消息
func makeResponse(header map[string]string, code int, body []byte) []byte {
	response := bytes.Buffer{}
	response.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", code, statusText(code)))
	defaultHeader := make(map[string]string)
	defaultHeader["Content-Length"] = fmt.Sprint(len(body))
	defaultHeader["Content-Type"] = "text/html; charset=utf-8"
	defaultHeader["Connection"] = "close"
	for key, value := range header {
		if value != "" {
			defaultHeader[key] = value
		}
	}
	for key, value := range defaultHeader {
		response.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	response.WriteString("\r\n")
	response.Write(body)
	return response.Bytes()
}



// 解析消息为请求对象
func parseRequest(message []byte) requestType {
	request := requestType{}
	parts := bytes.SplitN(message, []byte("\r\n\r\n"), 2)
	if len(parts) > 0 {
		headerLines := bytes.Split(parts[0], []byte("\r\n"))
		if len(headerLines) > 0 {
			first := bytes.Split(headerLines[0], []byte(" "))
			if len(first) > 1 {
				request.method = string(first[0])
				request.path = string(first[1])
			}
		}
		if len(headerLines) > 1 {
			header := make(map[string]string)
			for _, line := range headerLines[1:] {
				kv := bytes.SplitN(line, []byte(":"), 2)
				key := bytes.TrimSpace(kv[0])
				value := bytes.TrimSpace(kv[1])
				header[string(key)] = string(value)
			}
			request.header = header
		}
	}
	if len(parts) > 1 {
		request.body = string(parts[1])
	}
	return request
}

// 获取请求对象
func getRequest(client net.Conn) requestType {
	message := getMessage(client)
	return parseRequest(message)
}

// 请求对象类
type requestType struct {
	path   string
	method string
	header map[string]string
	body   string
}

// 获取消息内容
func getMessage(client net.Conn) []byte {
	message := make([]byte, 1024)
	client.Read(message)
	return message
}

// 获取状态码对应的状态文本
func statusText(code int) string {
	text := make(map[int]string)
	text[200] = "OK"
	text[404] = "Not Found"
	return text[code]
}
