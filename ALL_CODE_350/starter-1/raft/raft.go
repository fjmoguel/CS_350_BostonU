package raft

// colab:  Sofia Boada

//
// This is an outline of the API that raft must expose to
// the service (or tester). See comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   Create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   Start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   Each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester) in the same server.
//

import (
	"bytes"
	"cs350/labgob"
	"cs350/labrpc"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// As each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). Set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // This peer's index into peers[]
	dead      int32               // Set by Kill()

	// Your data here (4A, 4B).
	// 4A so far
	currentTerm     int
	votedFor        int
	role            Role
	electionTimeout time.Duration
	lastHeartbeat   time.Duration

	//adding 4B now
	log         []LogEntry // entries
	commitIndex int
	nextIndex   []int
	matchIndex  []int
	lastApplied int

	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// timers
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer

	// channels
	applyCh chan ApplyMsg
	Commit  chan struct{}

	// log entries
	logPrevIndex int
	logPrevTerm  int
}

// constats regarding the role of a server
type Role int

const (
	Follower = iota
	Candidate
	Leader
)

// log entry struct
type LogEntry struct {
	Command interface{}
	Term    int
}

// Return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock() // Lock to ensure exclusive access to the state.
	defer rf.mu.Unlock()
	var term int
	var isleader bool
	// Your code here (4A).
	term = rf.currentTerm
	isleader = rf.role == Leader

	return term, isleader
}

// Save Raft's persistent state to stable storage, where it
// can later be retrieved after a crash and restart. See paper's
// Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (4B).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

// Restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (4B).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var logs []LogEntry
	if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&logs) != nil {
		fmt.Println("Read persist fail ")
		log.Fatal("quit")
	} else {
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.log = logs
		rf.updateLastLogInfo()
	}
}

func (rf *Raft) updateLastLogInfo() {
	lastIndex := len(rf.log) - 1
	// log.Println("last index set to %d, lastIndex")
	if lastIndex >= 0 {
		rf.logPrevIndex = lastIndex
		rf.logPrevTerm = rf.log[lastIndex].Term
	} else {
		rf.logPrevIndex = -1
		rf.logPrevTerm = -1
	}
}

// Example RequestVote RPC arguments structure.
// Field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (4A, 4B).
	// for 4A
	Term        int
	CandidateId int
	// 4B part
	LastLogIndex int
	LastLogTerm  int
}

// Example RequestVote RPC reply structure.
// Field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (4A).
	Term        int
	VoteGranted bool
}

// arguments that will be sent
type AppendEntriesArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

// this is the RPC result reply as seen in the fgure of the paper
type AppendEntriesReply struct {
	Term     int
	Success  bool
	NewTerm  int
	NewIndex int
}

// Example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (4A, 4B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	termComp := args.Term < rf.currentTerm
	switch {
	case termComp:
		// older candiate term
		reply.Term = rf.currentTerm
		// fmt.Printf("current reply term is: %d\n", reply.Term)
		// fmt.Printf("no vote since terms dont match, fail)
		reply.VoteGranted = false
		return
	case args.Term == rf.currentTerm:
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		if rf.role == Leader {
			return
		} else if rf.votedFor == args.CandidateId {
			reply.VoteGranted = true
			// fmt.Printf("vote granted to server")
			return
		} else if rf.votedFor != -1 && rf.votedFor != args.CandidateId {
			return
		}

	default:
		// the arg term is behind the currrent term carried by the peers
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		rf.currentTerm = args.Term
		rf.votedFor = -1
		rf.role = Follower
		// fmt.Printf("Updates to follower after terms mistmatch")
		rf.updateLastLogInfo()
		if rf.logPrevTerm > args.LastLogTerm {
			rf.persist()
			return
		}

		if rf.logPrevTerm == args.LastLogTerm && rf.logPrevIndex > args.LastLogIndex {
			rf.persist()
			return
		}

		rf.currentTerm = args.Term
		rf.votedFor = args.CandidateId
		rf.role = Follower
		rf.resetElectionTimer()
		reply.VoteGranted = true
		rf.persist()
		return
	}
}

