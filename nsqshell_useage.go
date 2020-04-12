package nsqshell

import (
	"fmt"
	"time"
)

func main() {
	done := make(chan interface{})
	closed := make(chan interface{})

	go StartNsqdInternal(done, closed, "")

	time.Sleep(10 * time.Second)
	close(done)

	<-closed
	fmt.Println("success close nsqd, main out")
}
