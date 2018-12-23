package netengine

import (
	"net"
	"time"
	"sync"
)

type NetEngine struct {
	conntion_list map[int]*conntion
	listener_list map[int]*listener

	lock sync.Locker
	id   int

	add_conntion_chan chan add_conntion_msg
	del_conntion_chan chan int
	stop_chan         chan stop_msg

	get_remote_addr_chan chan get_addr_msg
	get_local_addr_chan  chan get_addr_msg
	add_listen_chan      chan add_listen_msg
	start_chan           chan start_msg
	send_chan            chan send_msg
	get_sendfunc_chan    chan get_sendfunc_msg
	close_chan           chan close_msg
}

func (c *NetEngine) manage_run() {
	is_stop := false
	var stop_req stop_msg

	check_stop_finish := time.NewTicker(time.Second)
	defer check_stop_finish.Stop()

for_loop:
	for {
		select {
		case r := <-c.send_chan:
			con,ok := c.conntion_list[r.ID]
			if !ok {
				break
			}
			con.send_data(r.Data)

		case r := <-c.add_conntion_chan:
			var msg add_conntion_ret_msg
			new_con := c.add_conntion(r.Con, r.Notify, r.MaxBufLen, r.RecvBufLen, r.ReadTimeout, r.WriteTimeout)
			msg.ID, msg.err = new_con.ID, nil
			r.ch <- msg

		case r := <-c.del_conntion_chan:
			c.del_conntion(r)

		case r := <-c.add_listen_chan:
			var msg listen_ret_msg
			nlis := c.add_listen(r.Lis, r.Notify)
			msg.ID, msg.err = nlis.ID, nil
			r.ch <- msg

		case r := <-c.get_remote_addr_chan:
			addr, ok := c.get_remote_addr(r.ID)
			if ok {
				r.ch <- addr
			} else {
				r.ch <- nil
			}

		case r := <-c.get_local_addr_chan:
			addr, ok := c.get_local_addr(r.ID)
			if ok {
				r.ch <- addr
			} else {
				r.ch <- nil
			}

		case r := <-c.get_sendfunc_chan:
			f := c.get_send_func(r.ID)
			r.ch <- f
			
		case r := <-c.start_chan:
			c.start(r.ID)

		case r := <-c.close_chan:
			c.close(r.ID)

		case r := <-c.stop_chan:
			stop_req = r
			is_stop = true
			c.closeall()

		case <-check_stop_finish.C:
			if !is_stop {
				break
			}
			if len(c.conntion_list)+len(c.listener_list) <= 0 {
				break for_loop
			}

		}
	}
	stop_req.ch <- 0
}

func (c *NetEngine) get_id() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	r := c.id
	for {
		if r > 100*10000 {
			r = 1
		}
		if _, ok := c.conntion_list[r]; !ok {
			break
		}
		if _, ok := c.listener_list[r]; !ok {
			break
		}
		r = r + 1
	}
	c.id = r + 1
	return r
}

func (c *NetEngine) add_conntion(con net.Conn, notify NetNotify, maxBufLen,recvBufLen int, readTimeout,writeTimeout time.Duration) *conntion {
	n := new(conntion)
	n.ID = c.get_id()
	n.C = c
	n.Con = con
	n.Notify = notify
	n.MaxBufLen = maxBufLen
	n.RecvBufLen = recvBufLen
	n.ReadTimeout = readTimeout
	n.WriteTimeout = writeTimeout
	n.SendChan = make(chan []byte, 8)
	n.IsStart = false
	n.Send = n.get_send()

	c.conntion_list[n.ID] = n

	return n
}

func (c *NetEngine) add_listen(lis net.Listener, notify NetNotify) *listener {
	n := new(listener)
	n.ID = c.get_id()
	n.C = c
	n.Listen = lis
	n.Notify = notify
	
	n.IsStart = false

	c.listener_list[n.ID] = n

	return n
}


func (c *NetEngine) del_conntion(id int) {
	// 到这里的都是已经close的
	con, ok := c.conntion_list[id]
	if ok {
		close(con.SendChan)
		delete(c.conntion_list, id)
		return
	}
	delete(c.listener_list, id)
}

func (c *NetEngine) closeall() {
	listen_idlist := make([]int, 0)
	conntion_idlist := make([]int, 0)
	for _, v := range c.listener_list {
		v.Listen.Close()
		if !v.IsStart {
			listen_idlist = append(listen_idlist, v.ID)
		}
	}
	for _, v := range c.conntion_list {
		v.Con.Close()
		if !v.IsStart {
			conntion_idlist = append(conntion_idlist, v.ID)
		}
	}

	for _, v := range listen_idlist {
		delete(c.listener_list, v)
	}
	for _, v := range conntion_idlist {
		delete(c.conntion_list, v)
	}
}

func (c *NetEngine) get_remote_addr(id int) (addr net.Addr, r bool) {
	addr = nil
	r = false

	_, ok := c.listener_list[id]
	if ok {
		return nil, false
	}
	con, ok := c.conntion_list[id]
	if ok {
		return con.Con.RemoteAddr(), true
	}
	return
}

func (c *NetEngine) get_local_addr(id int) (addr net.Addr, r bool) {
	addr = nil
	r = false

	lis, ok := c.listener_list[id]
	if ok {
		return lis.Listen.Addr(), true
	}
	con, ok := c.conntion_list[id]
	if ok {
		return con.Con.RemoteAddr(), true
	}
	return
}

func (c *NetEngine) start(id int) {
	lis, ok := c.listener_list[id]
	if ok {
		lis.start()
	}
	con, ok := c.conntion_list[id]
	if ok {
		con.start()
	}
}

func (c *NetEngine) close(id int) {
	lis, ok := c.listener_list[id]
	if ok {
		lis.close()
	}
	con, ok := c.conntion_list[id]
	if ok {
		con.close()
	}
}

func (c *NetEngine) get_send_func(id int) SendFunc {
	con, ok := c.conntion_list[id]
	if ok {
		if con.Send == nil {
			panic("send func error")
		}
		return con.Send
	}
	return nil
}
