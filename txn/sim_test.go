package txn_test

import (
	"flag"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"labix.org/v2/mgo/txn"
	. "launchpad.net/gocheck"
	"math/rand"
	"time"
)

var duration = flag.Duration("duration", 1*time.Second, "duration for each simulation")

type params struct {
	killChance     float64
	slowdownChance float64
	slowdown       time.Duration

	unsafe         bool
	workers        int
	accounts       int
	reinsertCopy   bool
	reinsertZeroed bool

	changes int
}

func (s *S) TestSimulateFake(c *C) {
	simulate(c, params{
		unsafe:   true,
		workers:  1,
		accounts: 4,
	})
}

func (s *S) TestSimulate1Worker(c *C) {
	simulate(c, params{
		workers:        1,
		accounts:       4,
		killChance:     0.01,
		slowdownChance: 0.3,
		slowdown:       100 * time.Millisecond,
	})
}

func (s *S) TestSimulate4WorkersDense(c *C) {
	simulate(c, params{
		workers:        4,
		accounts:       2,
		killChance:     0.01,
		slowdownChance: 0.3,
		slowdown:       100 * time.Millisecond,
	})
}

func (s *S) TestSimulate4WorkersSparse(c *C) {
	simulate(c, params{
		workers:        4,
		accounts:       10,
		killChance:     0.01,
		slowdownChance: 0.3,
		slowdown:       100 * time.Millisecond,
	})
}

func (s *S) TestSimulateReinsertCopyFake(c *C) {
	simulate(c, params{
		unsafe:       true,
		workers:      1,
		accounts:     10,
		reinsertCopy: true,
	})
}

func (s *S) TestSimulateReinsertCopy1Worker(c *C) {
	simulate(c, params{
		workers:        1,
		accounts:       10,
		reinsertCopy:   true,
		killChance:     0.01,
		slowdownChance: 0.3,
		slowdown:       100 * time.Millisecond,
	})
}

func (s *S) TestSimulateReinsertCopy4Workers(c *C) {
	simulate(c, params{
		workers:        4,
		accounts:       10,
		reinsertCopy:   true,
		killChance:     0.01,
		slowdownChance: 0.3,
		slowdown:       100 * time.Millisecond,
	})
}

func (s *S) TestSimulateReinsertZeroedFake(c *C) {
	simulate(c, params{
		unsafe:         true,
		workers:        1,
		accounts:       10,
		reinsertZeroed: true,
	})
}

func (s *S) TestSimulateReinsertZeroed1Worker(c *C) {
	simulate(c, params{
		workers:        1,
		accounts:       10,
		reinsertZeroed: true,
		killChance:     0.01,
		slowdownChance: 0.3,
		slowdown:       100 * time.Millisecond,
	})
}

func (s *S) TestSimulateReinsertZeroed4Workers(c *C) {
	simulate(c, params{
		workers:        4,
		accounts:       10,
		reinsertZeroed: true,
		killChance:     0.01,
		slowdownChance: 0.3,
		slowdown:       100 * time.Millisecond,
	})
}

type balanceChange struct {
	id     bson.ObjectId
	origin int
	target int
	amount int
}

