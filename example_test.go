package scratch

import (
	"encoding/json"
	"fmt"
)

func Example() {
	pool := NewPool(128)
	buf := pool.Get()
	defer pool.Put(buf)

	usrID := uint16('U')
	msgID := uint16('M')
	key := buf.
		AppendString("messages").
		AppendByte('/').
		PutUint16(usrID).
		AppendByte('/').
		PutUint16(msgID).
		UnsafeString()

	json.NewEncoder(buf).Encode(struct{ Msg string }{"Hello, World!"})
	msg := buf.Bytes()[len(key):]
	fmt.Printf("key=%q\nmsg=%s\n", key, msg)
	// Output:
	// key="messages/\x00U/\x00M"
	// msg={"Msg":"Hello, World!"}
}
