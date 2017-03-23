# netengine-go
netengine in golang

## NetNotify
```
type SendFunc func(data []byte) error
type NetNotify interface {
	// return false to close
	// 同一个listenid 会在一个goroute中调用
	OnAcceptBefore(listenid int, addr net.Addr) bool
	OnAccept(listenid int, id int, addr net.Addr)

	/* return -1 to close
	return >0 to consume data
	return 0 need more data
	use send go send data get more performance
	data is valid only in OnRecv, so careful to use it,
	*/
	OnRecv(id int, data []byte, send SendFunc) int

	// OnClosed,OnBufferLimit,OnAccept不会在同一个goroute中调用
	OnClosed(id int)
	// write buffer limit, OnClosed always will call
	OnBufferLimit(id int)
}
```

## server
```
neten := new(NetEngine)
neten.Init(notify)
id,err := neten.Listen("tcp", "127.0.0.1:9000")
neten.Start(id)
```
## client
```
neten := new(NetEngine)
neten.Init(notify)
id,err := neten.ConnectTo("tcp", "127.0.0.1:9000")
// set max write buffer len, close when over
neten.SetBuffer(id, 1024*1024)
// set max read write idle time(second), you can control write or read who can update the valid time
neten.SetCloseTime(id, 30, true, true)
neten.Start(id)
// send is asynchronous
neten.Send(id, data)
neten.Close(id)
```
