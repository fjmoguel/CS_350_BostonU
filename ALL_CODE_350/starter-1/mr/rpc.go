package mr

//
// RPC definitions.
// hi i am just checking the files
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

// example to show how to declare the arguments
// and reply for an RPC.
type ExampleArgs struct {
	X int
}
type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

// wow
type TaskKind int // this is to set 1 for map tasls and 0 for reduce ones to differenciate cases :)

const (
	KindMap    TaskKind = 0 // set 1 for map kinf of task
	KindReduce TaskKind = 1 // set 0 for the reduce task
	AllDone    TaskKind = 2 // exit program
	SleepABit  TaskKind = 3 // sleep
)

type TaskArgs struct{} // the start a worker line

// handling the reply part and calls from the wokrer with this struct
type TaskReply struct {
	Success      bool           // tells if we are done or not
	NumberOfReds int            // indicating number of tasks
	TypeOfTask   *MapReduceTask // pointer to elaborating the task
}

// now here is where we are going to be calling the rpc of the map amnd reduce tasks
type MapReduceTask struct {
	TaskID            int
	TaskKind          TaskKind
	InputFile         string
	TotalMapTasks     int
	IntermediateFiles []string
}

// we are taking the type of task, either 1 or 2, and the files we are working with so we know what to do and use the pointer of this struct to start the worker
//
// this segment is used by workers to notify the coordinator that a task has been completed succesfully with the help of the boolean
type DoneRequest struct {
	TaskID      int
	TaskKind    TaskKind
	OutputFiles []string
}

type WorkerArgs struct {
	TaskID   int
	TaskKind TaskKind
	NFiles   []string
}

// this struct is used to acknowledge a workers completion
type DoneReply struct {
	DoneRep bool
}

type RequestTaskArgs struct {
	TaskType  TaskKind
	TaskIndex int
	Filenames []string
}
type RequestTaskReply struct {
	TaskType  TaskKind
	TaskIndex int
	Filenames []string
	NReduce   int
}

type ReportTaskDoneArgs struct {
	TaskType              TaskKind
	TaskIndex             int
	IntermediateFilenames []string
}

type ReportTaskDoneReply struct {
	TaskType              TaskKind
	TaskIndex             int
	IntermediateFilenames []string
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}

// save progress here
