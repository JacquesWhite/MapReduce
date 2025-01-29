package mr

import (
	"fmt"
)

func HelloFromWorker() int {
	fmt.Println("Hello from worker")
	return 0
}
