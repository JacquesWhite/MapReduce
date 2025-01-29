package mr

import (
	"fmt"
	"os"
	"sort"
)

type WorkerContext struct {
	MasterIP   string
	MasterPort string
	MapFunc    func(string, string) []KeyValue
	ReduceFunc func(string, []string) string
}

func WorkerMain(context WorkerContext) {

	var intermediate []KeyValue
	content, err := os.ReadFile("../datasets/test.txt")
	if err != nil {
		return
	}

	kva := context.MapFunc("../datasets/test.txt", string(content))
	intermediate = append(intermediate, kva...)

	sort.Sort(ByKey(intermediate))

	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		var values []string
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := context.ReduceFunc(intermediate[i].Key, values)
		// this is the correct format for each line of Reduce output.
		// please do not change it.
		fmt.Println(intermediate[i].Key, output)
		i = j
	}

	for {
		x := 1 + 2
		x = x - 1
		// connect to master
		// wait for messages
		// handle the messages and do the work

		// upon receiving a message from the master
		// which will contain the file name
		// open it and pass the contents to the Map function

		//fmt.Println("Worker")
	}
}
