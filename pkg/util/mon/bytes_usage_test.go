// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package mon

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/randutil"

	"golang.org/x/net/context"
)

// randomSize generates a size greater or equal to zero, with a random
// distribution that is skewed towards zero and ensures that most
// generated values are smaller than `mag`.
func randomSize(rnd *rand.Rand, mag int64) int64 {
	return int64(rnd.ExpFloat64() * float64(mag) * 0.3679)
}

func TestMemoryAllocations(t *testing.T) {
	maxs := []int64{1, 9, 10, 11, 99, 100, 101, 0}
	hysteresisFactors := []int{1, 2, 10, 10000}
	poolAllocSizes := []int64{1, 2, 9, 10, 11, 100}
	preBudgets := []int64{0, 1, 2, 9, 10, 11, 100}

	rnd, seed := randutil.NewPseudoRand()
	t.Logf("random seed: %v", seed)

	ctx := context.Background()
	accs := make([]BytesAccount, 4)

	var pool BytesMonitor
	var m BytesMonitor
	var paramHeader func()

	// The following invariants will be checked at every step of the
	// test underneath.
	checkInvariants := func() {
		t.Helper()

		var sum int64
		fail := false
		for accI := range accs {
			if accs[accI].used < 0 {
				t.Errorf("account %d went negative: %d", accI, accs[accI].used)
				fail = true
			}
			sum += accs[accI].allocated()
		}
		if m.mu.curAllocated < 0 {
			t.Errorf("monitor current count went negative: %d", m.mu.curAllocated)
			fail = true
		}
		if sum != m.mu.curAllocated {
			t.Errorf("total account sum %d different from monitor count %d", sum, m.mu.curAllocated)
			fail = true
		}
		if m.mu.curBudget.used < 0 {
			t.Errorf("monitor current budget went negative: %d", m.mu.curBudget.used)
			fail = true
		}
		avail := m.mu.curBudget.allocated() + m.reserved.used
		if sum > avail {
			t.Errorf("total account sum %d greater than total monitor budget %d", sum, avail)
			fail = true
		}
		if pool.mu.curAllocated > pool.reserved.used {
			t.Errorf("pool cur %d exceeds max %d", pool.mu.curAllocated, pool.reserved.used)
			fail = true
		}
		if m.mu.curBudget.allocated() != pool.mu.curAllocated {
			t.Errorf("monitor budget %d different from pool cur %d", m.mu.curBudget.used, pool.mu.curAllocated)
			fail = true
		}

		if fail {
			t.Fatal("invariants not preserved")
		}
	}

	const numAccountOps = 200
	var linesBetweenHeaderReminders int
	var generateHeader func()
	var reportAndCheck func(string, ...interface{})
	if log.V(2) {
		// Detailed output: report the intermediate values of the
		// important variables at every stage of the test.
		linesBetweenHeaderReminders = 5
		generateHeader = func() {
			fmt.Println("")
			paramHeader()
			fmt.Printf(" mcur  mbud  mpre  pool ")
			for accI := range accs {
				fmt.Printf("%5s ", fmt.Sprintf("a%d", accI))
			}
			fmt.Println("")
		}
		reportAndCheck = func(extraFmt string, extras ...interface{}) {
			t.Helper()
			fmt.Printf("%5d %5d %5d %5d ", m.mu.curAllocated, m.mu.curBudget.used, m.reserved.used, pool.mu.curAllocated)
			for accI := range accs {
				fmt.Printf("%5d ", accs[accI].used)
			}
			fmt.Print("\t")
			fmt.Printf(extraFmt, extras...)
			fmt.Println("")
			checkInvariants()
		}
	} else {
		// More compact output.
		linesBetweenHeaderReminders = numAccountOps
		if testing.Verbose() {
			generateHeader = func() { paramHeader() }
		} else {
			generateHeader = func() {}
		}
		reportAndCheck = func(_ string, _ ...interface{}) {
			t.Helper()
			checkInvariants()
		}
	}

	for _, max := range maxs {
		pool = MakeMonitor("test", MemoryResource, nil, nil, 1, 1000)
		pool.Start(ctx, nil, MakeStandaloneBudget(max))

		for _, hf := range hysteresisFactors {
			maxAllocatedButUnusedBlocks = hf

			for _, pb := range preBudgets {
				mmax := pb + max

				for _, pa := range poolAllocSizes {
					paramHeader = func() { fmt.Printf("max %d, pb %d, as %d, hf %d\n", max, pb, pa, hf) }

					// We start with a fresh monitor for every set of
					// parameters.
					m = MakeMonitor("test", MemoryResource, nil, nil, pa, 1000)
					m.Start(ctx, &pool, MakeStandaloneBudget(pb))

					for i := 0; i < numAccountOps; i++ {
						if i%linesBetweenHeaderReminders == 0 {
							generateHeader()
						}

						// The following implements a random operation generator.
						// At every test iteration a random account is selected
						// and then a random operation is performed for that
						// account.

						accI := rnd.Intn(len(accs))
						switch rnd.Intn(3 /* number of states below */) {
						case 0:
							sz := randomSize(rnd, mmax)
							reportAndCheck("G [%5d] %5d", accI, sz)
							err := m.GrowAccount(ctx, &accs[accI], sz)
							if err == nil {
								reportAndCheck("G [%5d] ok", accI)
							} else {
								reportAndCheck("G [%5d] %s", accI, err)
							}
						case 1:
							reportAndCheck("C [%5d]", accI)
							m.ClearAccount(ctx, &accs[accI])
							reportAndCheck("C [%5d]", accI)
						case 2:
							osz := rnd.Int63n(accs[accI].used + 1)
							nsz := randomSize(rnd, mmax)
							reportAndCheck("R [%5d] %5d %5d", accI, osz, nsz)
							err := m.ResizeItem(ctx, &accs[accI], osz, nsz)
							if err == nil {
								reportAndCheck("R [%5d] ok", accI)
							} else {
								reportAndCheck("R [%5d] %s", accI, err)
							}
						}
					}

					// After all operations have been performed, ensure
					// that closing everything comes back to the initial situation.
					for accI := range accs {
						reportAndCheck("CL[%5d]", accI)
						m.ClearAccount(ctx, &accs[accI])
						reportAndCheck("CL[%5d]", accI)
					}

					m.Stop(ctx)
					if pool.mu.curAllocated != 0 {
						t.Fatalf("pool not empty after monitor close: %d", pool.mu.curAllocated)
					}
				}
			}
		}
		pool.Stop(ctx)
	}
}

