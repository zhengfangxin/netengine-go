package main

import (
	"fmt"
	"github.com/zhengfangxin/netengine"
	//"log"
	"net"
	//"net/http"
	//_ "net/http/pprof"
	"runtime"
	//"runtime/debug"
	"time"
)

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

func main() {
	go func() {
		//http.ListenAndServe("localhost:6060", nil)
	}()

	server = new(netengine.NetEngine)
	var sernotify servernotify

	server.Init(&sernotify)

	listenid_count = make(map[int]int)
	id_listenid = make(map[int]int)
	server_chan = make(chan servermsg, 1024)

	go server_run()

	add_server(server, "tcp", "127.0.0.1:9000")
	add_server(server, "tcp", "127.0.0.1:9001")

	for {
		time.Sleep(time.Second * 5)
		//runtime.GC()
		//debug.FreeOSMemory()
	}
}

func add_server(neten *netengine.NetEngine, nettype, addr string) {
	fmt.Printf("listen on:%s addr:%s\n", nettype, addr)

	id, err := neten.Listen(nettype, addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	neten.Start(id)
	//neten.SetBuffer(id, 10000)
	//neten.SetCloseTime(id, 100, true, false)
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
	//fmt.Printf("accept before listenid:%d addr:%s\n", listenid, addr)
	return true
}
func (c *servernotify) OnAccept(listenid int, id int, addr net.Addr) {
	//fmt.Printf("accepted listenid:%d netid:%d addr:%s\n", listenid, id, addr)
	msg := servermsg{true, id, listenid}
	server_chan <- msg
}
func (c *servernotify) OnRecv(id int, data []byte) int {
	//fmt.Printf("recv data id:%d len:%d\n", id, len(data))
	server.Send(id, data)
	return len(data)
}
func (c *servernotify) OnClosed(id int) {
	//fmt.Printf("on closed id:%d\n", id)
	msg := servermsg{false, id, 0}
	server_chan <- msg
}
func (c *servernotify) OnBufferLimit(id int) {
	fmt.Printf("on buffer limit id:%d\n", id)
}