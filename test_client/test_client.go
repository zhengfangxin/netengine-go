package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	netengine "github.com/zhengfangxin/netengine-go"
	"math/rand"
	"net"
	"time"
)

const pro_flag = 0xef

type clientmsg struct {
	add bool
	id  int
}

var neten *netengine.NetEngine
var client_list map[int]int
var client_chan chan clientmsg
var client_len chan int
var client_send_ch chan int
var clinotify clientnotify

const conntion_count = 2

func main() {

	neten = new(netengine.NetEngine)
	
	neten.Init()

	client_list = make(map[int]int)
	client_chan = make(chan clientmsg, 128)
	client_len = make(chan int, 128)
	client_send_ch = make(chan int)

	go client_run()

	for i := 0; i < conntion_count/2; i++ {
		add_client("tcp", "127.0.0.1:9000")
		add_client("tcp", "127.0.0.1:9001")
	}

	go send_run()

	/*time.Sleep(time.Second*10)
	fmt.Println("net stop ..")
	neten.Stop()
	fmt.Println("net stop suc")*/

	for {
		time.Sleep(time.Second * 5)
	}
}

func add_client(nettype, addr string) {
	fmt.Printf("connect to:%s addr:%s\n", nettype, addr)

	conn, err := net.Dial(nettype, addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	readTimeout := time.Second*0
	writeTimeout := time.Second*0
	id,err := neten.AddConnection(conn, &clinotify, 5*1024, 1024*1024, readTimeout, writeTimeout)
	if err != nil {
		fmt.Println("add conn", err)
		return
	}

	msg := clientmsg{true, id}
	client_chan <- msg
	neten.Start(id)
}

func send_req(id int, data []byte, send netengine.SendFunc) {
	var buf bytes.Buffer

	datalen := int16(len(data))
	binary.Write(&buf, binary.LittleEndian, datalen)

	var flag uint8 = pro_flag
	binary.Write(&buf, binary.LittleEndian, flag)

	buf.Write(data)

	send_d := buf.Bytes()
	if send == nil {
		neten.Send(id, send_d)
	} else {
		send(send_d)
	}
}
func send_run() {
	time.Sleep(time.Second * 3)
	for i := 0; i < 5; i++ {
		n := rand.Intn(1024)
		client_send_ch <- n

		fmt.Println("send", n)
		time.Sleep(time.Second)
	}
}

func client_run() {

	datalen := 0
	count := 0
	last := time.Now()
	for {
		timer := time.NewTimer(time.Second * 3)
		select {
		case d := <-client_send_ch:
			data := make([]byte, d)
			for _, v := range client_list {
				send_req(v, data, nil)
			}
		case d, ok := <-client_chan:
			if !ok {
				return
			}
			if d.add {
				id := d.id
				client_list[id] = id
				
				/*if len(client_list) >= conntion_count {
					fmt.Println("start send", len(client_list))
					data := make([]byte, 100)
					for _, v := range client_list {
						send := neten.GetSendFunc(v)
						if send == nil {
							fmt.Println("get nil send func", v)
							continue
						}
						send_req(neten, v, data, send)
					}
				}*/
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
				fmt.Println("neten", len(client_list), datalen/3, count/3)
				datalen = 0
				count = 0
			}
		case <-timer.C:
			timer = nil
			fmt.Println("neten", len(client_list), datalen/3, datalen/3)
			datalen = 0
		}
		if timer != nil && !timer.Stop() {
			<-timer.C
		}
	}
}

type clientnotify struct {
}

func (c *clientnotify) OnAccepted(listenid int, conn net.Conn) {
	addr := conn.RemoteAddr()
	fmt.Printf("accepted listenid:%d netid:%d addr:%s\n", listenid, addr)
}
func (c *clientnotify) OnRecv(id int, data []byte, send netengine.SendFunc) int {
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

	fmt.Println("recv pack", packlen)

	client_len <- all_len

	return all_len
}
func (c *clientnotify) OnClosed(id int) {
	msg := clientmsg{false, id}
	client_chan <- msg
	fmt.Printf("on closed id:%d\n", id)
}
func (c *clientnotify) OnBufferLimit(id int) {
	fmt.Printf("on buffer limit id:%d\n", id)
}
