package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"time"
)

var boolLogging = false

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// mr-main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	args := RequestTaskArgs{}
	reply := RequestTaskReply{}
	for {
		if boolLogging {
			log.Printf("2")
		}
		reply = RequestTask(&args)
		var intermediateFilenames []string
		switch reply.TaskType {
		case MapTask:
			if boolLogging {
				log.Printf("Time to work with a %d and %v", reply.TaskType, reply.Filenames)
			}
			if len(reply.Filenames) > 0 {
				if boolLogging {
					log.Println("Do we ever get here?")
				}
				intermediateFilenames = doMap(reply.TaskIndex, reply.Filenames[0], mapf, reply.NReduce)
			} else {
				log.Printf("but then this runs forever")
				if boolLogging {
					log.Println("Received a MapTask with no filenames")
				}
				continue
			}

		case ReduceTask:
			if boolLogging {
				log.Println("Do we ever get here v.7?")
			}
			if boolLogging {
				log.Println("Do we ever get here v.4?")
			}
			if len(reply.Filenames) > 0 {
				if boolLogging {
					log.Println("Do we ever get here v.2?")
				}
				doReduce(reply.TaskIndex, reply.Filenames, reducef)
			} else {
				if boolLogging {
					log.Println("Do we ever get here v.3?")
					log.Println("Received a ReduceTask with no filenames")
				}
				continue
			}
		case AllDone:
			return
		case SleepABit:
			time.Sleep(500 * time.Millisecond)
			args = RequestTaskArgs{}
			reply = RequestTaskReply{}
			continue
		default:
			if boolLogging {
				fmt.Println("Invalid task type received from coordinator")
			}
			continue
		}

		reportArgs := ReportTaskDoneArgs{
			TaskType:              reply.TaskType,
			TaskIndex:             reply.TaskIndex,
			IntermediateFilenames: intermediateFilenames,
		}
		reportReply := ReportTaskDoneReply{}
		call("Coordinator.ReportTaskDone", &reportArgs, &reportReply)
		if reportReply.TaskType == AllDone {
			log.Printf("End")
			return
		}
	}

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

func doMap(taskIndex int, filename string, mapf func(string, string) []KeyValue, nReduce int) []string {
	file, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	content := string(file)
	kva := mapf(filename, content)

	for i := 0; i < nReduce; i++ {
		tmpFileName := fmt.Sprintf("mr-%d-%d.json", taskIndex, i)
		file, err := os.Create(tmpFileName)
		if err != nil {
			log.Fatalf("cannot create temp file")
		}
		defer file.Close()

		enc := json.NewEncoder(file)
		for _, kv := range kva {
			if ihash(kv.Key)%nReduce == i {
				err := enc.Encode(&kv)
				if err != nil {
					log.Fatalf("cannot encode kv pair")
				}
			}
		}
	}
	var intermediateFilenames []string
	for i := 0; i < nReduce; i++ {
		intermediateFilenames = append(intermediateFilenames, fmt.Sprintf("mr-%d-%d.json", taskIndex, i))
	}
	return intermediateFilenames
}

func doReduce(taskIndex int, filenames []string, reducef func(string, []string) string) {
	kvMap := make(map[string][]string)
	if boolLogging {
		log.Println("Are we here now?")
	}
	for _, filename := range filenames {
		file, err := os.Open(filename)
		if err != nil {
			if boolLogging {
				log.Println("hi")
			}
			log.Fatalf("cannot open")
		}
		defer file.Close()

		dec := json.NewDecoder(file)
		var kv KeyValue
		for dec.Decode(&kv) == nil {
			kvMap[kv.Key] = append(kvMap[kv.Key], kv.Value)
		}
	}

	oname := fmt.Sprintf("mr-out-%d", taskIndex)
	ofile, err := os.Create(oname)
	if err != nil {
		log.Fatalf("cannot create output file")
	}
	defer ofile.Close()

	for key, values := range kvMap {
		output := reducef(key, values)
		fmt.Fprintf(ofile, "%v %v\n", key, output)

	}
}

func RequestTask(args *RequestTaskArgs) RequestTaskReply {
	reply := RequestTaskReply{}
	ret := call("Coordinator.RequestTask", args, &reply)
	if !ret {
		log.Printf("call failed!\n")
		os.Exit(0)
	}
	return reply
}

// example function to show how to make an RPC call to the coordinator.
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	call("Coordinator.Example", &args, &reply)

	// reply.Y should be 100.
	fmt.Printf("reply.Y %v\n", reply.Y)
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
// DO NOT MODIFY
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println("Unable to Call", rpcname, "- Got error:", err)
	return false
}

// hi
// save progress here