// timer to reset
func (rf *Raft) resetElectionTimer() {
	rf.electionTimer.Stop()
	rf.electionTimer.Reset(rf.electionTimeout + (time.Duration(rand.Int63()) % rf.electionTimeout))
}

// timer to reset heartbeats
func (rf *Raft) resetHeartbeatTimer() {
	rf.heartbeatTimer.Stop()
	rf.heartbeatTimer.Reset(rf.lastHeartbeat)
}

// Example code to send a RequestVote RPC to a server.
// Server is the index of the target server in rf.peers[].
// Expects RPC arguments in args. Fills in *reply with RPC reply,
// so caller should pass &reply.
//
// The types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// Look at the comments in ../labrpc/labrpc.go for more details.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// The service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. If this
// server isn't the leader, returns false. Otherwise start the
// agreement and return immediately. There is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. Even if the Raft instance has been killed,
// this function should return gracefully.
//
// The first return value is the index that the command will appear at
// if it's ever committed. The second return value is the current
// term. The third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (4B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	term = int(rf.currentTerm)
	isLeader = rf.role == Leader
	lastIndex := len(rf.log) - 1
	index = lastIndex + 1

	if !isLeader {
		return index, term, false
	}

	if isLeader {
		rf.log = append(rf.log, LogEntry{
			Term:    int(rf.currentTerm),
			Command: command,
		})
		rf.matchIndex[rf.me] = index
		rf.persist()
	}

	return index, term, isLeader
}

// The tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. Your code can use killed() to
// check whether Kill() has been called. The use of atomic avoids the
// need for a lock.
//
// The issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. Any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for !rf.killed() {

		// fmt.Println("Ticker started")
		// Your code here to check if a leader election should be started
		select {
		case <-rf.heartbeatTimer.C:
			rf.resetHeartbeatTimer() // its a leder so it starts to send heartbests to all peers until a timeout happens
			rf.requestHeartbeats()
		case <-rf.electionTimer.C:
			// fmt.Printf("Election Timer expired. Role: %s\n", []string{"Follower", "Candidate", "Leader"}[rf.role])
			if rf.shouldStartElection() {
				rf.resetElectionTimer() // follower it starts an election once we run out of timeout for the electiopn
				// fmt.Println("Starting election")
				go rf.election()
			}
		}
	}
}

func (rf *Raft) requestHeartbeats() {
	// see if its a leder that can start to sendd heartbeats to the main
	for index := range rf.peers { // loop peers to see which ones are responding until we reach us
		if rf.me == index {
			continue
		}
		rf.mu.Lock()
		if rf.role == Leader { // once we reach the leader
			next := rf.nextIndex[index]
			term := rf.currentTerm
			rf.mu.Unlock()
			// fmt.Printf("Sending to go function heartbeat to peer %d. Next Index: %d, Term: %d\n", index, next, term)
			go rf.heartbeat(index, next, term) // leader sends heartbesats to the rest
		} else {
			rf.mu.Unlock()
			continue
		}
	}
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// append entries method
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// updsate for a rf server its log information
	rf.updateLastLogInfo()
	rf.initializeAppendEntriesResponse(reply) // make structs that are releveant to the heartbeats
	if !rf.checkTermWithLeader(args, reply) { // do comparison with the leader
		return
	}
	rf.updateLastLogInfo()                // uodate log info again for previous index and term so we can start and intilaize
	if !rf.lookConsistency(args, reply) { // check for the log consistency
		return
	}
	// fmt.Printf("AppendEntries functions called with term: %d, leaderID: %d\n", args.Term, args.LeaderID)
	if rf.appendEntriesIfLogMatches(args, reply) { // append actually once we know all log entries are in the same page
		rf.updateCommitIndexAndNotify(args)
	}
}

func (rf *Raft) updateCommitIndexAndNotify(args *AppendEntriesArgs) {
	// make commit and tell commmit log to send messagws tp the main server of tyeh raft via the apply channel
	if rf.commitIndex < args.LeaderCommit {
		newCommitIndex := args.LeaderCommit
		if newCommitIndex > len(rf.log) {
			newCommitIndex = len(rf.log)
		}
		// fmt.Printf("Updated the commit index. now we have the commit index as: %d, Leader commit: %d\n", rf.commitIndex, args.LeaderCommit)
		rf.commitIndex = newCommitIndex // commit
		// fmt.Printf("sending strict to the commit log function")
		rf.Commit <- struct{}{} // tell main server to send messages, we are all in the same log index
	}
	// fmt.Printf("calling persist")
	rf.persist()
}

