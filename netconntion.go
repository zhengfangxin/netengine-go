package netengine

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

func (c *NetEngine) connectto(nettype, addr string, notify NetNotify) (int, error) {
	tcpaddr, err := net.ResolveTCPAddr(nettype, addr)
	if err != nil {
		return 0, err
	}

	con, err := net.DialTCP(nettype, nil, tcpaddr)
	if err != nil {
		return 0, err
	}

	r := c.add_conntion(con, notify, default_max_buf_len, default_read_buf_len, 0, 0)

	return r.ID, nil
}
func get_send(con *conntion) SendFunc {
	r := func(d []byte) error {
		netcon := con.Con
		
		writeTimeout := con.WriteTimeout
		fmt.Println("write timeout", writeTimeout)

		if writeTimeout != 0 {
			netcon.SetWriteDeadline(time.Now().Add(writeTimeout))
		}

		for {
			n, err := netcon.Write(d)
			if err != nil {
				netcon.Close()
				return err
			}
			if n == len(d) {
				break
			} else {
				fmt.Println("write return", n, len(d))
				d = d[n:]
			}
		}
		return nil
	}
	return r
}
func (c *NetEngine) add_conntion(con *net.TCPConn, notify NetNotify, maxBufLen,recvBufLen int32, readTimeout,writeTimeout time.Duration) *conntion {
	n := new(conntion)
	n.ID = c.get_id()
	n.Con = con
	n.Notify = notify
	n.MaxBufLen = maxBufLen
	n.RecvBufLen = recvBufLen
	n.ReadTimeout = readTimeout
	n.WriteTimeout = writeTimeout
	n.SendChan = make(chan []byte, 8)
	n.IsStart = false
	n.Send = get_send(n)

	c.conntion_list[n.ID] = n

	return n
}

func (c *NetEngine) set_conntion_buf(con *conntion, maxBuf,recvBuf int) {
	atomic.StoreInt32(&con.MaxBufLen, int32(maxBuf))
	atomic.StoreInt32(&con.RecvBufLen, int32(recvBuf))
}
func (c *NetEngine) set_conntion_timeout(con *conntion, readTimeout,writeTimeout time.Duration) {
	con.ReadTimeout = readTimeout
	con.WriteTimeout = writeTimeout
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

	con.Notify.OnClosed(con.ID)

	c.del_conntion_chan <- con.ID
}

func (c *NetEngine) conntion_recv(con *conntion) {
	net_con := con.Con
	defer net_con.Close()

	send_fun := con.Send

	buflen := int(con.RecvBufLen)
	all_buf := make([]byte, buflen)	
	
	notify := con.Notify

	valid_begin_pos := 0
	valid_end_pos := 0
recv_loop:
	for {
		readTimeout := con.ReadTimeout
		if readTimeout != 0 {
			net_con.SetReadDeadline(time.Now().Add(readTimeout))
		}

		if valid_end_pos >= len(all_buf) {
			if valid_begin_pos > 2048 {
				copy(all_buf, all_buf[valid_begin_pos:valid_end_pos])
			} else {
				all_len := len(all_buf)
				org_buf := all_buf
				all_buf = make([]byte, all_len*2)
				copy(all_buf, org_buf[valid_begin_pos:valid_end_pos])
			}
			valid_end_pos = valid_end_pos - valid_begin_pos
			valid_begin_pos = 0
		}

		currecv := all_buf[valid_end_pos:len(all_buf)]
		n, err := net_con.Read(currecv)
		if err != nil {
			break
		}
		valid_end_pos = valid_end_pos + n

		for {
			curdata := all_buf[valid_begin_pos:valid_end_pos]
			r := notify.OnRecv(con.ID, curdata, send_fun)

			if r < 0 {
				break recv_loop
			}
			if r > 0 {
				if r > len(curdata) {
					break recv_loop
				}
				valid_begin_pos = valid_begin_pos + r
			} else {
				break
			}
		}
		if valid_begin_pos == valid_end_pos {
			valid_begin_pos = 0
			valid_end_pos = 0
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

	notify := con.Notify

	datalen := 0
	data := make([][]byte, 0, 1)
	maxBufLen := int(con.MaxBufLen)

for_loop:
	for {		
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
					notify.OnBufferLimit(con.ID)
					break for_loop
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
					notify.OnBufferLimit(con.ID)
					break for_loop
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
		writeTimeout := con.WriteTimeout
		fmt.Println("write timeout", writeTimeout)

		if writeTimeout != 0 {
			netcon.SetWriteDeadline(time.Now().Add(writeTimeout))
		}

		err := con.Send(d)
		if err != nil {
			break
		}
	}
}
