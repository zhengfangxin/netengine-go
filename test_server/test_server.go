package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	netengine "github.com/zhengfangxin/netengine-go"
	"net"
	"runtime"
	"time"
)

const pro_flag = 0xef

type servernotify struct {
}
type servermsg struct {
	add      bool
	id       int
	listenid int
}

var server *netengine.NetEngine
var listenid_count map[int]int
var id_listenid map[int]int
var server_chan chan servermsg
var sernotify servernotify

func main() {
	server = new(netengine.NetEngine)

	server.Init()

	listenid_count = make(map[int]int)
	id_listenid = make(map[int]int)
	server_chan = make(chan servermsg, 1024)

	go server_run()

	add_server(server, "tcp", "127.0.0.1:9000")
	add_server(server, "tcp", "127.0.0.1:9001")

	for {
		time.Sleep(time.Second * 5)
	}
}

func add_server(neten *netengine.NetEngine, nettype, addr string) {
	fmt.Printf("listen on:%s addr:%s\n", nettype, addr)

	id, err := neten.Listen(nettype, addr, &sernotify)
	if err != nil {
		fmt.Println(err)
		return
	}

	neten.SetBuffer(id, 5*1024*1024, 10*1024)
	neten.SetTimeout(id, time.Second*10, time.Hour)

	neten.Start(id)	
}

func server_run() {
	for {
		timer := time.NewTimer(time.Second * 3)
		select {
		case d, ok := <-server_chan:
			if !ok {
				return
			}
			if d.add {
				listenid := d.listenid
				if _, ok := listenid_count[listenid]; !ok {
					listenid_count[listenid] = 0
				}
				listenid_count[listenid] = listenid_count[listenid] + 1
				id_listenid[d.id] = listenid
			} else {
				id := d.id
				listenid, ok := id_listenid[id]
				if !ok {
					panic("del id")
				}
				listenid_count[listenid] = listenid_count[listenid] - 1
				delete(id_listenid, id)
			}
		case <-timer.C:
			timer = nil
			fmt.Println(listenid_count)
			fmt.Println("goroute", runtime.NumGoroutine())
		}
		if timer != nil && !timer.Stop() {
			<-timer.C
		}
	}
}

func (c *servernotify) OnAcceptBefore(listenid int, addr net.Addr) bool {
	fmt.Printf("accept before listenid:%d addr:%s\n", listenid, addr)
	return true
}
func (c *servernotify) OnAccept(listenid int, id int, addr net.Addr) {
	fmt.Printf("accepted listenid:%d netid:%d addr:%s\n", listenid, id, addr)
	msg := servermsg{true, id, listenid}
	server_chan <- msg
}
func (c *servernotify) OnRecv(id int, data []byte, send netengine.SendFunc) int {
	fmt.Printf("recv data id:%d len:%d\n", id, len(data))
	datalen := len(data)
	const headlen = 3
	if datalen < headlen {
		return 0
	}
	buf := bytes.NewBuffer(data)
	var packlen int16
	var flag uint8
	err := binary.Read(buf, binary.LittleEndian, &packlen)
	if err != nil {
		fmt.Println("read", err)
	}
	err = binary.Read(buf, binary.LittleEndian, &flag)
	if err != nil {
		fmt.Println("read", err)
	}
	if flag != pro_flag {
		fmt.Println("recv flag error", flag, pro_flag)
		panic("flag error")
	}
	all_len := int(packlen) + headlen
	if datalen < all_len {
		return 0
	}

	send_d := data[:all_len]

	// 使用下面两种方式发送数据
	//server.Send(id, send_d)
	send(send_d)

	return all_len
}
func (c *servernotify) OnClosed(id int) {
	fmt.Printf("on closed id:%d\n", id)
	msg := servermsg{false, id, 0}
	server_chan <- msg
}
func (c *servernotify) OnBufferLimit(id int) {
	fmt.Printf("on buffer limit id:%d\n", id)
}
