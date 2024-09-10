package mr

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type Coordinator struct {
	mapTasks    []TypeAndTasks
	reduceTasks []TypeAndTasks
	nReduce     int
	mapDone     bool
	reduceDone  bool
	mu          sync.Mutex
}

// Your code here -- RPC handlers for the worker to call.
// type of task
const (
	MapTask    = 0
	ReduceTask = 1
)

// now the task status
const (
	Unassigned = 3
	InProgress = 4
	Completed  = 5
)

type TypeAndTasks struct {
	Idx          int       // The int for the task based on the const
	AllFilenames []string  // all inout nput files for map, intermediary files for reduce
	Status       int       // the task status
	Type         int       // The Type of Task (Map or Reduce)
	StartTime    time.Time // start timer to indicate and check for timeouts
}

func (c *Coordinator) RequestTask(args *RequestTaskArgs, reply *RequestTaskReply) error {
	if boolLogging {
		log.Printf("!")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// adding the map tasks to the array in the struct
	if !c.allMapTasksDone() {
		timeoutTime := time.Now().Add(-10 * time.Second)
		for i := 0; i < len(c.mapTasks); i++ {
			if (c.mapTasks[i].Status == Unassigned) && !c.allMapTasksDone() && c.mapTasks[i].StartTime.Before(timeoutTime) {
				c.mapTasks[i].Status = InProgress
				c.mapTasks[i].StartTime = time.Now()
				reply.TaskType = MapTask
				reply.TaskIndex = i
				reply.Filenames = c.mapTasks[i].AllFilenames
				reply.NReduce = c.nReduce
				if boolLogging {
					log.Printf("Assigned MapTask %d to worker, intermediate files: %v\n", reply.TaskIndex, reply.Filenames)
					log.Printf("Time to work with %d", reply.TaskType)
				}
				return nil
			}
		}
		reply.TaskType = SleepABit
	}
	if c.allMapTasksDone() && !c.mapDone {
		if boolLogging {
			log.Println("wow")
			log.Println("Is this here?")
		}
		c.mapDone = true
		if boolLogging {
			log.Println("All map tasks completed, proceeding to reduce tasks.")
		}
	}

	if c.allMapTasksDone() && !c.allReduceTasksDone() { // check the args and how it returns the reply !!!
		if !boolLogging {
			reply.TaskType = ReduceTask
		}
		timeoutTime := time.Now().Add(-10 * time.Second)
		for i := 0; i < len(c.reduceTasks); i++ {
			if boolLogging {
				log.Println("Are we even assigning?")
			}
			if c.reduceTasks[i].Status == Unassigned && c.reduceTasks[i].StartTime.Before(timeoutTime) && !c.allReduceTasksDone() {
				c.reduceTasks[i].Status = InProgress
				c.reduceTasks[i].StartTime = time.Now()

				reply.TaskType = ReduceTask
				reply.TaskIndex = i
				reply.Filenames = c.reduceTasks[i].AllFilenames
				reply.NReduce = c.nReduce
				if boolLogging {
					log.Printf("Assigned ReduceTask %d to worker, intermediate files: %v\n", reply.TaskIndex, reply.Filenames)
					log.Printf("Time to work with a %v\n", reply.TaskType)
				}
				return nil
			}
		}
		reply.TaskType = SleepABit
	}
	reply.TaskType = AllDone
	if boolLogging {
		log.Println("No task assigned, all tasks in progress or completed.")
	}
	return nil
}

func (c *Coordinator) ReportTaskDone(args *ReportTaskDoneArgs, reply *ReportTaskDoneReply) error {
	if boolLogging {
		log.Println("hey (we are reporting tasks)")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.allMapTasksDone() && c.allReduceTasksDone() {
		reply.TaskType = AllDone
		return nil
	}
	if args.TaskType == MapTask {
		if boolLogging {
			log.Println("hey (valid map task)")
		}
		c.mapTasks[args.TaskIndex].Status = Completed
		if boolLogging {
			log.Printf("Map task completed")
		}
		for _, filename := range args.IntermediateFilenames {
			if boolLogging {
				log.Println("hey (time to range all files)")
			}
			reduceTaskIndex := extractReduceTaskIndex(filename)
			c.reduceTasks[reduceTaskIndex].AllFilenames = append(c.reduceTasks[reduceTaskIndex].AllFilenames, filename)
		}
	} else if args.TaskType == ReduceTask {
		if boolLogging {
			log.Println("hey (reduce )")
		}
		c.reduceTasks[args.TaskIndex].Status = Completed
		if boolLogging {
			log.Printf("reduce task completed")
		}
	} else {
		if boolLogging {
			log.Printf("Invalid task type reported by worker")
		}
		return errors.New("invalid task type")
	}

	return nil
}

// an example RPC handler.
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// mr-main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	c.mu.Lock()
	defer c.mu.Unlock()

	mapTasksDone := true
	reduceTasksDone := true

	for _, task := range c.mapTasks {
		switch task.Status {
		case Completed:
			continue
		default:
			mapTasksDone = false
			return false
		}
	}

	if mapTasksDone {
		for _, task := range c.reduceTasks {
			switch task.Status {
			case Completed:
				continue
			default:
				reduceTasksDone = false
				return false
			}
		}

	} else {
		return false
	}

	ret = mapTasksDone && reduceTasksDone

	if ret {
		if boolLogging {
			log.Println("All tasks completed, MapReduce job done.")
		}
	}

	return ret
}

// helper method for replies
func (c *Coordinator) allMapTasksDone() bool {
	if boolLogging {
		log.Println(len(c.mapTasks))
		log.Println("We are printing this endlessly")
	}
	for _, task := range c.mapTasks {
		if task.Status != Completed {
			return false
		}
	}
	if boolLogging {
		log.Printf("Map Tasks Finished")
	}
	return true
}

func (c *Coordinator) allReduceTasksDone() bool {
	if boolLogging {
		log.Println(len(c.reduceTasks))
		log.Println("We are printing this endlessly")
	}
	for _, task := range c.reduceTasks {
		if task.Status != Completed {
			return false
		}
	}
	if boolLogging {
		log.Printf("Reduce Tasks Finished")
	}
	return true
}

// create a Coordinator.
// mr-main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{

		mapTasks:    make([]TypeAndTasks, len(files)),
		reduceTasks: make([]TypeAndTasks, nReduce),
		nReduce:     nReduce,
		mapDone:     false,
		reduceDone:  false,
	}

	for i, file := range files {
		if boolLogging {
			log.Printf("Assigning file %s to map task %d\n", file, i)
		}
		c.mapTasks[i] = TypeAndTasks{
			Idx:          i,
			AllFilenames: []string{file},
			Status:       Unassigned,
			Type:         MapTask,
		}
	}

	for i := range c.reduceTasks {
		if boolLogging {
			log.Printf("Assigning task to reduce task %d\n", i)
		}
		c.reduceTasks[i] = TypeAndTasks{
			Idx:    i,
			Status: Unassigned,
			Type:   ReduceTask,
		}
	}

	c.server()
	return &c
}

// start a thread that listens for RPCs from worker.go
// DO NOT MODIFY
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func extractReduceTaskIndex(filename string) int {
	var x, y int
	_, err := fmt.Sscanf(filename, "mr-%d-%d", &x, &y)
	if err != nil {
		log.Fatalf("Failed to parse filename")
	}
	return y
}

// save progress here
