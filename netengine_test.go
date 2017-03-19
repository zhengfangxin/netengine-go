package netengine

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"
)

type mynotify struct {
}

var client_list map[int]int

func (c *mynotify) OnAcceptBefore(listenid int, addr net.Addr) bool {
	fmt.Printf("accept before listenid:%d addr:%s\n", listenid, addr)
	return true
}
func (c *mynotify) OnAccept(listenid int, id int, addr net.Addr) {
	fmt.Printf("accepted listenid:%d netid:%d addr:%s\n", listenid, id, addr)
}
func (c *mynotify) OnRecv(id int, data []byte) int {
	fmt.Printf("recv data id:%d len:%d\n", id, len(data))
	return len(data)
}
func (c *mynotify) OnClosed(id int) {
	fmt.Printf("on closed id:%d\n", id)
}
func (c *mynotify) OnBufferLimit(id int) {
	fmt.Printf("on buffer limit id:%d\n", id)
}

func TestServer(t *testing.T) {
	neten := new(NetEngine)
	var notify mynotify
	neten.Init(&notify)

	client_list = make(map[int]int)

	server(neten, t, "tcp", "127.0.0.1:9000")
	server(neten, t, "tcp", "127.0.0.1:9001")

	for i := 0; i < 1; i++ {
		client(neten, t, "tcp", "127.0.0.1:9000")
		client(neten, t, "tcp", "127.0.0.1:9001")
	}
	go send_run(neten, t)
	for {
		time.Sleep(time.Second)
	}
}

func server(neten *NetEngine, t *testing.T, nettype, addr string) {
	fmt.Printf("listen on:%s addr:%s\n", nettype, addr)

	id, err := neten.Listen(nettype, addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	neten.Start(id)
}

func client(neten *NetEngine, t *testing.T, nettype, addr string) {
	//fmt.Printf("connect to:%s addr:%s\n", nettype, addr)

	id, err := neten.ConnectTo(nettype, addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	client_list[id] = id
	neten.Start(id)
}

func send_req(neten *NetEngine, id int, data []byte) {

}
func send_run(neten *NetEngine, t *testing.T) {
	for {
		for _, v := range client_list {
			n := rand.Intn(1024)
			data := make([]byte, n)
			send_req(neten, v, data)
		}
		time.Sleep(time.Second)
	}
}
