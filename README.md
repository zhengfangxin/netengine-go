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
	use send to send data get more performance
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
neten.Init()
lis, err := net.Listen("tcp", "127.0.0.1:9000")
id, err := neten.AddListen(lis, &sernotify)
neten.Start(id)


OnAccepted:
id,err := neten.AddConnection(conn, &sernotify, ...)
neten.Start(id)

```
## client
```
neten := new(NetEngine)
neten.Init()
conn, err := net.Dial("tcp", "127.0.0.1:9000")
id,err := neten.AddConnection(conn, &clinotify, ...)
neten.Start(id)
// send is asynchronous
neten.Send(id, data)
neten.Close(id)
```
