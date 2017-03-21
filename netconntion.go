package netengine

import (
	"bytes"
	//"fmt"
	"net"
	"sync/atomic"
	"time"
)

func (c *NetEngine) connectto(nettype, addr string) (int, error) {
	tcpaddr, err := net.ResolveTCPAddr(nettype, addr)
	if err != nil {
		return 0, err
	}

	con, err := net.DialTCP(nettype, nil, tcpaddr)
	if err != nil {
		return 0, err
	}

	r := c.add_conntion(con, default_buf_len, default_timeout, 1, 1)

	return r.ID, nil
}
func (c *NetEngine) add_conntion(con *net.TCPConn, maxBufLen, timeout, send, recv int32) *conntion {
	n := new(conntion)
	n.ID = c.get_id()
	n.Con = con
	n.MaxBufLen = maxBufLen
	n.Timeout = timeout
	n.RecvValid = recv
	n.SendValid = send
	n.SendChan = make(chan []byte, 128)
	n.IsStart = false

	c.conntion_list[n.ID] = n

	return n
}

func (c *NetEngine) set_conntion_buf(con *conntion, maxBuf int) {
	atomic.StoreInt32(&con.MaxBufLen, int32(maxBuf))
}
func (c *NetEngine) set_conntion_close_time(con *conntion, close_second int, send, recv bool) {
	var isend int32 = 0
	if send {
		isend = 1
	}
	var irecv int32 = 0
	if recv {
		irecv = 1
	}

	add := time.Second * time.Duration(close_second)
	atomic.StoreInt32(&con.Timeout, int32(close_second))
	atomic.StoreInt32(&con.SendValid, isend)
	atomic.StoreInt32(&con.RecvValid, irecv)
	newtime := time.Now().Add(add)
	if send {
		con.Con.SetWriteDeadline(newtime)
	} else {
		con.Con.SetWriteDeadline(time.Time{})
	}
	if recv {
		con.Con.SetReadDeadline(newtime)
	} else {
		con.Con.SetReadDeadline(time.Time{})
	}
}
func (c *NetEngine) start_conntion(con *conntion) {
	if con.IsStart {
		return
	}
	con.IsStart = true
	go c.conntion_run(con)
}
func (c *NetEngine) close_conntion(con *conntion) {
	con.Con.Close()
}
func (c *NetEngine) conntion_run(con *conntion) {
	go c.conntion_write(con)
	c.conntion_recv(con)

	c.notify.OnClosed(con.ID)

	c.del_conntion_chan <- con.ID
}
func (c *NetEngine) conntion_recv(con *conntion) {
	var buf bytes.Buffer
	net_con := con.Con
	defer net_con.Close()

	all_buf := make([]byte, 1024*5)
recv_loop:
	for {
		recv := atomic.LoadInt32(&con.RecvValid)
		timeout := atomic.LoadInt32(&con.Timeout)
		if recv != 0 {
			add := time.Second * time.Duration(timeout)
			net_con.SetReadDeadline(time.Now().Add(add))
		} else {
		}

		n, err := net_con.Read(all_buf)
		if err != nil {
			break
		}
		cur := all_buf[:n]
		buf.Write(cur)
		var recvd = buf.Bytes()

		for {
			r := c.notify.OnRecv(con.ID, recvd)
			if r < 0 {
				break recv_loop
			}
			if r > 0 {
				recvd = recvd[r:]
				buf.Reset()
				buf.Write(recvd)
			} else {
				break
			}
		}
	}
}
func (c *NetEngine) send_data(id int, data []byte) {
	con, ok := c.conntion_list[id]
	if !ok {
		return
	}
	con.SendChan <- data
}
func (c *NetEngine) conntion_write(con *conntion) {
	defer con.Con.Close()

	data_chan := make(chan []byte)
	defer close(data_chan)

	go conntion_write_net(con, data_chan)

	datalen := 0
	data := make([][]byte, 0, 100)

	is_closed := false
for_loop:
	for {
		maxBufLen := int(atomic.LoadInt32(&con.MaxBufLen))
		timer := time.NewTimer(time.Hour)
		if len(data) > 0 {
			s := data[0]
			select {
			case msg, ok := <-con.SendChan:
				if !ok {
					break for_loop
				}
				datalen += len(msg)
				data = append(data, msg)
				if datalen > maxBufLen {
					if !is_closed {
						is_closed = true
						c.notify.OnBufferLimit(con.ID)
						con.Con.Close()
					}
				}
			case data_chan <- s:
				data = data[1:]
				datalen = datalen - len(s)
			case <-timer.C:
				timer = nil
			}
		} else {
			select {
			case msg, ok := <-con.SendChan:
				if !ok {
					break for_loop
				}
				datalen += len(msg)
				data = append(data, msg)
				if datalen > maxBufLen {
					if !is_closed {
						is_closed = true
						c.notify.OnBufferLimit(con.ID)
						con.Con.Close()
					}
				}
			case <-timer.C:
				timer = nil
			}
		}
		if timer != nil && !timer.Stop() {
			<-timer.C
		}
	}
}
func conntion_write_net(con *conntion, data chan []byte) {
	netcon := con.Con
	defer netcon.Close()
	for d := range data {
		send := atomic.LoadInt32(&con.SendValid)
		if send != 0 {
			timeout := atomic.LoadInt32(&con.Timeout)
			add := time.Second * time.Duration(timeout)
			netcon.SetWriteDeadline(time.Now().Add(add))
		} else {
		}
		_, err := netcon.Write(d)
		if err != nil {
			break
		}
	}
}