func (rf *Raft) appendEntriesIfLogMatches(args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	// if all logs are matching, we can append via this function
	rf.updateLastLogInfo()
	// fmt.Printf("Log updated")
	// fmt.Printf("Checking if log matches. PrevLogIndex: %d, PrevLogTerm: %d\n", args.PrevLogIndex, args.PrevLogTerm)
	if args.PrevLogIndex > rf.logPrevIndex { // fir instabce to check a mismatch
		// see if the index match failed, so we upodate but reply false
		reply.Success = false
		reply.NewIndex = rf.logPrevIndex
		reply.NewTerm = rf.log[rf.logPrevIndex].Term
		// fmt.Printf("Conflict found at index %d, NewTerm: %d\n", index, rf.log[index].Term)
		return false
	}
	if rf.log[args.PrevLogIndex].Term == args.PrevLogTerm { // the real check that we need
		rf.log = append(rf.log[:args.PrevLogIndex+1], args.Entries...) // see if the index match append new log entrues
		return true
	}
	reply.Success = false
	index := rf.findConflictIndex(args.PrevLogIndex) // check if any logs are left behind, checlk for all types of updates
	reply.NewIndex = index
	reply.NewTerm = rf.log[index].Term
	// fmt.Printf("Conflict found at index %d, NewTerm: %d\n", index, rf.log[index].Term)
	return false
}

func (rf *Raft) findConflictIndex(startIdx int) int {
	index := startIdx
	// fmt.Printf("Finding conflict index starting from %d\n", startIdx)                                                      // see if the index match
	for index > rf.commitIndex && rf.log[index].Term == rf.log[startIdx].Term { // if someone is not in the same page, update
		index -= 1
	}
	return index
}

func (rf *Raft) lookConsistency(args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	// check for log consistcny amongst servers
	if len(args.Entries) > 0 {
		// fmt.Printf("Checking consistency of entries amongst servers. Number of entries: %d\n", len(args.Entries))
		lastEntryTerm := args.Entries[len(args.Entries)-1].Term
		if rf.logPrevTerm > lastEntryTerm || (rf.logPrevTerm == lastEntryTerm && rf.logPrevIndex > args.PrevLogIndex+len(args.Entries)) {
			reply.Success = false
			reply.Term = -1
			return false
		}
	}
	return true
}

func (rf *Raft) checkTermWithLeader(args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	// checl that leader and servers aer in teh same term so it can carry the updated logs
	if rf.currentTerm < args.Term {
		rf.becomeFollower(args.Term)
		return true
	} else if rf.currentTerm > args.Term {
		// fmt.Println("Term inconsistency detected. Term or index mismatch, figure 8 fail")
		reply.Success = false
		return false
	}
	return true
}

func (rf *Raft) becomeFollower(term int) {
	rf.role = Follower // set rf to follower since no election was won by them
	rf.votedFor = -1
	rf.currentTerm = term // update term again after we make a server a follower
}

func (rf *Raft) initializeAppendEntriesResponse(reply *AppendEntriesReply) {
	// make a new append entries struct
	reply.Success = true
	reply.Term = rf.currentTerm
	reply.NewIndex = rf.logPrevIndex
	reply.NewTerm = rf.logPrevTerm
	rf.resetElectionTimer()
}

func (rf *Raft) requestAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) shouldStartElection() bool {
	rf.mu.Lock()
	defer rf.mu.Unlock() // check if the server is NOT leader so it can carry out an alection
	return rf.role != Leader
}