func TestBytesAccount(t *testing.T) {
	defer leaktest.AfterTest(t)()

	ctx := context.Background()
	m := MakeMonitor("test", MemoryResource, nil, nil, 1, 1000)
	m.Start(ctx, nil, MakeStandaloneBudget(100))
	m.poolAllocationSize = 1
	maxAllocatedButUnusedBlocks = 1

	var a1, a2 BytesAccount

	m.OpenAccount(&a1)
	m.OpenAccount(&a2)

	if err := m.GrowAccount(ctx, &a1, 10); err != nil {
		t.Fatalf("monitor refused allocation: %v", err)
	}

	if err := m.GrowAccount(ctx, &a2, 30); err != nil {
		t.Fatalf("monitor refused allocation: %v", err)
	}

	if err := m.GrowAccount(ctx, &a1, 61); err == nil {
		t.Fatalf("monitor accepted excessive allocation")
	}

	if err := m.GrowAccount(ctx, &a2, 61); err == nil {
		t.Fatalf("monitor accepted excessive allocation")
	}

	m.ClearAccount(ctx, &a1)

	if err := m.GrowAccount(ctx, &a2, 61); err != nil {
		t.Fatalf("monitor refused allocation: %v", err)
	}

	if err := m.ResizeItem(ctx, &a2, 50, 60); err == nil {
		t.Fatalf("monitor accepted excessive allocation")
	}

	if err := m.ResizeItem(ctx, &a1, 0, 5); err != nil {
		t.Fatalf("monitor refused allocation: %v", err)
	}

	if err := m.ResizeItem(ctx, &a2, a2.used, 40); err != nil {
		t.Fatalf("monitor refused reset + allocation: %v", err)
	}

	m.CloseAccount(ctx, &a1)
	m.CloseAccount(ctx, &a2)

	if m.mu.curAllocated != 0 {
		t.Fatal("closing spans leaves bytes in monitor")
	}

	m.Stop(ctx)
}