func simulate(c *C, params params) {
	rand.Seed(time.Now().UnixNano())

	txn.SetChaos(txn.Chaos{
		KillChance:     params.killChance,
		SlowdownChance: params.slowdownChance,
		Slowdown:       params.slowdown,
	})
	defer txn.SetChaos(txn.Chaos{})

	session, err := mgo.Dial(mgoaddr)
	c.Assert(err, IsNil)
	defer session.Close()

	db := session.DB("test")
	tc := db.C("tc")

	var runner *txn.Runner
	if params.unsafe {
		runner = txn.NewFakeRunner(tc)
	} else {
		runner = txn.NewRunner(tc)
	}

	accounts := db.C("accounts")
	for i := 0; i < params.accounts; i++ {
		err := accounts.Insert(M{"_id": i, "balance": 300})
		c.Assert(err, IsNil)
	}
	var stop time.Time
	if params.changes <= 0 {
		stop = time.Now().Add(*duration)
	}

	max := params.accounts
	if params.reinsertCopy || params.reinsertZeroed {
		max = int(float64(params.accounts) * 1.5)
	}

	changes := make(chan balanceChange, 1024)

	//session.SetMode(mgo.Eventual, true)
	for i := 0; i < params.workers; i++ {
		go func() {
			n := 0
			for {
				if n > 0 && n == params.changes {
					break
				}
				if !stop.IsZero() && time.Now().After(stop) {
					break
				}

				change := balanceChange{
					id:     bson.NewObjectId(),
					origin: rand.Intn(max),
					target: rand.Intn(max),
					amount: 100,
				}

				var old Account
				var oldExists bool
				if params.reinsertCopy || params.reinsertZeroed {
					if err := accounts.FindId(change.origin).One(&old); err != mgo.ErrNotFound {
						c.Check(err, IsNil)
						change.amount = old.Balance
						oldExists = true
					}
				}

				var ops []txn.Operation
				switch {
				case params.reinsertCopy && oldExists:
					ops = []txn.Operation{{
						Collection: "accounts",
						DocId:      change.origin,
						Assert:     M{"balance": change.amount},
						Remove:     true,
					}, {
						Collection: "accounts",
						DocId:      change.target,
						Assert:     txn.DocMissing,
						Insert:     M{"balance": change.amount},
					}}
				case params.reinsertZeroed && oldExists:
					ops = []txn.Operation{{
						Collection: "accounts",
						DocId:      change.target,
						Assert:     txn.DocMissing,
						Insert:     M{"balance": 0},
					}, {
						Collection: "accounts",
						DocId:      change.origin,
						Assert:     M{"balance": change.amount},
						Remove:     true,
					}, {
						Collection: "accounts",
						DocId:      change.target,
						Assert:     txn.DocExists,
						Change:     M{"$inc": M{"balance": change.amount}},
					}}
				default:
					ops = []txn.Operation{{
						Collection: "accounts",
						DocId:      change.origin,
						Assert:     M{"balance": M{"$gte": change.amount}},
						Change:     M{"$inc": M{"balance": -change.amount}},
					}, {
						Collection: "accounts",
						DocId:      change.target,
						Assert:     txn.DocExists,
						Change:     M{"$inc": M{"balance": change.amount}},
					}}
				}

				err = runner.Run(ops, change.id, nil)
				if err != nil && err != txn.ErrAborted && err != txn.ErrChaos {
					c.Check(err, IsNil)
				}
				n++
				changes <- change
			}
			changes <- balanceChange{}
		}()
	}

	alive := params.workers
	changeLog := make([]balanceChange, 0, 1024)
	for alive > 0 {
		change := <-changes
		if change.id == "" {
			alive--
		} else {
			changeLog = append(changeLog, change)
		}
	}
	c.Check(len(changeLog), Not(Equals), 0, Commentf("No operations were even attempted."))

	txn.SetChaos(txn.Chaos{})
	err = runner.ResumeAll()
	c.Assert(err, IsNil)

	n, err := accounts.Count()
	c.Check(err, IsNil)
	c.Check(n, Equals, params.accounts, Commentf("Number of accounts has changed."))

	n, err = accounts.Find(M{"balance": M{"$ge": 0}}).Count()
	c.Check(err, IsNil)
	c.Check(n, Equals, 0, Commentf("There are %d accounts with negative balance.", n))

	globalBalance := 0
	iter := accounts.Find(nil).Iter()
	account := Account{}
	for iter.Next(&account) {
		globalBalance += account.Balance
	}
	c.Check(iter.Err(), IsNil)
	c.Check(globalBalance, Equals, params.accounts*300, Commentf("Total amount of money should be constant."))

	// Compute and verify the exact final state of all accounts.
	balance := make(map[int]int)
	for i := 0; i < params.accounts; i++ {
		balance[i] += 300
	}
	var applied, aborted int
	for _, change := range changeLog {
		err := runner.Resume(change.id)
		if err == txn.ErrAborted {
			aborted++
			continue
		} else if err != nil {
			c.Fatalf("resuming %s failed: %v", change.id, err)
		}
		balance[change.origin] -= change.amount
		balance[change.target] += change.amount
		applied++
	}
	iter = accounts.Find(nil).Iter()
	for iter.Next(&account) {
		c.Assert(account.Balance, Equals, balance[account.Id])
	}
	c.Check(iter.Err(), IsNil)
	c.Logf("Total transactions: %d (%d applied, %d aborted)", len(changeLog), applied, aborted)
}