func (rf *Raft) election() {
	// log.Println("pass 1")
	// starting the election like the paper says in (§5.2)
	rf.mu.Lock()
	// new election start
	rf.resetElectionTimer()
	rf.role = Candidate // makwe instance a candidate whilst it casts the votes of its peers
	rf.votedFor = rf.me
	rf.currentTerm++
	rf.updateLastLogInfo()
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: rf.logPrevIndex,
		LastLogTerm:  rf.logPrevTerm,
	}
	rf.mu.Unlock()
	// constants to start an election
	majority := len(rf.peers) / 2
	var VoteCount int = 1
	var VotesGranted int = 1
	// loop all values to get RPC calls
	for i := range rf.peers {
		// log.Println("pass 2")
		if i != rf.me {
			var reply RequestVoteReply
			server := i
			// reauest vote from peer tghats a follwer AND not the same as the candidate server
			rf.sendRequestVote(server, &args, &reply)
			if reply.VoteGranted { // we got a vote for the elected candidate server
				VotesGranted += 1
			}
			VoteCount += 1              // bnumber of servers that have voted
			if reply.Term > args.Term { // if teh rerms are the same chabgfe to followr for  peer server
				rf.mu.Lock()
				if rf.currentTerm < reply.Term {
					rf.currentTerm = reply.Term
					rf.role = Follower
					rf.votedFor = -1
					rf.resetElectionTimer()
					rf.persist()
				}
				rf.mu.Unlock()
			}
			if int(VotesGranted) > majority { // check if we hae reacg a less majoroty vote to rteun false, no election won
				break
			}
			if int(VoteCount)-int(VotesGranted) > majority {
				break
			}
		}
	}
	if VotesGranted <= majority { // elrction won!
		return
	}
	rf.mu.Lock()
	rf.updateLastLogInfo() // updwte newest election term
	if rf.currentTerm == args.Term && rf.role == Candidate {
		rf.role = Leader
		rf.nextIndex = make([]int, len(rf.peers))
		for i := 0; i < len(rf.peers); i++ { // update the log index for all servers that are not the leader and have tp be reploicated
			rf.nextIndex[i] = rf.logPrevIndex + 1
		}
		rf.matchIndex = make([]int, len(rf.peers))
		rf.matchIndex[rf.me] = rf.logPrevIndex // match index ot a server to the nect index one per the election
	}
	rf.mu.Unlock()
}

func (rf *Raft) heartbeat(server int, next int, term int) {
	// leader sends heartbeats until someone stops redpomods or we time out, or systems killed from outside
	for !rf.killed() {
		rf.mu.Lock()
		if rf.role != Leader || rf.currentTerm != term { //. check if its still the leader
			rf.mu.Unlock()
			return
		}
		args := rf.initiateAppendEntries(next) // append entries for a new log entry
		reply := AppendEntriesReply{}
		rf.mu.Unlock()

		rf.requestAppendEntries(server, &args, &reply) // reauest to append

		if !rf.handleAppendEntriesResponse(&reply, &args, server, &next, term) { // actually append
			return
		}
	}
}

func (rf *Raft) initiateAppendEntries(next int) AppendEntriesArgs {
	args := AppendEntriesArgs{
		// make the arfuments to check if we can append the new log entries
		Term:         rf.currentTerm,
		LeaderID:     rf.me,
		LeaderCommit: rf.commitIndex,
		Entries:      rf.log[next:],
		PrevLogTerm:  rf.log[next-1].Term,
		PrevLogIndex: next - 1,
	}
	return args
}

func (rf *Raft) handleAppendEntriesResponse(reply *AppendEntriesReply, args *AppendEntriesArgs, server int, next *int, term int) bool {
	if !reply.Success && reply.Term == -1 {
		return false
	}
	rf.mu.Lock()
	// fmt.Printf("Handling AppendEntries response from server %d. Success: %t, Term: %d\n", server, reply.Success, reply.Term)
	defer rf.mu.Unlock()
	if reply.Term > term {
		// fmt.Println("Updating current term and resetting election timer")
		// make struct for the entries
		if rf.currentTerm < reply.Term {
			rf.currentTerm = reply.Term
			rf.role = Follower
			rf.votedFor = -1
			rf.resetElectionTimer()
			rf.persist()
		}
		return false
		// omce they are followers we return false since there is no leader
	} else if reply.Success {
		// fmt.Println("we got a reply")
		rf.updateSuccessIndexes(server, next, len(args.Entries))
		rf.attemptCommitLogs(args)
		// makw commit
		return false
	} else {
		*next = rf.adjustNextBasedOnReply(reply)
		return true
	}
}