func TestBytesMonitor(t *testing.T) {
	defer leaktest.AfterTest(t)()

	ctx := context.Background()
	m := MakeMonitor("test", MemoryResource, nil, nil, 1, 1000)
	m.Start(ctx, nil, MakeStandaloneBudget(100))
	maxAllocatedButUnusedBlocks = 1

	if err := m.reserveBytes(ctx, 10); err != nil {
		t.Fatalf("monitor refused small allocation: %v", err)
	}
	if err := m.reserveBytes(ctx, 91); err == nil {
		t.Fatalf("monitor accepted excessive allocation: %v", err)
	}
	if err := m.reserveBytes(ctx, 90); err != nil {
		t.Fatalf("monitor refused top allocation: %v", err)
	}
	if m.mu.curAllocated != 100 {
		t.Fatalf("incorrect current allocation: got %d, expected %d", m.mu.curAllocated, 100)
	}

	m.releaseBytes(ctx, 90) // Should succeed without panic.
	if m.mu.curAllocated != 10 {
		t.Fatalf("incorrect current allocation: got %d, expected %d", m.mu.curAllocated, 10)
	}
	if m.mu.maxAllocated != 100 {
		t.Fatalf("incorrect max allocation: got %d, expected %d", m.mu.maxAllocated, 100)
	}

	m.releaseBytes(ctx, 10) // Should succeed without panic.
	if m.mu.curAllocated != 0 {
		t.Fatalf("incorrect current allocation: got %d, expected %d", m.mu.curAllocated, 0)
	}

	limitedMonitor := MakeMonitorWithLimit("testlimit", MemoryResource, 10, nil, nil, 1, 1000)
	limitedMonitor.Start(ctx, &m, BoundAccount{})

	if err := limitedMonitor.reserveBytes(ctx, 10); err != nil {
		t.Fatalf("limited monitor refused small allocation: %v", err)
	}
	if err := limitedMonitor.reserveBytes(ctx, 1); err == nil {
		t.Fatal("limited monitor allowed allocation over limit")
	}
	limitedMonitor.releaseBytes(ctx, 10)

	limitedMonitor.Stop(ctx)
	m.Stop(ctx)
}

func TestMemoryAllocationEdgeCases(t *testing.T) {
	defer leaktest.AfterTest(t)()

	ctx := context.Background()
	m := MakeMonitor("test", MemoryResource,
		nil /* curCount */, nil /* maxHist */, 1e9 /* increment */, 1e9 /* noteworthy */)
	m.Start(ctx, nil, MakeStandaloneBudget(1e9))

	a := m.MakeBoundAccount()
	if err := a.Grow(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if err := a.Grow(ctx, math.MaxInt64); err == nil {
		t.Fatalf("expected error, but found success")
	}

	a.Close(ctx)
	m.Stop(ctx)
}

func BenchmarkBoundAccountGrow(b *testing.B) {
	ctx := context.Background()
	m := MakeMonitor("test", MemoryResource,
		nil /* curCount */, nil /* maxHist */, 1e9 /* increment */, 1e9 /* noteworthy */)
	m.Start(ctx, nil, MakeStandaloneBudget(1e9))

	a := m.MakeBoundAccount()
	for i := 0; i < b.N; i++ {
		_ = a.Grow(ctx, 1)
	}
}
