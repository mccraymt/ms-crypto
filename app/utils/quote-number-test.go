package utils

import (
	"fmt"
	rand "math/rand"
	"time"

	rdata "github.com/Pallinder/go-randomdata"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func randChomp() chomp {
	return chomp{ByteValue: byte(rand.Int31n(31))}
}

func randChompArray() []*chomp {
	rtn := make([]*chomp, IDCharLength)
	for idx := 0; idx < int(IDCharLength); idx++ {
		nc := randChomp()
		rtn[idx] = &nc
	}

	return rtn
}

// print out ids as generated to visually verify non-sequential
const dumpIDs = false

// how many times to run the unit tests
const unitTestCount = 100

// howmany times to build complete numbers
const integrationTestCount = 100

func TestChompXor() bool {
	rtn := true
	suc := 0
	var errors []string
	for cnt := 0; cnt < unitTestCount; cnt++ {
		c1 := randChomp()
		c2 := randChomp()

		c3, _ := c1.xor(&c2)
		c4, _ := c3.xor(&c1)
		c5, _ := c3.xor(&c2)
		if c4.ByteValue != c2.ByteValue {
			rtn = false
			errors = append(errors, fmt.Sprintf("XOR of %v and %v failed. Value %v did not back out with %v to expected %v", c1.ByteValue, c2.ByteValue, c3.ByteValue, c1.ByteValue, c2.ByteValue))
		} else if c5.ByteValue != c1.ByteValue {
			rtn = false
			errors = append(errors, fmt.Sprintf("XOR of %v and %v failed. Value %v did not back out with %v to expected %v", c1.ByteValue, c2.ByteValue, c3.ByteValue, c2.ByteValue, c1.ByteValue))
		} else {
			suc++
		}
	}
	if len(errors) > 0 {
		for _, te := range errors {
			fmt.Println(te)
		}
	}

	fmt.Printf("Successfully tested chomp xor %v times\r\n", suc)
	return rtn
}

func TestTimeToBitsAndBack() bool {
	suc := 0
	rtn := true
	for cnt := 0; cnt < unitTestCount; cnt++ {
		rightNow := time.Now()
		bits := QuoteNumber{}.GenerateTimeBits(&rightNow)
		backTime := QuoteNumber{}.TimeFromBits(bits)
		if rightNow.Sub(*backTime) > time.Duration(time.Nanosecond)*time.Duration(100000) {
			fmt.Printf("Time in '%v' does not match time out '%v'\r\n", rightNow, *backTime)
			rtn = false
		} else {
			suc++
		}
		time.Sleep(time.Millisecond)
	}
	fmt.Printf("Successfully tested time to bits and back %v times\r\n", suc)
	return rtn
}

func TestSaltAndUnsalt() bool {
	suc := 0
	//fmt.Printf("timestampUpperBound is '%v'\r\n", timestampUpperBound)
	rtn := true
	for cnt := 0; cnt < unitTestCount; cnt++ {
		inbits := rand.Int63n(timestampUpperBound)
		saltybits := QuoteNumber{}.SaltToFullBitlength(inbits)
		unsalted := QuoteNumber{}.UnsaltToTimestampBits(saltybits)
		if inbits != unsalted {
			rtn = false
			fmt.Printf("Error salting/unsalting: in '%v', out '%v'\r\n", inbits, unsalted)
		} else {
			suc++
		}
	}
	fmt.Printf("Successfully tested salt and unsalt %v times\r\n", suc)
	return rtn
}

func TestIntToChompArrayAndBack() bool {
	suc := 0
	q := QuoteNumber{}
	rtn := true
	maxKey := q.Power(2, (5 * IDCharLength))
	for cnt := 0; cnt < unitTestCount; cnt++ {
		inkey := rand.Int63n(maxKey)
		chomps := q.toChomps(inkey, 0)
		outkey := q.toNumber(chomps)
		if inkey != outkey {
			rtn = false
			fmt.Printf("Error converting to/from chomps: in '%v', out '%v'\r\n", inkey, outkey)
		} else {
			suc++
		}
	}

	fmt.Printf("Successfully tested int to chomp array and back %v times\r\n", suc)
	return rtn
}

func TestConfusticateAndDeconfusticate() bool {
	suc := 0
	rtn := true
	q := QuoteNumber{}

	for cnt := 0; cnt < unitTestCount; cnt++ {
		sa := randChompArray()
		ca := q.confusticate(sa)
		dca := q.deconfusticate(ca)
		iseq := true
		for idx := 0; idx < len(sa); idx++ {
			if sa[idx].ByteValue != dca[idx].ByteValue {
				iseq = false
			}
		}

		if !iseq {
			rtn = false
		} else {
			suc++
		}
	}

	fmt.Printf("Successfully confusticated and deconfusticated %v times\r\n", suc)
	return rtn
}

func TestChompArrayToTokensAndBack() bool {
	suc := 0
	rtn := true
	q := QuoteNumber{}

	for cnt := 0; cnt < unitTestCount; cnt++ {
		sa := randChompArray()
		ca := q.chompArrayToChompTokens(sa)
		dca := q.fromChompTokens(ca)
		iseq := true
		for idx := 0; idx < len(sa); idx++ {
			if sa[idx].ByteValue != dca[idx].ByteValue {
				iseq = false
			}
		}

		if !iseq {
			rtn = false
		} else {
			suc++
		}
	}

	fmt.Printf("Successfully converted chomp array to tokens and back %v times\r\n", suc)
	return rtn
}

func TestTimeToQuoteNumberAndBack() bool {
	suc := 0
	rtn := true
	q := QuoteNumber{}
	seenMap := make(map[string]bool)
	for cnt := 0; cnt < integrationTestCount; cnt++ {
		rightNow := time.Now()
		bits := q.CreateQuoteNumber(&rightNow)
		backTime := q.GetTimeAndServer(&bits)
		if rightNow.Sub(*backTime) > time.Duration(time.Nanosecond)*time.Duration(100000) {
			fmt.Printf("Time in '%v' does not match time out '%v'\r\n", rightNow, *backTime)
			rtn = false
		} else if ruhroh, found := seenMap[bits]; ruhroh && found {
			fmt.Printf("Quote number '%v' generated multiple times\r\n", bits)
			rtn = false
		} else {
			suc++
			seenMap[bits] = true
		}
		if dumpIDs {
			fmt.Println(bits)
		}
		time.Sleep(time.Millisecond)
	}
	fmt.Printf("Successfully tested full quote number generation cycle %v times\r\n", suc)
	return rtn
}

func TestEncodeDecodeIdDobName() bool {
	suc := 0
	rtn := true
	q := QuoteNumber{}
	baseId := int32(601123456)

	//integrationTestCount
	for cnt := 0; cnt < 1000; cnt++ {
		//id := rand.Int31()
		id := baseId + int32(cnt)
		y := rand.Intn(150) + 1910
		m := rand.Intn(12) + 1
		var d int
		switch m {
		case 1, 3, 5, 7, 8, 10, 12:
			d = rand.Intn(31) + 1
			break
		case 2:
			d = rand.Intn(28) + 1
			break
		default:
			d = rand.Intn(30) + 1
			break
		}

		nl := rand.Intn(10) + 2
		name := rdata.Letters(nl)
		hash := q.toSixteenBitHash(name)

		key := q.CreateForIdNameDOB(id, name, y, m, d)

		oid, onh, oy, om, od := q.ParseIdDOBNameHash(&key)
		//dumpIDs
		if true {
			fmt.Printf("\"%v\" => %v, \"%v\", %v-%v-%v\r\n", key, id, name, y, m, d)
		}

		thisSuccess := true
		if oy != y || om != m || od != d {
			thisSuccess = false
			fmt.Printf("Input date %v-%-v-%v did not match output date %v-%v-%v\r\n", y, m, d, oy, om, od)
		}

		if oid != id {
			thisSuccess = false
			fmt.Printf("Input id %v did not match output id %v\r\n", id, oid)
		}

		if onh != hash {
			thisSuccess = false
			fmt.Printf("Input name hash %v did not match output name hash %v\r\n", hash, onh)
		}

		if thisSuccess {
			suc++
		}
		rtn = rtn && thisSuccess
	}
	fmt.Printf("Successfully tested job number/DOB/name generation cycle %v times\r\n", suc)
	return rtn
}

var _ = Describe("DeepCompare", func() {
	It("should correctly xor chomps", func() {
		foo := TestChompXor()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly map time to bits and back", func() {
		foo := TestTimeToBitsAndBack()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly test complex structs", func() {
		foo := TestDeepCompareComplexStruct()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly salt and unsalt big integers", func() {
		foo := TestSaltAndUnsalt()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly convert big integers to chomp arrays and back", func() {
		foo := TestIntToChompArrayAndBack()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly confusticate and deconfusticate chomp arrays", func() {
		foo := TestConfusticateAndDeconfusticate()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly convert chomp arrays to tokens and back", func() {
		foo := TestChompArrayToTokensAndBack()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

	It("should correctly generate and decompose quote numbers", func() {
		foo := TestTimeToQuoteNumberAndBack()
		Ω(foo).Should(BeTrue())
		//Ω(gRecorder.Body).Should(MatchJSON(expectedNotFoundResponse()))
	})

})