func (rf *Raft) updateSuccessIndexes(server int, next *int, entryLen int) {
	// update the index of the main server and its followrers
	rf.nextIndex[server] = *next + entryLen
	rf.matchIndex[server] = *next - 1 + entryLen
}

func (rf *Raft) attemptCommitLogs(args *AppendEntriesArgs) {
	if len(args.Entries) > 0 && args.Entries[len(args.Entries)-1].Term == rf.currentTerm { // the log have been made all teg entrues
		rf.updateCommitIndex() // make a commit
	}
}

func (rf *Raft) updateCommitIndex() {
	var goCommit bool
	for i := rf.commitIndex + 1; i <= len(rf.log); i++ {
		if rf.isMajorityReplicated(i) { // once again check its time to update
			// fmt.Println("most servers are replicasted to leader")
			rf.commitIndex = i
			goCommit = true // reasdy tp start commit since the majorirty of servers are in the same index
		} else {
			break
		}
	}
	if goCommit {
		// ready to make a rf commit
		// fmt.Println("send to commit log")
		rf.Commit <- struct{}{} // send a commit to the new state
	}
}

func (rf *Raft) isMajorityReplicated(index int) bool {
	count := 0
	majority := len(rf.peers) / 2 // declaring majority to cast the votes later
	for _, matchIndex := range rf.matchIndex {
		if matchIndex >= index { // see if te match index of rf matches the new one
			// fmt.Println("index match, increaisng count")
			count++
			if count > majority { // see if the peers have been replicated just liek in figure 8 so we can send the messages now
				return true
			}
		}
	}
	return false
}

func (rf *Raft) adjustNextBasedOnReply(reply *AppendEntriesReply) int {
	if rf.log[reply.NewIndex].Term == reply.NewTerm {
		return reply.NewIndex + 1 // check index
	} else {
		return reply.NewIndex // send the new index so we can move on and make a commit
	}
}

func (rf *Raft) CommitLog() {
	for !rf.killed() {
		<-rf.Commit // leader will change term in the figure 8 to commit messages
		rf.applyCommits()
	}
}

func (rf *Raft) applyCommits() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// if the lst log applied is less than teh actual commit index, we go
	if rf.lastApplied < rf.commitIndex {
		msgs := rf.prepareMessages() //  make a message struct to commit index
		rf.applyMessages(msgs)       // send messages when ready
	}
}

func (rf *Raft) prepareMessages() []ApplyMsg {
	msgs := make([]ApplyMsg, 0, rf.commitIndex-rf.lastApplied)
	for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
		// append the message to the channel and send
		msgs = append(msgs, ApplyMsg{
			CommandValid: true,
			Command:      rf.log[i].Command,
			CommandIndex: i,
		})
	}
	return msgs
}

func (rf *Raft) applyMessages(msgs []ApplyMsg) {
	for _, msg := range msgs {
		rf.applyCh <- msg
		// fmt.Println("messages sent")
		rf.lastApplied = msg.CommandIndex // send messages tobteh main state
	}
}

// The service or tester wants to create a Raft server. The ports
// of all the Raft servers (including this one) are in peers[]. This
// server's port is peers[me]. All the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int, persister *Persister, applyCh chan ApplyMsg) *Raft {
	// Your initialization code here (4A, 4B).
	rf := &Raft{
		peers:           peers,
		persister:       persister,
		me:              me,
		currentTerm:     0,
		votedFor:        -1,
		role:            Follower,
		electionTimeout: time.Millisecond * 150,
	}
	rf.lastHeartbeat = time.Millisecond * 100 // set a benchmark for the heartneat timer
	rf.applyCh = applyCh
	rf.peers = peers
	rf.persister = persister // im tired start a pesister state
	rf.me = me
	rf.log = make([]LogEntry, 1)
	// initialize from state persisted before a crash.
	rf.readPersist(persister.ReadRaftState())
	// final initialize
	rf.Commit = make(chan struct{}, 100)
	rf.electionTimer = time.NewTimer(rf.electionTimeout + (time.Duration(rand.Int63()) % rf.electionTimeout))
	rf.heartbeatTimer = time.NewTimer(rf.lastHeartbeat)
	// start ticker goroutine to start elections.
	go rf.ticker()
	go rf.CommitLog()
	// END
	// im tired
	return rf
}
