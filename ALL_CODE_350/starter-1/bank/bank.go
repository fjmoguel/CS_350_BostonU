package bank

import (
	"log"
	"sync"
	"fmt"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

// Bank represents a simple banking system
type Bank struct {
	bankLock *sync.Mutex
	accounts map[int]*Account
}

type Account struct {
	balance int
	lock    *sync.Mutex
}

// initializes a new bank
func BankInit() *Bank {
	b := Bank{&sync.Mutex{}, make(map[int]*Account)}
	return &b
}

func (b *Bank) notifyAccountHolder(accountID int) {
	b.sendEmailTo(accountID)
}

func (b *Bank) notifySupportTeam(accountID int) {
	b.sendEmailTo(accountID)
}

func (b *Bank) logInsufficientBalanceEvent(accountID int) {
	DPrintf("Insufficient balance in account %d for withdrawal\n", accountID)
}

func (b *Bank) sendEmailTo(accountID int) {
	// Dummy function
	// hello
	// marker
	// eraser
}

// creates a new account with the given account ID
func (b *Bank) CreateAccount(accountID int) {
	b.bankLock.Lock() // mutex so when the data changes we get concurrency
	// adding statment that will check if the account id is existant
	_, exists := b.accounts[accountID]
	// true or false cases
    if !exists {
        b.accounts[accountID] = &Account{ 
			balance: 0, // setting it balance to 0
            lock:    &sync.Mutex{},
		}
			b.bankLock.Unlock()  // unlock the mutex after
	}
	if exists{
		b.bankLock.Unlock()
		panic("Account already exists")
		 // do nothing, account exists and just print a message
			}
	}

// deposit a given amount to the specified account
func (b *Bank) Deposit(accountID, amount int) {
	// adding statment that will check if the account id is existant
	b.bankLock.Lock()
	_, exists := b.accounts[accountID]
	account := b.accounts[accountID]
	// true or false cases with if
    if exists{
	b.bankLock.Unlock()
	account.lock.Lock()
	DPrintf("[ACQUIRED LOCK][DEPOSIT] for account %d\n", accountID)

	newBalance := account.balance + amount
	account.balance = newBalance
	DPrintf("Deposited %d into account %d. New balance: %d\n", amount, accountID, newBalance)

	account.lock.Unlock()
	DPrintf("[RELEASED LOCK][DEPOSIT] for account %d\n", accountID)
	}else {
		b.bankLock.Unlock()
		fmt.Println("Account does not not exists")
			return // account does not exist, return nothing and print message
	}
}

// withdraw the given amount from the given account id
func (b *Bank) Withdraw(accountID, amount int) bool {
	b.bankLock.Lock()
	account := b.accounts[accountID]
	_, exists := b.accounts[accountID]
	// true or false cases with if once again
    if exists{
	b.bankLock.Unlock()
	account.lock.Lock()
	DPrintf("[ACQUIRED LOCK][WITHDRAW] for account %d\n", accountID)

	if account.balance >= amount {
		newBalance := account.balance - amount
		if (account.balance < 0){
			DPrintf("Account balance can't drop below 0")
			return false
		}
		account.balance = newBalance
		DPrintf("Withdrawn %d from account %d. New balance: %d\n", amount, accountID, newBalance)
		account.lock.Unlock()
		DPrintf("[RELEASED LOCK][WITHDRAW] for account %d\n", accountID)
		return true
	} else {
		// Insufficient balance in account %d for withdrawal
		// Please contact the account holder or take appropriate action.
		// trigger a notification or alert mechanism
		b.notifyAccountHolder(accountID)
		b.notifySupportTeam(accountID)
		account.lock.Unlock()
		// log the event for further investigation
		b.logInsufficientBalanceEvent(accountID)
		return false
	}
	}else{
		b.bankLock.Unlock()
		account.lock.Unlock()
		fmt.Println("Account does not not exists")
			return false// account does not exist, return nothing and print message
	}
}

// transfer amount from sender to receiver
func (b *Bank) Transfer(sender int, receiver int, amount int, allowOverdraw bool) bool {
	if sender != receiver{
	b.bankLock.Lock()
	senderAccount := b.accounts[sender]
	receiverAccount := b.accounts[receiver]
	_, exists := b.accounts[sender]
   	_, exists2 := b.accounts[receiver]
	if exists && exists2{
	b.bankLock.Unlock()

	// using ordered locks to prevend a deadlock
		var firstAccount, secondAccount *Account
		order := sender < receiver

		switch {
		case order:
			firstAccount = senderAccount
			secondAccount = receiverAccount
		default:
			firstAccount = receiverAccount
			secondAccount = senderAccount
		}

		firstAccount.lock.Lock()
		secondAccount.lock.Lock()

	// now check for a valid transfer operation
	success := false
	if senderAccount.balance >= amount || allowOverdraw {
		senderAccount.balance -= amount
		receiverAccount.balance += amount
		success = true
	}
	secondAccount.lock.Unlock()
    firstAccount.lock.Unlock()

	return success
	}else{
		b.bankLock.Unlock()
		DPrintf("one of the accounts does not exist")
		return false
	}
	}else{
		DPrintf("Accounts cannot be the same")
		return false
	}
}

func (b *Bank) DepositAndCompare(accountID int, amount int, compareThreshold int) bool {
	b.bankLock.Lock()
	account := b.accounts[accountID]
	_, exists := b.accounts[accountID]
    b.bankLock.Unlock()
	if exists && amount > 0{
	account.lock.Lock()
    defer account.lock.Unlock()
	account.balance += amount
	compareResult := (account.balance >= compareThreshold)
	return compareResult
	}else{
		DPrintf("account is not real or value is not real")
		return false
	}
}

// returns the balance of the given account id
func (b *Bank) GetBalance(accountID int) int {
	b.bankLock.Lock()
	account := b.accounts[accountID]
	_, exists := b.accounts[accountID]
	b.bankLock.Unlock()
	// true or false cases with if once again
    if exists{
	account.lock.Lock()
	defer account.lock.Unlock()
	return account.balance
	}else{
		panic("account does not exist")
	}

}
// hi!
// this is cool
// yay!