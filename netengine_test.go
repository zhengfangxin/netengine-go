package netengine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"testing"
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

var server *NetEngine
var listenid_count map[int]int
var id_listenid map[int]int
var server_chan chan servermsg

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

type clientmsg struct {
	add bool
	id  int
}

var client *NetEngine
var client_list map[int]int
var client_chan chan clientmsg
var client_len chan int

func client_run() {
	datalen := 0
	count := 0
	last := time.Now()
	for {
		timer := time.NewTimer(time.Second * 3)
		select {
		case d, ok := <-client_chan:
			if !ok {
				return
			}
			if d.add {
				id := d.id
				client_list[id] = id
			} else {
				id := d.id
				delete(client_list, id)
			}
		case d, ok := <-client_len:
			if !ok {
				return
			}
			datalen = datalen + d
			count = count + 1
			cur := time.Now()
			sub := cur.Sub(last)
			if sub > time.Second*3 {
				last = cur
				fmt.Println("client", len(client_list), datalen, count)
				datalen = 0
				count = 1
			}
		case <-timer.C:
			timer = nil
			fmt.Println("client", len(client_list), datalen)
			datalen = 0
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

type clientnotify struct {
}

func (c *clientnotify) OnAcceptBefore(listenid int, addr net.Addr) bool {
	fmt.Printf("accept before listenid:%d addr:%s\n", listenid, addr)
	return true
}
func (c *clientnotify) OnAccept(listenid int, id int, addr net.Addr) {
	fmt.Printf("accepted listenid:%d netid:%d addr:%s\n", listenid, id, addr)
}
func (c *clientnotify) OnRecv(id int, data []byte) int {
	//fmt.Printf("recv data id:%d len:%d\n", id, len(data))
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
		fmt.Println("flag error", flag, pro_flag)
	}
	all_len := int(packlen) + headlen
	if datalen < all_len {
		return 0
	}

	client_len <- all_len
	return all_len
}
func (c *clientnotify) OnClosed(id int) {
	msg := clientmsg{false, id}
	client_chan <- msg
	//fmt.Printf("on closed id:%d\n", id)
}
func (c *clientnotify) OnBufferLimit(id int) {
	fmt.Printf("on buffer limit id:%d\n", id)
}

func TestServer(t *testing.T) {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	server = new(NetEngine)
	var sernotify servernotify

	server.Init(&sernotify)

	listenid_count = make(map[int]int)
	id_listenid = make(map[int]int)
	server_chan = make(chan servermsg, 1024)

	go server_run()

	add_server(server, t, "tcp", "127.0.0.1:9000")
	add_server(server, t, "tcp", "127.0.0.1:9001")

	client = new(NetEngine)
	var clinotify clientnotify
	client.Init(&clinotify)

	client_list = make(map[int]int)
	client_chan = make(chan clientmsg, 1024)
	client_len = make(chan int, 10000)

	go client_run()

	for i := 0; i < 1000; i++ {
		if i%2 == 0 {
			//add_client(client, t, "tcp", "127.0.0.1:9000")
		}
		add_client(client, t, "tcp", "127.0.0.1:9001")
	}

	go send_run(client, t)

	for {
		time.Sleep(time.Second)
	}
}

func add_server(neten *NetEngine, t *testing.T, nettype, addr string) {
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

func add_client(neten *NetEngine, t *testing.T, nettype, addr string) {
	//fmt.Printf("connect to:%s addr:%s\n", nettype, addr)

	id, err := neten.ConnectTo(nettype, addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	//neten.SetBuffer(id, 10001)
	//neten.SetCloseTime(id, 100, true, false)
	msg := clientmsg{true, id}
	client_chan <- msg
	neten.Start(id)
}

func send_req(neten *NetEngine, id int, data []byte) {
	var buf bytes.Buffer
	datalen := int16(len(data))
	binary.Write(&buf, binary.LittleEndian, datalen)
	var flag uint8 = pro_flag
	binary.Write(&buf, binary.LittleEndian, flag)
	buf.Write(data)
	neten.Send(id, buf.Bytes())
}
func send_run(neten *NetEngine, t *testing.T) {
	count := 1
	for {
		for _, v := range client_list {
			n := rand.Intn(1024)
			data := make([]byte, n)
			send_req(neten, v, data)
		}
		time.Sleep(time.Second)
		count = count + 1
		/*if count > 5 {
			fmt.Println("close")
			for _, v := range client_list {
				neten.Close(v)
			}

			break
		}*/
	}
}
