package counter

import (
	"fmt"
	"sync"
	"testing"
)

func TestCountUp(t *testing.T) {

	c := New()

	if c.Read() != 0 {
		t.Error("Counter did not initialise correctly")
	}

	for j := 0; j < 2; j++ {

		iterations := 1000

		for i := 0; i < iterations; i++ {
			c.Increment()
			if c.Read() != i+1 {
				t.Error("Counter did not increment correctly")
			}

		}

		c.Reset()
		if c.Read() != 0 {
			t.Error("Counter did not Reset correctly")
		}
	}

}

func TestCompetingWrites(t *testing.T) {

	c := New()

	iterations := 1000
	competingFuncs := 200

	var wg sync.WaitGroup
	wg.Add(competingFuncs)

	for j := 0; j < competingFuncs; j++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				c.Increment()
			}
		}()
	}
	wg.Wait()

	if c.Read() != iterations*competingFuncs {
		t.Error("Locking failed, count was wrong")
	}

}

func TestCompareAgainstNoMux(t *testing.T) {

	iterations := 1000
	competingFuncs := 200

	nmExpected, nmGot := demoNonMux(iterations, competingFuncs)
	mExpected, mGot := demoMux(iterations, competingFuncs)

	fmt.Println("\u250C----------------------------------------\u2510")
	fmt.Println("| method | expected |    got   |  ok?    |")
	fmt.Println("|--------|----------|----------|---------|")
	fmt.Printf("|non-mux |%8d  |%8d  |  %v  |\n", nmExpected, nmGot, nmExpected == nmGot)
	fmt.Printf("|  this  |%8d  |%8d  |  %v   |\n", mExpected, mGot, mExpected == mGot)
	fmt.Println("\u2514----------------------------------------\u2518")

	if mExpected != mGot {
		t.Error("Muxed counter failed to get the right count")
	}

	if nmExpected == nmGot {
		t.Logf("non-muxed method did not experience an error - even a stopped clock is right twice day, but don't bank on it being right at other times!")
	}

}

func demoNonMux(iterations int, competingFuncs int) (int, int) {

	c := 0

	var wg sync.WaitGroup
	wg.Add(competingFuncs)

	for j := 0; j < competingFuncs; j++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				c++
			}
		}()
	}
	wg.Wait()

	return iterations * competingFuncs, c

}

func demoMux(iterations int, competingFuncs int) (int, int) {

	c := New()

	var wg sync.WaitGroup
	wg.Add(competingFuncs)

	for j := 0; j < competingFuncs; j++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				c.Increment()
			}
		}()
	}
	wg.Wait()

	return iterations * competingFuncs, c.Read()

}
