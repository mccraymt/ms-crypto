package utils

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

//QuoteNumber type encapsulates functions for generating and interpreting Quote Ids
type QuoteNumber struct{}

/*
 * What is a Chomp?
 * A byte is 8 bits. A "nybble" is 4 bits. We need something to deal with 5 bits at a time. So... chomp.
 * A 5-bit number gives 32 possibilities, easily converted to an alphanumeric character, avoiding ambiguous 1/l, 0/O.
 */

/*
 * How this shit works -- the setup
 * First, decide how many characters long you want your identifier to be (Chomp.IDCharLength)
 * Each character gives you five bits of data to work with. So 9 chars => 45 bits.
 * Decide how many of your bits to devote to the timestamp (Chomp.TimestampBits).
 * Timestamps are generated in tenth-miliseconds.
 * There are about 30 bits of tenth-miliseconds in a day.
 * There are a little more than 38 bits worth in a year.
 * 42 bits gives 13.94 years.
 * The rest of your bits will be used for random "salt".
 * More salt will reduce the chance of a collision when generating identifiers on different machines.
 * Pick a date for the epoch. This should be earlier than the earliest day you care about.
 */

/*
 * How this shit works -- the math
 * To generate a policy ID:
 * 1. Compute the tenth miliseconds (ticks / 1,000) since a certain min date (epoch).
 * 2. Mod this with 2 ^ TimestampBits to get an int of the desired length.
 * 3. Salt this int with the necessary number of high-order bits to get the full IDCharLength * 5 bits.
 * 4. Break the big int into 5-byte Chomps.
 * 5. Starting with the low-order byte, which changes 10000 times per second,
 *     XOR each byte upward. This allows each byte to change in a big way,
 *     obscuring the sequence. Yield the XOR product for each Chomp.
 * 6. Convert the Chomps into characters and concatenate them into a string.
 *
 * To get a datetime from a policy id:
 * 1. Convert the string into an array of Chomps.
 * 2. Reverse the XOR process to deconfusticate the Chomps.
 * 3. Combine the deconfusticated Chomps into a big integer.
 * 4. Cut off the salt bits to leave a timestampBits-sized integer.
 * 5. Add this integer to the epoch to get back a time.
 */

const MaxLegalQuoteNumberLength = 15
const MinLegalQuoteNumberLength = 8

type chomp struct {
	ByteValue byte
}

//CreateQuoteNumber is an all-in-one function for generating a Quote Id
func (q QuoteNumber) CreateQuoteNumber(timeCreated *time.Time) string {
	timeBits := q.GenerateTimeBits(timeCreated)
	saltyBits := q.SaltToFullBitlength(timeBits)
	chompArray := q.toChomps(saltyBits, 0)
	swirlyChomps := q.confusticate(chompArray)

	return q.chompArrayToChompTokens(swirlyChomps)
}

// GetTimeAndServer extracts date and other data stored in a Quote Id
func (q QuoteNumber) GetTimeAndServer(quoteIdentifier *string) *time.Time {
	if quoteIdentifier == nil || *quoteIdentifier == "" {
		return nil
	}
	cleanID := ""
	for _, thisChar := range *quoteIdentifier {
		uc := strings.ToUpper(string(thisChar))
		if _, found := IDNumbers[uc]; found {
			cleanID += uc
		}
	}
	swirlyChomps := q.fromChompTokens(cleanID)
	chompArray := q.deconfusticate(swirlyChomps)
	saltyBits := q.toNumber(chompArray)
	timeBits := q.UnsaltToTimestampBits(saltyBits)

	return q.TimeFromBits(timeBits)
}

// How long should the identifier be?
// Each char gives us 5 bits of data to work with.
const IDCharLength int32 = 9

// How much of the id should be used to store the time?
// Whatever bits remain will be randomly generated.
// This CANNOT be greater than IDCharLength * 5
const TimestampBits int32 = 42

//timestampUpperBound is the maximum number of tenth-miliseconds that we can work with
var timestampUpperBound = (&QuoteNumber{}).Power(2, TimestampBits)

// Turn a 5-bit number into an alphanumeric character. A, I, O, and U are omitted to
//  increase clarity and reduce the chances of generating an offensive string
var IDChars = map[int]string{
	0: "1", 1: "9", 2: "B", 3: "8", 4: "C", 5: "7",
	6: "D", 7: "6", 8: "E", 9: "5", 10: "F", 11: "4",
	12: "G", 13: "3", 14: "H", 15: "2", 16: "Q", 17: "S",
	18: "J", 19: "Z", 20: "K", 21: "Y", 22: "L", 23: "X",
	24: "M", 25: "W", 26: "N", 27: "V", 28: "R", 29: "0",
	30: "P", 31: "T",
}

// Turn a char back into a number.
// Redundant mappings for A, I, O, and U are included for backward compatibility
//  and error tolerance.
var IDNumbers = map[string]int{
	"A": 0, "9": 1, "B": 2, "8": 3, "C": 4, "7": 5,
	"D": 6, "6": 7, "E": 8, "5": 9, "F": 10, "4": 11,
	"G": 12, "3": 13, "H": 14, "2": 15, "Q": 16, "S": 17,
	"J": 18, "Z": 19, "K": 20, "Y": 21, "L": 22, "X": 23,
	"M": 24, "W": 25, "N": 26, "V": 27, "R": 28, "U": 29,
	"P": 30, "T": 31, "1": 0, "I": 0, "0": 29, "O": 29,
}

// Epoch is the earliest date we care about, expressed as a really big integer.
var Epoch = time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

// FilterBits is used to mask away the random bits we add
var FilterBits = (&QuoteNumber{}).Power(int64(2), (TimestampBits+1)) - 1

// Chomp Constructs chomp from a ulong
func (q QuoteNumber) chomp(seed int64) (*chomp, error) {
	if seed > 31 {
		return nil, fmt.Errorf("Error creating chomp: Seed %v too large", seed)
	}
	rtn := chomp{
		ByteValue: byte(seed),
	}
	return &rtn, nil
}

//xor is The all-important XOR operator
func (c *chomp) xor(c2 *chomp) (*chomp, error) {
	if c == nil || c2 == nil {
		return nil, errors.New("Error attempting to XOR two chomps: arguments cannot be nil")
	}
	xv := byte((*c).ByteValue ^ (*c2).ByteValue)
	rtn := chomp{
		ByteValue: xv,
	}
	return &rtn, nil
}

// takeChomp takes a chomp from the low-order end of a ulong
func (q QuoteNumber) takeChomp(input int64) (*chomp, int64) {
	bv := input & 31
	removed := chomp{ByteValue: byte(bv)}
	remain := int64(0)
	if input > 31 {
		remain = input >> 5
	}
	return &removed, remain
}

// toChomps breaks an int64 into a big-endian array of chomps
// pads the array to a determined length
func (q QuoteNumber) toChomps(input int64, pad int32) []*chomp {
	if pad == 0 {
		pad = IDCharLength
	}
	var chompList []*chomp
	remainder := input
	for remainder > 0 {
		var bittenOff *chomp
		bittenOff, remainder = q.takeChomp(remainder)
		chompList = append(chompList, bittenOff)
	}

	for int32(len(chompList)) < pad {
		mtchomp := chomp{ByteValue: byte(0)}
		chompList = append(chompList, &mtchomp)
	}

	length := len(chompList)
	rtnList := make([]*chomp, length)
	for idx := 0; idx < length; idx++ {
		rtnList[idx] = chompList[length-idx-1]
	}

	return rtnList
}

//toNumber converts a big-endian collection of chomps into an int64
func (q QuoteNumber) toNumber(chomps []*chomp) int64 {
	rtn := int64(0)
	for _, thisChomp := range chomps {
		rtn = int64((rtn << 5) | int64(thisChomp.ByteValue))
	}
	return rtn
}

//toToken converts a chomp into a token character
func (c *chomp) toToken() string {
	rtn, _ := IDChars[int(c.ByteValue)]
	return rtn
}

//fromToken converts a token character into a chomp
func (c *chomp) fromToken(input string) error {
	key := strings.ToUpper(strings.TrimSpace(input))
	if len(key) > 1 {
		return fmt.Errorf("Error creating chomp from token. Must be a single character: %v", input)
	}

	if thisByte, found := IDNumbers[key]; !found {
		return fmt.Errorf("Error creating chomp from token. Invalid token: %v", input)
	} else {
		c.ByteValue = byte(thisByte)
	}
	return nil
}

//intToChompTokens turns a big integer into a string of chomp tokens
func (q QuoteNumber) intToChompTokens(input int64) (string, error) {
	if input < 0 {
		return "", fmt.Errorf("Error creating chomp tokens. Must be nonnegative: %v", input)
	}

	chomps := q.toChomps(input, 0)
	return q.chompArrayToChompTokens(chomps), nil
}

//chompArrayToChompTokens turns an array of chomps into a string
func (q QuoteNumber) chompArrayToChompTokens(src []*chomp) string {
	rtn := ""
	for _, thisChomp := range src {
		rtn += thisChomp.toToken()
	}
	return rtn
}

//fromChompTokens turns an ID string into an array of chomps
func (q QuoteNumber) fromChompTokens(input string) []*chomp {
	trimPut := strings.ToUpper(strings.TrimSpace(input))
	rtn := make([]*chomp, len(trimPut))
	for idx, thisChar := range trimPut {
		thisChomp := chomp{}
		thisChomp.fromToken(string(thisChar))
		rtn[idx] = &thisChomp
	}

	return rtn
}

//numberfromChompTokens turns a Quote ID into an integer
func (q QuoteNumber) numberfromChompTokens(input string) int64 {
	chomps := q.fromChompTokens(input)

	return q.toNumber(chomps)
}

//confusticate adds cyclic xor obfuscation to an array of chomps
func (q QuoteNumber) confusticate(src []*chomp) []*chomp {
	length := len(src)
	first := src[0]
	last := src[length-1]
	rtn := make([]*chomp, length)
	lastChomp, _ := first.xor(last)
	rtn[0] = lastChomp

	for cnt := 1; cnt < length; cnt++ {
		rtn[cnt], _ = rtn[cnt-1].xor(src[cnt])
	}

	return rtn
}

//deconfusticate removes cyclic xor obfuscation from an array of chomps
func (q QuoteNumber) deconfusticate(src []*chomp) []*chomp {
	length := len(src)
	rtn := make([]*chomp, length)
	secondToLast := src[length-2]
	last := src[length-1]
	lastChomp, _ := last.xor(secondToLast)
	rtn[0], _ = lastChomp.xor(src[0])

	for idx := 1; idx < length; idx++ {
		rtn[idx], _ = src[idx-1].xor(src[idx])
	}

	return rtn
}

//ToString Converts a chomp to a string
func (c *chomp) ToString() string {
	return fmt.Sprintf("%v", c.ByteValue)
}

//GenerateTimeBits creates a big integer from a given time or current time
func (q QuoteNumber) GenerateTimeBits(date *time.Time) int64 {
	if date == nil {
		// sleep for a tenth-milisecond to make sure we aren't hitting this multiple times too quickly
		time.Sleep(100000 * time.Nanosecond)
		jetzt := time.Now()
		date = &jetzt
	}

	// returns the number of tenth-seconds since the epoch, modded to bitlength
	dur := date.Sub(Epoch)
	return (int64(dur.Nanoseconds() / 100000)) & FilterBits
}

//SaltToFullBitlength adds random high-order bits to a ulong to yield a longer ulong
func (q QuoteNumber) SaltToFullBitlength(src int64) int64 {
	maxRand := q.Power(2, ((IDCharLength * 5) - TimestampBits))
	salt := rand.Int63n(maxRand)

	return (salt * timestampUpperBound) + src
}

//UnsaltToTimestampBits removes random salt bits from a QuoteNumber
func (q QuoteNumber) UnsaltToTimestampBits(src int64) int64 {
	return src % timestampUpperBound
}

//TimeFromBits extracts an actual time from a computed integer
func (q QuoteNumber) TimeFromBits(bits int64) *time.Time {
	for true {
		dur := time.Duration(time.Nanosecond) * time.Duration(100000*bits)
		dur2 := time.Duration(time.Nanosecond) * time.Duration(100000*timestampUpperBound)
		firstTime := Epoch.Add(dur)
		secondTime := firstTime.Add(dur2)

		if secondTime.After(time.Now()) {
			return &firstTime
		}

		bits += timestampUpperBound
	}

	return nil
}

//Power raises one int to the power of another, an operator oddly missing from golang
func (q QuoteNumber) Power(x int64, n int32) int64 {
	if n == 0 {
		return int64(1)
	}
	if n == 1 {
		return x
	}

	even := n%2 == 0
	xsq := x * x
	if even {
		newPow := int32(n / 2)
		return q.Power(xsq, newPow)
	}
	newPow := int32((n - 1) / 2)
	return x * q.Power(xsq, newPow)
}

func (q QuoteNumber) IsValid(qn *string) bool {
	if qn == nil {
		return false
	}

	qns := *qn

	if len(qns) > MaxLegalQuoteNumberLength {
		return false
	}

	if len(qns) < MinLegalQuoteNumberLength {
		return false
	}

	return true
}

func (q QuoteNumber) ValidOrNew(qn *string) string {
	if qn == nil || !q.IsValid(qn) {
		return q.CreateQuoteNumber(nil)
	}

	return *qn
}

// CreateForIdNameDOB creates a 12-char identifier from a JobNumber, birthdate, and last name
// JobNumber and birthdate are recoverable; lastname serves as a confirmation hash
func (q QuoteNumber) CreateForIdNameDOB(id int32, name string, y, m, d int) string {
	saltyBits := int64(q.idNameDOBToInt(id, name, y, m, d))
	chompArray := q.toChomps(saltyBits, 12)
	swirlyChomps := q.confusticate(chompArray)

	return q.chompArrayToChompTokens(swirlyChomps)
}

func (q QuoteNumber) ParseIdDOBNameHash(quoteIdentifier *string) (int32, uint32, int, int, int) {
	if quoteIdentifier == nil || *quoteIdentifier == "" {
		return 0, 0, 0, 0, 0
	}
	cleanID := ""
	for _, thisChar := range *quoteIdentifier {
		uc := strings.ToUpper(string(thisChar))
		if _, found := IDNumbers[uc]; found {
			cleanID += uc
		}
	}
	swirlyChomps := q.fromChompTokens(cleanID)
	chompArray := q.deconfusticate(swirlyChomps)
	saltyBits := uint64(q.toNumber(chompArray))

	return q.intToIDNameDOB(saltyBits)
}

func (q QuoteNumber) idNameDOBToInt(id int32, name string, doby, dobm, dobd int) uint64 {
	rtn := uint64(0)

	// Use 12 bits from the name for the high-order bits
	nameInt := q.toSixteenBitHash(name) & 0xfff

	// make room for the other 48 bits
	rtn = rtn | (uint64(nameInt) << 48)

	// put all 32 bits of the passed-in id in the middle
	rtn = rtn | (uint64(id) << 16)

	// we'll keep 16 bits of days for the DOB -- that's almost 180 years
	dobInt := uint64(q.daysAfterBirthdateEpoch(doby, dobm, dobd)) & 0xffff

	rtn = rtn | dobInt

	return rtn
}

func (q QuoteNumber) intToIDNameDOB(key uint64) (int32, uint32, int, int, int) {
	dobInt := int32(key & 0xffff)
	doby, dobm, dobd := q.intToBirthdate(dobInt)
	key = key >> 16
	id := int32(key & 0xffffffff)
	key = key >> 32

	return id, uint32(key), doby, dobm, dobd
}

// Guessing that as I write this we don't insure anyone over 106 years old
var birthdateEpoch = time.Date(1910, 1, 1, 12, 0, 0, 0, time.UTC)

func (q QuoteNumber) daysAfterBirthdateEpoch(y, m, d int) int32 {
	dob := time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.UTC)

	if !dob.After(birthdateEpoch) {
		return 0
	}
	afterEpochDur := dob.Sub(birthdateEpoch)
	return int32(afterEpochDur.Hours() / 24)
}

func (q QuoteNumber) intToBirthdate(bdd int32) (int, int, int) {
	dur := time.Duration(bdd*24) * time.Hour
	rtn := birthdateEpoch.Add(dur)

	return rtn.Year(), int(rtn.Month()), rtn.Day()
}

// pad with 21 since it makes a nice 10101 pattern
const padByte = byte(21)
const fiveBits = byte(31)
const sixteenBits = uint32(65535)
const ignoreChars = byte(97)

func (q QuoteNumber) toSixteenBitHash(name string) uint32 {
	asciiFinder := regexp.MustCompile("[a-z]")
	buf := strings.Join(asciiFinder.FindAllString(strings.ToLower(name), -1), "")
	if len(buf) == 0 {
		return uint32(0)
	}
	origBuf := buf
	lenSalt := uint32(len(buf) % 4)
	// need at least 6 characters to do anything useful
	for len(buf) < 6 {
		buf += origBuf
	}
	bufPos := 0
	bufLength := len(buf)
	hash := uint32(0)
	// Prime the frame with the length of the name
	currentFrameLength := uint(2)
	frame := lenSalt

	// Keep moving through the buffer until we get to the end
	for bufPos < bufLength-1 {
		// load the frame with more bits
		for currentFrameLength < 16 {
			bitsToLoad := padByte
			// don't start padding until we run out of characters
			if bufPos < bufLength {
				// we're working with lowercase utf-8 here, so we don't care about the first 96
				// subtract those 96 and take the 5 low-order bits
				bitsToLoad = (buf[bufPos] - ignoreChars) & fiveBits
				bufPos++
			} // end grab next byte
			newBitsMagnified := uint32(bitsToLoad) << currentFrameLength
			frame = newBitsMagnified | frame
			currentFrameLength += 5
		} // end load frame

		_ = "breakpoint"

		// the frame is loaded, so now we pop the low-order 16 bits and XOR them with the old hash
		freshBits := frame & sixteenBits
		frame = frame >> 16
		currentFrameLength = currentFrameLength - 16
		// add the fresh bits to the hash
		hash = hash ^ freshBits
	}

	_ = "breakpoint"
	// smash down to 12 bits
	hash = hash & 0xfff
	return hash
}

func trackThis(m *map[uint32]string, c *int, name string) {
	hash := QuoteNumber{}.toSixteenBitHash(name)
	theMap := *m
	if foo, success := theMap[hash]; success {
		fmt.Printf("%v: Name %v shares hash %v with name %v\r\n", *c, name, hash, foo)
		*c++
		return
	}
	theMap[hash] = name
}

func TestBitch() {
	count := 0
	nameDict := make(map[uint32]string)
	//q := QuoteNumber{}
	trackThis(&nameDict, &count, "")
	// for checking
	trackThis(&nameDict, &count, "NG")
	trackThis(&nameDict, &count, "Sizemore")
	trackThis(&nameDict, &count, "SMITH")
	trackThis(&nameDict, &count, "JOHNSON")
	trackThis(&nameDict, &count, "WILLIAMS")
	trackThis(&nameDict, &count, "BROWN")
	trackThis(&nameDict, &count, "JONES")
	trackThis(&nameDict, &count, "MILLER")
	trackThis(&nameDict, &count, "DAVIS")
	trackThis(&nameDict, &count, "GARCIA")
	trackThis(&nameDict, &count, "RODRIGUEZ")
	trackThis(&nameDict, &count, "WILSON")
	trackThis(&nameDict, &count, "MARTINEZ")
	trackThis(&nameDict, &count, "ANDERSON")
	trackThis(&nameDict, &count, "TAYLOR")
	trackThis(&nameDict, &count, "THOMAS")
	trackThis(&nameDict, &count, "HERNANDEZ")
	trackThis(&nameDict, &count, "MOORE")
	trackThis(&nameDict, &count, "MARTIN")
	trackThis(&nameDict, &count, "JACKSON")
	trackThis(&nameDict, &count, "THOMPSON")
	trackThis(&nameDict, &count, "WHITE")
	trackThis(&nameDict, &count, "LOPEZ")
	trackThis(&nameDict, &count, "LEE")
	trackThis(&nameDict, &count, "GONZALEZ")
	trackThis(&nameDict, &count, "HARRIS")
	trackThis(&nameDict, &count, "CLARK")
	trackThis(&nameDict, &count, "LEWIS")
	trackThis(&nameDict, &count, "ROBINSON")
	trackThis(&nameDict, &count, "WALKER")
	trackThis(&nameDict, &count, "PEREZ")
	trackThis(&nameDict, &count, "HALL")
	trackThis(&nameDict, &count, "YOUNG")
	trackThis(&nameDict, &count, "ALLEN")
	trackThis(&nameDict, &count, "SANCHEZ")
	trackThis(&nameDict, &count, "WRIGHT")
	trackThis(&nameDict, &count, "KING")
	trackThis(&nameDict, &count, "SCOTT")
	trackThis(&nameDict, &count, "GREEN")
	trackThis(&nameDict, &count, "BAKER")
	trackThis(&nameDict, &count, "ADAMS")
	trackThis(&nameDict, &count, "NELSON")
	trackThis(&nameDict, &count, "HILL")
	trackThis(&nameDict, &count, "RAMIREZ")
	trackThis(&nameDict, &count, "CAMPBELL")
	trackThis(&nameDict, &count, "MITCHELL")
	trackThis(&nameDict, &count, "ROBERTS")
	trackThis(&nameDict, &count, "CARTER")
	trackThis(&nameDict, &count, "PHILLIPS")
	trackThis(&nameDict, &count, "EVANS")
	trackThis(&nameDict, &count, "TURNER")
	trackThis(&nameDict, &count, "TORRES")
	trackThis(&nameDict, &count, "PARKER")
	trackThis(&nameDict, &count, "COLLINS")
	trackThis(&nameDict, &count, "EDWARDS")
	trackThis(&nameDict, &count, "STEWART")
	trackThis(&nameDict, &count, "FLORES")
	trackThis(&nameDict, &count, "MORRIS")
	trackThis(&nameDict, &count, "NGUYEN")
	trackThis(&nameDict, &count, "MURPHY")
	trackThis(&nameDict, &count, "RIVERA")
	trackThis(&nameDict, &count, "COOK")
	trackThis(&nameDict, &count, "ROGERS")
	trackThis(&nameDict, &count, "MORGAN")
	trackThis(&nameDict, &count, "PETERSON")
	trackThis(&nameDict, &count, "COOPER")
	trackThis(&nameDict, &count, "REED")
	trackThis(&nameDict, &count, "BAILEY")
	trackThis(&nameDict, &count, "BELL")
	trackThis(&nameDict, &count, "GOMEZ")
	trackThis(&nameDict, &count, "KELLY")
	trackThis(&nameDict, &count, "HOWARD")
	trackThis(&nameDict, &count, "WARD")
	trackThis(&nameDict, &count, "COX")
	trackThis(&nameDict, &count, "DIAZ")
	trackThis(&nameDict, &count, "RICHARDSON")
	trackThis(&nameDict, &count, "WOOD")
	trackThis(&nameDict, &count, "WATSON")
	trackThis(&nameDict, &count, "BROOKS")
	trackThis(&nameDict, &count, "BENNETT")
	trackThis(&nameDict, &count, "GRAY")
	trackThis(&nameDict, &count, "JAMES")
	trackThis(&nameDict, &count, "REYES")
	trackThis(&nameDict, &count, "CRUZ")
	trackThis(&nameDict, &count, "HUGHES")
	trackThis(&nameDict, &count, "PRICE")
	trackThis(&nameDict, &count, "MYERS")
	trackThis(&nameDict, &count, "LONG")
	trackThis(&nameDict, &count, "FOSTER")
	trackThis(&nameDict, &count, "SANDERS")
	trackThis(&nameDict, &count, "ROSS")
	trackThis(&nameDict, &count, "MORALES")
	trackThis(&nameDict, &count, "POWELL")
	trackThis(&nameDict, &count, "SULLIVAN")
	trackThis(&nameDict, &count, "RUSSELL")
	trackThis(&nameDict, &count, "ORTIZ")
	trackThis(&nameDict, &count, "JENKINS")
	trackThis(&nameDict, &count, "GUTIERREZ")
	trackThis(&nameDict, &count, "PERRY")
	trackThis(&nameDict, &count, "BUTLER")
	trackThis(&nameDict, &count, "BARNES")
	trackThis(&nameDict, &count, "FISHER")
	trackThis(&nameDict, &count, "HENDERSON")
	trackThis(&nameDict, &count, "COLEMAN")
	trackThis(&nameDict, &count, "SIMMONS")
	trackThis(&nameDict, &count, "PATTERSON")
	trackThis(&nameDict, &count, "JORDAN")
	trackThis(&nameDict, &count, "REYNOLDS")
	trackThis(&nameDict, &count, "HAMILTON")
	trackThis(&nameDict, &count, "GRAHAM")
	trackThis(&nameDict, &count, "KIM")
	trackThis(&nameDict, &count, "GONZALES")
	trackThis(&nameDict, &count, "ALEXANDER")
	trackThis(&nameDict, &count, "RAMOS")
	trackThis(&nameDict, &count, "WALLACE")
	trackThis(&nameDict, &count, "GRIFFIN")
	trackThis(&nameDict, &count, "WEST")
	trackThis(&nameDict, &count, "COLE")
	trackThis(&nameDict, &count, "HAYES")
	trackThis(&nameDict, &count, "CHAVEZ")
	trackThis(&nameDict, &count, "GIBSON")
	trackThis(&nameDict, &count, "BRYANT")
	trackThis(&nameDict, &count, "ELLIS")
	trackThis(&nameDict, &count, "STEVENS")
	trackThis(&nameDict, &count, "MURRAY")
	trackThis(&nameDict, &count, "FORD")
	trackThis(&nameDict, &count, "MARSHALL")
	trackThis(&nameDict, &count, "OWENS")
	trackThis(&nameDict, &count, "MCDONALD")
	trackThis(&nameDict, &count, "HARRISON")
	trackThis(&nameDict, &count, "RUIZ")
	trackThis(&nameDict, &count, "KENNEDY")
	trackThis(&nameDict, &count, "WELLS")
	trackThis(&nameDict, &count, "ALVAREZ")
	trackThis(&nameDict, &count, "WOODS")
	trackThis(&nameDict, &count, "MENDOZA")
	trackThis(&nameDict, &count, "CASTILLO")
	trackThis(&nameDict, &count, "OLSON")
	trackThis(&nameDict, &count, "WEBB")
	trackThis(&nameDict, &count, "WASHINGTON")
	trackThis(&nameDict, &count, "TUCKER")
	trackThis(&nameDict, &count, "FREEMAN")
	trackThis(&nameDict, &count, "BURNS")
	trackThis(&nameDict, &count, "HENRY")
	trackThis(&nameDict, &count, "VASQUEZ")
	trackThis(&nameDict, &count, "SNYDER")
	trackThis(&nameDict, &count, "SIMPSON")
	trackThis(&nameDict, &count, "CRAWFORD")
	trackThis(&nameDict, &count, "JIMENEZ")
	trackThis(&nameDict, &count, "PORTER")
	trackThis(&nameDict, &count, "MASON")
	trackThis(&nameDict, &count, "SHAW")
	trackThis(&nameDict, &count, "GORDON")
	trackThis(&nameDict, &count, "WAGNER")
	trackThis(&nameDict, &count, "HUNTER")
	trackThis(&nameDict, &count, "ROMERO")
	trackThis(&nameDict, &count, "HICKS")
	trackThis(&nameDict, &count, "DIXON")
	trackThis(&nameDict, &count, "HUNT")
	trackThis(&nameDict, &count, "PALMER")
	trackThis(&nameDict, &count, "ROBERTSON")
	trackThis(&nameDict, &count, "BLACK")
	trackThis(&nameDict, &count, "HOLMES")
	trackThis(&nameDict, &count, "STONE")
	trackThis(&nameDict, &count, "MEYER")
	trackThis(&nameDict, &count, "BOYD")
	trackThis(&nameDict, &count, "MILLS")
	trackThis(&nameDict, &count, "WARREN")
	trackThis(&nameDict, &count, "FOX")
	trackThis(&nameDict, &count, "ROSE")
	trackThis(&nameDict, &count, "RICE")
	trackThis(&nameDict, &count, "MORENO")
	trackThis(&nameDict, &count, "SCHMIDT")
	trackThis(&nameDict, &count, "PATEL")
	trackThis(&nameDict, &count, "FERGUSON")
	trackThis(&nameDict, &count, "NICHOLS")
	trackThis(&nameDict, &count, "HERRERA")
	trackThis(&nameDict, &count, "MEDINA")
	trackThis(&nameDict, &count, "RYAN")
	trackThis(&nameDict, &count, "FERNANDEZ")
	trackThis(&nameDict, &count, "WEAVER")
	trackThis(&nameDict, &count, "DANIELS")
	trackThis(&nameDict, &count, "STEPHENS")
	trackThis(&nameDict, &count, "GARDNER")
	trackThis(&nameDict, &count, "PAYNE")
	trackThis(&nameDict, &count, "KELLEY")
	trackThis(&nameDict, &count, "DUNN")
	trackThis(&nameDict, &count, "PIERCE")
	trackThis(&nameDict, &count, "ARNOLD")
	trackThis(&nameDict, &count, "TRAN")
	trackThis(&nameDict, &count, "SPENCER")
	trackThis(&nameDict, &count, "PETERS")
	trackThis(&nameDict, &count, "HAWKINS")
	trackThis(&nameDict, &count, "GRANT")
	trackThis(&nameDict, &count, "HANSEN")
	trackThis(&nameDict, &count, "CASTRO")
	trackThis(&nameDict, &count, "HOFFMAN")
	trackThis(&nameDict, &count, "HART")
	trackThis(&nameDict, &count, "ELLIOTT")
	trackThis(&nameDict, &count, "CUNNINGHAM")
	trackThis(&nameDict, &count, "KNIGHT")
	trackThis(&nameDict, &count, "BRADLEY")
	trackThis(&nameDict, &count, "CARROLL")
	trackThis(&nameDict, &count, "HUDSON")
	trackThis(&nameDict, &count, "DUNCAN")
	trackThis(&nameDict, &count, "ARMSTRONG")
	trackThis(&nameDict, &count, "BERRY")
	trackThis(&nameDict, &count, "ANDREWS")
	trackThis(&nameDict, &count, "JOHNSTON")
	trackThis(&nameDict, &count, "RAY")
	trackThis(&nameDict, &count, "LANE")
	trackThis(&nameDict, &count, "RILEY")
	trackThis(&nameDict, &count, "CARPENTER")
	trackThis(&nameDict, &count, "PERKINS")
	trackThis(&nameDict, &count, "AGUILAR")
	trackThis(&nameDict, &count, "SILVA")
	trackThis(&nameDict, &count, "RICHARDS")
	trackThis(&nameDict, &count, "WILLIS")
	trackThis(&nameDict, &count, "MATTHEWS")
	trackThis(&nameDict, &count, "CHAPMAN")
	trackThis(&nameDict, &count, "LAWRENCE")
	trackThis(&nameDict, &count, "GARZA")
	trackThis(&nameDict, &count, "VARGAS")
	trackThis(&nameDict, &count, "WATKINS")
	trackThis(&nameDict, &count, "WHEELER")
	trackThis(&nameDict, &count, "LARSON")
	trackThis(&nameDict, &count, "CARLSON")
	trackThis(&nameDict, &count, "HARPER")
	trackThis(&nameDict, &count, "GEORGE")
	trackThis(&nameDict, &count, "GREENE")
	trackThis(&nameDict, &count, "BURKE")
	trackThis(&nameDict, &count, "GUZMAN")
	trackThis(&nameDict, &count, "MORRISON")
	trackThis(&nameDict, &count, "MUNOZ")
	trackThis(&nameDict, &count, "JACOBS")
	trackThis(&nameDict, &count, "OBRIEN")
	trackThis(&nameDict, &count, "LAWSON")
	trackThis(&nameDict, &count, "FRANKLIN")
	trackThis(&nameDict, &count, "LYNCH")
	trackThis(&nameDict, &count, "BISHOP")
	trackThis(&nameDict, &count, "CARR")
	trackThis(&nameDict, &count, "SALAZAR")
	trackThis(&nameDict, &count, "AUSTIN")
	trackThis(&nameDict, &count, "MENDEZ")
	trackThis(&nameDict, &count, "GILBERT")
	trackThis(&nameDict, &count, "JENSEN")
	trackThis(&nameDict, &count, "WILLIAMSON")
	trackThis(&nameDict, &count, "MONTGOMERY")
	trackThis(&nameDict, &count, "HARVEY")
	trackThis(&nameDict, &count, "OLIVER")
	trackThis(&nameDict, &count, "HOWELL")
	trackThis(&nameDict, &count, "DEAN")
	trackThis(&nameDict, &count, "HANSON")
	trackThis(&nameDict, &count, "WEBER")
	trackThis(&nameDict, &count, "GARRETT")
	trackThis(&nameDict, &count, "SIMS")
	trackThis(&nameDict, &count, "BURTON")
	trackThis(&nameDict, &count, "FULLER")
	trackThis(&nameDict, &count, "SOTO")
	trackThis(&nameDict, &count, "MCCOY")
	trackThis(&nameDict, &count, "WELCH")
	trackThis(&nameDict, &count, "CHEN")
	trackThis(&nameDict, &count, "SCHULTZ")
	trackThis(&nameDict, &count, "WALTERS")
	trackThis(&nameDict, &count, "REID")
	trackThis(&nameDict, &count, "FIELDS")
	trackThis(&nameDict, &count, "WALSH")
	trackThis(&nameDict, &count, "LITTLE")
	trackThis(&nameDict, &count, "FOWLER")
	trackThis(&nameDict, &count, "BOWMAN")
	trackThis(&nameDict, &count, "DAVIDSON")
	trackThis(&nameDict, &count, "MAY")
	trackThis(&nameDict, &count, "DAY")
	trackThis(&nameDict, &count, "SCHNEIDER")
	trackThis(&nameDict, &count, "NEWMAN")
	trackThis(&nameDict, &count, "BREWER")
	trackThis(&nameDict, &count, "LUCAS")
	trackThis(&nameDict, &count, "HOLLAND")
	trackThis(&nameDict, &count, "WONG")
	trackThis(&nameDict, &count, "BANKS")
	trackThis(&nameDict, &count, "SANTOS")
	trackThis(&nameDict, &count, "CURTIS")
	trackThis(&nameDict, &count, "PEARSON")
	trackThis(&nameDict, &count, "DELGADO")
	trackThis(&nameDict, &count, "VALDEZ")
	trackThis(&nameDict, &count, "PENA")
	trackThis(&nameDict, &count, "RIOS")
	trackThis(&nameDict, &count, "DOUGLAS")
	trackThis(&nameDict, &count, "SANDOVAL")
	trackThis(&nameDict, &count, "BARRETT")
	trackThis(&nameDict, &count, "HOPKINS")
	trackThis(&nameDict, &count, "KELLER")
	trackThis(&nameDict, &count, "GUERRERO")
	trackThis(&nameDict, &count, "STANLEY")
	trackThis(&nameDict, &count, "BATES")
	trackThis(&nameDict, &count, "ALVARADO")
	trackThis(&nameDict, &count, "BECK")
	trackThis(&nameDict, &count, "ORTEGA")
	trackThis(&nameDict, &count, "WADE")
	trackThis(&nameDict, &count, "ESTRADA")
	trackThis(&nameDict, &count, "CONTRERAS")
	trackThis(&nameDict, &count, "BARNETT")
	trackThis(&nameDict, &count, "CALDWELL")
	trackThis(&nameDict, &count, "SANTIAGO")
	trackThis(&nameDict, &count, "LAMBERT")
	trackThis(&nameDict, &count, "POWERS")
	trackThis(&nameDict, &count, "CHAMBERS")
	trackThis(&nameDict, &count, "NUNEZ")
	trackThis(&nameDict, &count, "CRAIG")
	trackThis(&nameDict, &count, "LEONARD")
	trackThis(&nameDict, &count, "LOWE")
	trackThis(&nameDict, &count, "RHODES")
	trackThis(&nameDict, &count, "BYRD")
	trackThis(&nameDict, &count, "GREGORY")
	trackThis(&nameDict, &count, "SHELTON")
	trackThis(&nameDict, &count, "FRAZIER")
	trackThis(&nameDict, &count, "BECKER")
	trackThis(&nameDict, &count, "MALDONADO")
	trackThis(&nameDict, &count, "FLEMING")
	trackThis(&nameDict, &count, "VEGA")
	trackThis(&nameDict, &count, "SUTTON")
	trackThis(&nameDict, &count, "COHEN")
	trackThis(&nameDict, &count, "JENNINGS")
	trackThis(&nameDict, &count, "PARKS")
	trackThis(&nameDict, &count, "MCDANIEL")
	trackThis(&nameDict, &count, "WATTS")
	trackThis(&nameDict, &count, "BARKER")
	trackThis(&nameDict, &count, "NORRIS")
	trackThis(&nameDict, &count, "VAUGHN")
	trackThis(&nameDict, &count, "VAZQUEZ")
	trackThis(&nameDict, &count, "HOLT")
	trackThis(&nameDict, &count, "SCHWARTZ")
	trackThis(&nameDict, &count, "STEELE")
	trackThis(&nameDict, &count, "BENSON")
	trackThis(&nameDict, &count, "NEAL")
	trackThis(&nameDict, &count, "DOMINGUEZ")
	trackThis(&nameDict, &count, "HORTON")
	trackThis(&nameDict, &count, "TERRY")
	trackThis(&nameDict, &count, "WOLFE")
	trackThis(&nameDict, &count, "HALE")
	trackThis(&nameDict, &count, "LYONS")
	trackThis(&nameDict, &count, "GRAVES")
	trackThis(&nameDict, &count, "HAYNES")
	trackThis(&nameDict, &count, "MILES")
	trackThis(&nameDict, &count, "PARK")
	trackThis(&nameDict, &count, "WARNER")
	trackThis(&nameDict, &count, "PADILLA")
	trackThis(&nameDict, &count, "BUSH")
	trackThis(&nameDict, &count, "THORNTON")
	trackThis(&nameDict, &count, "MCCARTHY")
	trackThis(&nameDict, &count, "MANN")
	trackThis(&nameDict, &count, "ZIMMERMAN")
	trackThis(&nameDict, &count, "ERICKSON")
	trackThis(&nameDict, &count, "FLETCHER")
	trackThis(&nameDict, &count, "MCKINNEY")
	trackThis(&nameDict, &count, "PAGE")
	trackThis(&nameDict, &count, "DAWSON")
	trackThis(&nameDict, &count, "JOSEPH")
	trackThis(&nameDict, &count, "MARQUEZ")
	trackThis(&nameDict, &count, "REEVES")
	trackThis(&nameDict, &count, "KLEIN")
	trackThis(&nameDict, &count, "ESPINOZA")
	trackThis(&nameDict, &count, "BALDWIN")
	trackThis(&nameDict, &count, "MORAN")
	trackThis(&nameDict, &count, "LOVE")
	trackThis(&nameDict, &count, "ROBBINS")
	trackThis(&nameDict, &count, "HIGGINS")
	trackThis(&nameDict, &count, "BALL")
	trackThis(&nameDict, &count, "CORTEZ")
	trackThis(&nameDict, &count, "LE")
	trackThis(&nameDict, &count, "GRIFFITH")
	trackThis(&nameDict, &count, "BOWEN")
	trackThis(&nameDict, &count, "SHARP")
	trackThis(&nameDict, &count, "CUMMINGS")
	trackThis(&nameDict, &count, "RAMSEY")
	trackThis(&nameDict, &count, "HARDY")
	trackThis(&nameDict, &count, "SWANSON")
	trackThis(&nameDict, &count, "BARBER")
	trackThis(&nameDict, &count, "ACOSTA")
	trackThis(&nameDict, &count, "LUNA")
	trackThis(&nameDict, &count, "CHANDLER")
	trackThis(&nameDict, &count, "DANIEL")
	trackThis(&nameDict, &count, "BLAIR")
	trackThis(&nameDict, &count, "CROSS")
	trackThis(&nameDict, &count, "SIMON")
	trackThis(&nameDict, &count, "DENNIS")
	trackThis(&nameDict, &count, "OCONNOR")
	trackThis(&nameDict, &count, "QUINN")
	trackThis(&nameDict, &count, "GROSS")
	trackThis(&nameDict, &count, "NAVARRO")
	trackThis(&nameDict, &count, "MOSS")
	trackThis(&nameDict, &count, "FITZGERALD")
	trackThis(&nameDict, &count, "DOYLE")
	trackThis(&nameDict, &count, "MCLAUGHLIN")
	trackThis(&nameDict, &count, "ROJAS")
	trackThis(&nameDict, &count, "RODGERS")
	trackThis(&nameDict, &count, "STEVENSON")
	trackThis(&nameDict, &count, "SINGH")
	trackThis(&nameDict, &count, "YANG")
	trackThis(&nameDict, &count, "FIGUEROA")
	trackThis(&nameDict, &count, "HARMON")
	trackThis(&nameDict, &count, "NEWTON")
	trackThis(&nameDict, &count, "PAUL")
	trackThis(&nameDict, &count, "MANNING")
	trackThis(&nameDict, &count, "GARNER")
	trackThis(&nameDict, &count, "MCGEE")
	trackThis(&nameDict, &count, "REESE")
	trackThis(&nameDict, &count, "FRANCIS")
	trackThis(&nameDict, &count, "BURGESS")
	trackThis(&nameDict, &count, "ADKINS")
	trackThis(&nameDict, &count, "GOODMAN")
	trackThis(&nameDict, &count, "CURRY")
	trackThis(&nameDict, &count, "BRADY")
	trackThis(&nameDict, &count, "CHRISTENSEN")
	trackThis(&nameDict, &count, "POTTER")
	trackThis(&nameDict, &count, "WALTON")
	trackThis(&nameDict, &count, "GOODWIN")
	trackThis(&nameDict, &count, "MULLINS")
	trackThis(&nameDict, &count, "MOLINA")
	trackThis(&nameDict, &count, "WEBSTER")
	trackThis(&nameDict, &count, "FISCHER")
	trackThis(&nameDict, &count, "CAMPOS")
	trackThis(&nameDict, &count, "AVILA")
	trackThis(&nameDict, &count, "SHERMAN")
	trackThis(&nameDict, &count, "TODD")
	trackThis(&nameDict, &count, "CHANG")
	trackThis(&nameDict, &count, "BLAKE")
	trackThis(&nameDict, &count, "MALONE")
	trackThis(&nameDict, &count, "WOLF")
	trackThis(&nameDict, &count, "HODGES")
	trackThis(&nameDict, &count, "JUAREZ")
	trackThis(&nameDict, &count, "GILL")
	trackThis(&nameDict, &count, "FARMER")
	trackThis(&nameDict, &count, "HINES")
	trackThis(&nameDict, &count, "GALLAGHER")
	trackThis(&nameDict, &count, "DURAN")
	trackThis(&nameDict, &count, "HUBBARD")
	trackThis(&nameDict, &count, "CANNON")
	trackThis(&nameDict, &count, "MIRANDA")
	trackThis(&nameDict, &count, "WANG")
	trackThis(&nameDict, &count, "SAUNDERS")
	trackThis(&nameDict, &count, "TATE")
	trackThis(&nameDict, &count, "MACK")
	trackThis(&nameDict, &count, "HAMMOND")
	trackThis(&nameDict, &count, "CARRILLO")
	trackThis(&nameDict, &count, "TOWNSEND")
	trackThis(&nameDict, &count, "WISE")
	trackThis(&nameDict, &count, "INGRAM")
	trackThis(&nameDict, &count, "BARTON")
	trackThis(&nameDict, &count, "MEJIA")
	trackThis(&nameDict, &count, "AYALA")
	trackThis(&nameDict, &count, "SCHROEDER")
	trackThis(&nameDict, &count, "HAMPTON")
	trackThis(&nameDict, &count, "ROWE")
	trackThis(&nameDict, &count, "PARSONS")
	trackThis(&nameDict, &count, "FRANK")
	trackThis(&nameDict, &count, "WATERS")
	trackThis(&nameDict, &count, "STRICKLAND")
	trackThis(&nameDict, &count, "OSBORNE")
	trackThis(&nameDict, &count, "MAXWELL")
	trackThis(&nameDict, &count, "CHAN")
	trackThis(&nameDict, &count, "DELEON")
	trackThis(&nameDict, &count, "NORMAN")
	trackThis(&nameDict, &count, "HARRINGTON")
	trackThis(&nameDict, &count, "CASEY")
	trackThis(&nameDict, &count, "PATTON")
	trackThis(&nameDict, &count, "LOGAN")
	trackThis(&nameDict, &count, "BOWERS")
	trackThis(&nameDict, &count, "MUELLER")
	trackThis(&nameDict, &count, "GLOVER")
	trackThis(&nameDict, &count, "FLOYD")
	trackThis(&nameDict, &count, "HARTMAN")
	trackThis(&nameDict, &count, "BUCHANAN")
	trackThis(&nameDict, &count, "COBB")
	trackThis(&nameDict, &count, "FRENCH")
	trackThis(&nameDict, &count, "KRAMER")
	trackThis(&nameDict, &count, "MCCORMICK")
	trackThis(&nameDict, &count, "CLARKE")
	trackThis(&nameDict, &count, "TYLER")
	trackThis(&nameDict, &count, "GIBBS")
	trackThis(&nameDict, &count, "MOODY")
	trackThis(&nameDict, &count, "CONNER")
	trackThis(&nameDict, &count, "SPARKS")
	trackThis(&nameDict, &count, "MCGUIRE")
	trackThis(&nameDict, &count, "LEON")
	trackThis(&nameDict, &count, "BAUER")
	trackThis(&nameDict, &count, "NORTON")
	trackThis(&nameDict, &count, "POPE")
	trackThis(&nameDict, &count, "FLYNN")
	trackThis(&nameDict, &count, "HOGAN")
	trackThis(&nameDict, &count, "ROBLES")
	trackThis(&nameDict, &count, "SALINAS")
	trackThis(&nameDict, &count, "YATES")
	trackThis(&nameDict, &count, "LINDSEY")
	trackThis(&nameDict, &count, "LLOYD")
	trackThis(&nameDict, &count, "MARSH")
	trackThis(&nameDict, &count, "MCBRIDE")
	trackThis(&nameDict, &count, "OWEN")
	trackThis(&nameDict, &count, "SOLIS")
	trackThis(&nameDict, &count, "PHAM")
	trackThis(&nameDict, &count, "LANG")
	trackThis(&nameDict, &count, "PRATT")
	trackThis(&nameDict, &count, "LARA")
	trackThis(&nameDict, &count, "BROCK")
	trackThis(&nameDict, &count, "BALLARD")
	trackThis(&nameDict, &count, "TRUJILLO")
	trackThis(&nameDict, &count, "SHAFFER")
	trackThis(&nameDict, &count, "DRAKE")
	trackThis(&nameDict, &count, "ROMAN")
	trackThis(&nameDict, &count, "AGUIRRE")
	trackThis(&nameDict, &count, "MORTON")
	trackThis(&nameDict, &count, "STOKES")
	trackThis(&nameDict, &count, "LAMB")
	trackThis(&nameDict, &count, "PACHECO")
	trackThis(&nameDict, &count, "PATRICK")
	trackThis(&nameDict, &count, "COCHRAN")
	trackThis(&nameDict, &count, "SHEPHERD")
	trackThis(&nameDict, &count, "CAIN")
	trackThis(&nameDict, &count, "BURNETT")
	trackThis(&nameDict, &count, "HESS")
	trackThis(&nameDict, &count, "LI")
	trackThis(&nameDict, &count, "CERVANTES")
	trackThis(&nameDict, &count, "OLSEN")
	trackThis(&nameDict, &count, "BRIGGS")
	trackThis(&nameDict, &count, "OCHOA")
	trackThis(&nameDict, &count, "CABRERA")
	trackThis(&nameDict, &count, "VELASQUEZ")
	trackThis(&nameDict, &count, "MONTOYA")
	trackThis(&nameDict, &count, "ROTH")
	trackThis(&nameDict, &count, "MEYERS")
	trackThis(&nameDict, &count, "CARDENAS")
	trackThis(&nameDict, &count, "FUENTES")
	trackThis(&nameDict, &count, "WEISS")
	trackThis(&nameDict, &count, "WILKINS")
	trackThis(&nameDict, &count, "HOOVER")
	trackThis(&nameDict, &count, "NICHOLSON")
	trackThis(&nameDict, &count, "UNDERWOOD")
	trackThis(&nameDict, &count, "SHORT")
	trackThis(&nameDict, &count, "CARSON")
	trackThis(&nameDict, &count, "MORROW")
	trackThis(&nameDict, &count, "COLON")
	trackThis(&nameDict, &count, "HOLLOWAY")
	trackThis(&nameDict, &count, "SUMMERS")
	trackThis(&nameDict, &count, "BRYAN")
	trackThis(&nameDict, &count, "PETERSEN")
	trackThis(&nameDict, &count, "MCKENZIE")
	trackThis(&nameDict, &count, "SERRANO")
	trackThis(&nameDict, &count, "WILCOX")
	trackThis(&nameDict, &count, "CAREY")
	trackThis(&nameDict, &count, "CLAYTON")
	trackThis(&nameDict, &count, "POOLE")
	trackThis(&nameDict, &count, "CALDERON")
	trackThis(&nameDict, &count, "GALLEGOS")
	trackThis(&nameDict, &count, "GREER")
	trackThis(&nameDict, &count, "RIVAS")
	trackThis(&nameDict, &count, "GUERRA")
	trackThis(&nameDict, &count, "DECKER")
	trackThis(&nameDict, &count, "COLLIER")
	trackThis(&nameDict, &count, "WALL")
	trackThis(&nameDict, &count, "WHITAKER")
	trackThis(&nameDict, &count, "BASS")
	trackThis(&nameDict, &count, "FLOWERS")
	trackThis(&nameDict, &count, "DAVENPORT")
	trackThis(&nameDict, &count, "CONLEY")
	trackThis(&nameDict, &count, "HOUSTON")
	trackThis(&nameDict, &count, "HUFF")
	trackThis(&nameDict, &count, "COPELAND")
	trackThis(&nameDict, &count, "HOOD")
	trackThis(&nameDict, &count, "MONROE")
	trackThis(&nameDict, &count, "MASSEY")
	trackThis(&nameDict, &count, "ROBERSON")
	trackThis(&nameDict, &count, "COMBS")
	trackThis(&nameDict, &count, "FRANCO")
	trackThis(&nameDict, &count, "LARSEN")
	trackThis(&nameDict, &count, "PITTMAN")
	trackThis(&nameDict, &count, "RANDALL")
	trackThis(&nameDict, &count, "SKINNER")
	trackThis(&nameDict, &count, "WILKINSON")
	trackThis(&nameDict, &count, "KIRBY")
	trackThis(&nameDict, &count, "CAMERON")
	trackThis(&nameDict, &count, "BRIDGES")
	trackThis(&nameDict, &count, "ANTHONY")
	trackThis(&nameDict, &count, "RICHARD")
	trackThis(&nameDict, &count, "KIRK")
	trackThis(&nameDict, &count, "BRUCE")
	trackThis(&nameDict, &count, "SINGLETON")
	trackThis(&nameDict, &count, "MATHIS")
	trackThis(&nameDict, &count, "BRADFORD")
	trackThis(&nameDict, &count, "BOONE")
	trackThis(&nameDict, &count, "ABBOTT")
	trackThis(&nameDict, &count, "CHARLES")
	trackThis(&nameDict, &count, "ALLISON")
	trackThis(&nameDict, &count, "SWEENEY")
	trackThis(&nameDict, &count, "ATKINSON")
	trackThis(&nameDict, &count, "HORN")
	trackThis(&nameDict, &count, "JEFFERSON")
	trackThis(&nameDict, &count, "ROSALES")
	trackThis(&nameDict, &count, "YORK")
	trackThis(&nameDict, &count, "CHRISTIAN")
	trackThis(&nameDict, &count, "PHELPS")
	trackThis(&nameDict, &count, "FARRELL")
	trackThis(&nameDict, &count, "CASTANEDA")
	trackThis(&nameDict, &count, "NASH")
	trackThis(&nameDict, &count, "DICKERSON")
	trackThis(&nameDict, &count, "BOND")
	trackThis(&nameDict, &count, "WYATT")
	trackThis(&nameDict, &count, "FOLEY")
	trackThis(&nameDict, &count, "CHASE")
	trackThis(&nameDict, &count, "GATES")
	trackThis(&nameDict, &count, "VINCENT")
	trackThis(&nameDict, &count, "MATHEWS")
	trackThis(&nameDict, &count, "HODGE")
	trackThis(&nameDict, &count, "GARRISON")
	trackThis(&nameDict, &count, "TREVINO")
	trackThis(&nameDict, &count, "VILLARREAL")
	trackThis(&nameDict, &count, "HEATH")
	trackThis(&nameDict, &count, "DALTON")
	trackThis(&nameDict, &count, "VALENCIA")
	trackThis(&nameDict, &count, "CALLAHAN")
	trackThis(&nameDict, &count, "HENSLEY")
	trackThis(&nameDict, &count, "ATKINS")
	trackThis(&nameDict, &count, "HUFFMAN")
	trackThis(&nameDict, &count, "ROY")
	trackThis(&nameDict, &count, "BOYER")
	trackThis(&nameDict, &count, "SHIELDS")
	trackThis(&nameDict, &count, "LIN")
	trackThis(&nameDict, &count, "HANCOCK")
	trackThis(&nameDict, &count, "GRIMES")
	trackThis(&nameDict, &count, "GLENN")
	trackThis(&nameDict, &count, "CLINE")
	trackThis(&nameDict, &count, "DELACRUZ")
	trackThis(&nameDict, &count, "CAMACHO")
	trackThis(&nameDict, &count, "DILLON")
	trackThis(&nameDict, &count, "PARRISH")
	trackThis(&nameDict, &count, "ONEILL")
	trackThis(&nameDict, &count, "MELTON")
	trackThis(&nameDict, &count, "BOOTH")
	trackThis(&nameDict, &count, "KANE")
	trackThis(&nameDict, &count, "BERG")
	trackThis(&nameDict, &count, "HARRELL")
	trackThis(&nameDict, &count, "PITTS")
	trackThis(&nameDict, &count, "SAVAGE")
	trackThis(&nameDict, &count, "WIGGINS")
	trackThis(&nameDict, &count, "BRENNAN")
	trackThis(&nameDict, &count, "SALAS")
	trackThis(&nameDict, &count, "MARKS")
	trackThis(&nameDict, &count, "RUSSO")
	trackThis(&nameDict, &count, "SAWYER")
	trackThis(&nameDict, &count, "BAXTER")
	trackThis(&nameDict, &count, "GOLDEN")
	trackThis(&nameDict, &count, "HUTCHINSON")
	trackThis(&nameDict, &count, "LIU")
	trackThis(&nameDict, &count, "WALTER")
	trackThis(&nameDict, &count, "MCDOWELL")
	trackThis(&nameDict, &count, "WILEY")
	trackThis(&nameDict, &count, "RICH")
	trackThis(&nameDict, &count, "HUMPHREY")
	trackThis(&nameDict, &count, "JOHNS")
	trackThis(&nameDict, &count, "KOCH")
	trackThis(&nameDict, &count, "SUAREZ")
	trackThis(&nameDict, &count, "HOBBS")
	trackThis(&nameDict, &count, "BEARD")
	trackThis(&nameDict, &count, "GILMORE")
	trackThis(&nameDict, &count, "IBARRA")
	trackThis(&nameDict, &count, "KEITH")
	trackThis(&nameDict, &count, "MACIAS")
	trackThis(&nameDict, &count, "KHAN")
	trackThis(&nameDict, &count, "ANDRADE")
	trackThis(&nameDict, &count, "WARE")
	trackThis(&nameDict, &count, "STEPHENSON")
	trackThis(&nameDict, &count, "HENSON")
	trackThis(&nameDict, &count, "WILKERSON")
	trackThis(&nameDict, &count, "DYER")
	trackThis(&nameDict, &count, "MCCLURE")
	trackThis(&nameDict, &count, "BLACKWELL")
	trackThis(&nameDict, &count, "MERCADO")
	trackThis(&nameDict, &count, "TANNER")
	trackThis(&nameDict, &count, "EATON")
	trackThis(&nameDict, &count, "CLAY")
	trackThis(&nameDict, &count, "BARRON")
	trackThis(&nameDict, &count, "BEASLEY")
	trackThis(&nameDict, &count, "ONEAL")
	trackThis(&nameDict, &count, "SMALL")
	trackThis(&nameDict, &count, "PRESTON")
	trackThis(&nameDict, &count, "WU")
	trackThis(&nameDict, &count, "ZAMORA")
	trackThis(&nameDict, &count, "MACDONALD")
	trackThis(&nameDict, &count, "VANCE")
	trackThis(&nameDict, &count, "SNOW")
	trackThis(&nameDict, &count, "MCCLAIN")
	trackThis(&nameDict, &count, "STAFFORD")
	trackThis(&nameDict, &count, "OROZCO")
	trackThis(&nameDict, &count, "BARRY")
	trackThis(&nameDict, &count, "ENGLISH")
	trackThis(&nameDict, &count, "SHANNON")
	trackThis(&nameDict, &count, "KLINE")
	trackThis(&nameDict, &count, "JACOBSON")
	trackThis(&nameDict, &count, "WOODARD")
	trackThis(&nameDict, &count, "HUANG")
	trackThis(&nameDict, &count, "KEMP")
	trackThis(&nameDict, &count, "MOSLEY")
	trackThis(&nameDict, &count, "PRINCE")
	trackThis(&nameDict, &count, "MERRITT")
	trackThis(&nameDict, &count, "HURST")
	trackThis(&nameDict, &count, "VILLANUEVA")
	trackThis(&nameDict, &count, "ROACH")
	trackThis(&nameDict, &count, "NOLAN")
	trackThis(&nameDict, &count, "LAM")
	trackThis(&nameDict, &count, "YODER")
	trackThis(&nameDict, &count, "MCCULLOUGH")
	trackThis(&nameDict, &count, "LESTER")
	trackThis(&nameDict, &count, "SANTANA")
	trackThis(&nameDict, &count, "VALENZUELA")
	trackThis(&nameDict, &count, "WINTERS")
	trackThis(&nameDict, &count, "BARRERA")
	trackThis(&nameDict, &count, "ORR")
	trackThis(&nameDict, &count, "LEACH")
	trackThis(&nameDict, &count, "BERGER")
	trackThis(&nameDict, &count, "MCKEE")
	trackThis(&nameDict, &count, "STRONG")
	trackThis(&nameDict, &count, "CONWAY")
	trackThis(&nameDict, &count, "STEIN")
	trackThis(&nameDict, &count, "WHITEHEAD")
	trackThis(&nameDict, &count, "BULLOCK")
	trackThis(&nameDict, &count, "ESCOBAR")
	trackThis(&nameDict, &count, "KNOX")
	trackThis(&nameDict, &count, "MEADOWS")
	trackThis(&nameDict, &count, "SOLOMON")
	trackThis(&nameDict, &count, "VELEZ")
	trackThis(&nameDict, &count, "ODONNELL")
	trackThis(&nameDict, &count, "KERR")
	trackThis(&nameDict, &count, "STOUT")
	trackThis(&nameDict, &count, "BLANKENSHIP")
	trackThis(&nameDict, &count, "BROWNING")
	trackThis(&nameDict, &count, "KENT")
	trackThis(&nameDict, &count, "LOZANO")
	trackThis(&nameDict, &count, "BARTLETT")
	trackThis(&nameDict, &count, "PRUITT")
	trackThis(&nameDict, &count, "BUCK")
	trackThis(&nameDict, &count, "BARR")
	trackThis(&nameDict, &count, "GAINES")
	trackThis(&nameDict, &count, "DURHAM")
	trackThis(&nameDict, &count, "GENTRY")
	trackThis(&nameDict, &count, "MCINTYRE")
	trackThis(&nameDict, &count, "SLOAN")
	trackThis(&nameDict, &count, "ROCHA")
	trackThis(&nameDict, &count, "MELENDEZ")
	trackThis(&nameDict, &count, "HERMAN")
	trackThis(&nameDict, &count, "SEXTON")
	trackThis(&nameDict, &count, "MOON")
	trackThis(&nameDict, &count, "HENDRICKS")
	trackThis(&nameDict, &count, "RANGEL")
	trackThis(&nameDict, &count, "STARK")
	trackThis(&nameDict, &count, "LOWERY")
	trackThis(&nameDict, &count, "HARDIN")
	trackThis(&nameDict, &count, "HULL")
	trackThis(&nameDict, &count, "SELLERS")
	trackThis(&nameDict, &count, "ELLISON")
	trackThis(&nameDict, &count, "CALHOUN")
	trackThis(&nameDict, &count, "GILLESPIE")
	trackThis(&nameDict, &count, "MORA")
	trackThis(&nameDict, &count, "KNAPP")
	trackThis(&nameDict, &count, "MCCALL")
	trackThis(&nameDict, &count, "MORSE")
	trackThis(&nameDict, &count, "DORSEY")
	trackThis(&nameDict, &count, "WEEKS")
	trackThis(&nameDict, &count, "NIELSEN")
	trackThis(&nameDict, &count, "LIVINGSTON")
	trackThis(&nameDict, &count, "LEBLANC")
	trackThis(&nameDict, &count, "MCLEAN")
	trackThis(&nameDict, &count, "BRADSHAW")
	trackThis(&nameDict, &count, "GLASS")
	trackThis(&nameDict, &count, "MIDDLETON")
	trackThis(&nameDict, &count, "BUCKLEY")
	trackThis(&nameDict, &count, "SCHAEFER")
	trackThis(&nameDict, &count, "FROST")
	trackThis(&nameDict, &count, "HOWE")
	trackThis(&nameDict, &count, "HOUSE")
	trackThis(&nameDict, &count, "MCINTOSH")
	trackThis(&nameDict, &count, "HO")
	trackThis(&nameDict, &count, "PENNINGTON")
	trackThis(&nameDict, &count, "REILLY")
	trackThis(&nameDict, &count, "HEBERT")
	trackThis(&nameDict, &count, "MCFARLAND")
	trackThis(&nameDict, &count, "HICKMAN")
	trackThis(&nameDict, &count, "NOBLE")
	trackThis(&nameDict, &count, "SPEARS")
	trackThis(&nameDict, &count, "CONRAD")
	trackThis(&nameDict, &count, "ARIAS")
	trackThis(&nameDict, &count, "GALVAN")
	trackThis(&nameDict, &count, "VELAZQUEZ")
	trackThis(&nameDict, &count, "HUYNH")
	trackThis(&nameDict, &count, "FREDERICK")
	trackThis(&nameDict, &count, "RANDOLPH")
	trackThis(&nameDict, &count, "CANTU")
	trackThis(&nameDict, &count, "FITZPATRICK")
	trackThis(&nameDict, &count, "MAHONEY")
	trackThis(&nameDict, &count, "PECK")
	trackThis(&nameDict, &count, "VILLA")
	trackThis(&nameDict, &count, "MICHAEL")
	trackThis(&nameDict, &count, "DONOVAN")
	trackThis(&nameDict, &count, "MCCONNELL")
	trackThis(&nameDict, &count, "WALLS")
	trackThis(&nameDict, &count, "BOYLE")
	trackThis(&nameDict, &count, "MAYER")
	trackThis(&nameDict, &count, "ZUNIGA")
	trackThis(&nameDict, &count, "GILES")
	trackThis(&nameDict, &count, "PINEDA")
	trackThis(&nameDict, &count, "PACE")
	trackThis(&nameDict, &count, "HURLEY")
	trackThis(&nameDict, &count, "MAYS")
	trackThis(&nameDict, &count, "MCMILLAN")
	trackThis(&nameDict, &count, "CROSBY")
	trackThis(&nameDict, &count, "AYERS")
	trackThis(&nameDict, &count, "CASE")
	trackThis(&nameDict, &count, "BENTLEY")
	trackThis(&nameDict, &count, "SHEPARD")
	trackThis(&nameDict, &count, "EVERETT")
	trackThis(&nameDict, &count, "PUGH")
	trackThis(&nameDict, &count, "DAVID")
	trackThis(&nameDict, &count, "MCMAHON")
	trackThis(&nameDict, &count, "DUNLAP")
	trackThis(&nameDict, &count, "BENDER")
	trackThis(&nameDict, &count, "HAHN")
	trackThis(&nameDict, &count, "HARDING")
	trackThis(&nameDict, &count, "ACEVEDO")
	trackThis(&nameDict, &count, "RAYMOND")
	trackThis(&nameDict, &count, "BLACKBURN")
	trackThis(&nameDict, &count, "DUFFY")
	trackThis(&nameDict, &count, "LANDRY")
	trackThis(&nameDict, &count, "DOUGHERTY")
	trackThis(&nameDict, &count, "BAUTISTA")
	trackThis(&nameDict, &count, "SHAH")
	trackThis(&nameDict, &count, "POTTS")
	trackThis(&nameDict, &count, "ARROYO")
	trackThis(&nameDict, &count, "VALENTINE")
	trackThis(&nameDict, &count, "MEZA")
	trackThis(&nameDict, &count, "GOULD")
	trackThis(&nameDict, &count, "VAUGHAN")
	trackThis(&nameDict, &count, "FRY")
	trackThis(&nameDict, &count, "RUSH")
	trackThis(&nameDict, &count, "AVERY")
	trackThis(&nameDict, &count, "HERRING")
	trackThis(&nameDict, &count, "DODSON")
	trackThis(&nameDict, &count, "CLEMENTS")
	trackThis(&nameDict, &count, "SAMPSON")
	trackThis(&nameDict, &count, "TAPIA")
	trackThis(&nameDict, &count, "BEAN")
	trackThis(&nameDict, &count, "LYNN")
	trackThis(&nameDict, &count, "CRANE")
	trackThis(&nameDict, &count, "FARLEY")
	trackThis(&nameDict, &count, "CISNEROS")
	trackThis(&nameDict, &count, "BENTON")
	trackThis(&nameDict, &count, "ASHLEY")
	trackThis(&nameDict, &count, "MCKAY")
	trackThis(&nameDict, &count, "FINLEY")
	trackThis(&nameDict, &count, "BEST")
	trackThis(&nameDict, &count, "BLEVINS")
	trackThis(&nameDict, &count, "FRIEDMAN")
	trackThis(&nameDict, &count, "MOSES")
	trackThis(&nameDict, &count, "SOSA")
	trackThis(&nameDict, &count, "BLANCHARD")
	trackThis(&nameDict, &count, "HUBER")
	trackThis(&nameDict, &count, "FRYE")
	trackThis(&nameDict, &count, "KRUEGER")
	trackThis(&nameDict, &count, "BERNARD")
	trackThis(&nameDict, &count, "ROSARIO")
	trackThis(&nameDict, &count, "RUBIO")
	trackThis(&nameDict, &count, "MULLEN")
	trackThis(&nameDict, &count, "BENJAMIN")
	trackThis(&nameDict, &count, "HALEY")
	trackThis(&nameDict, &count, "CHUNG")
	trackThis(&nameDict, &count, "MOYER")
	trackThis(&nameDict, &count, "CHOI")
	trackThis(&nameDict, &count, "HORNE")
	trackThis(&nameDict, &count, "YU")
	trackThis(&nameDict, &count, "WOODWARD")
	trackThis(&nameDict, &count, "ALI")
	trackThis(&nameDict, &count, "NIXON")
	trackThis(&nameDict, &count, "HAYDEN")
	trackThis(&nameDict, &count, "RIVERS")
	trackThis(&nameDict, &count, "ESTES")
	trackThis(&nameDict, &count, "MCCARTY")
	trackThis(&nameDict, &count, "RICHMOND")
	trackThis(&nameDict, &count, "STUART")
	trackThis(&nameDict, &count, "MAYNARD")
	trackThis(&nameDict, &count, "BRANDT")
	trackThis(&nameDict, &count, "OCONNELL")
	trackThis(&nameDict, &count, "HANNA")
	trackThis(&nameDict, &count, "SANFORD")
	trackThis(&nameDict, &count, "SHEPPARD")
	trackThis(&nameDict, &count, "CHURCH")
	trackThis(&nameDict, &count, "BURCH")
	trackThis(&nameDict, &count, "LEVY")
	trackThis(&nameDict, &count, "RASMUSSEN")
	trackThis(&nameDict, &count, "COFFEY")
	trackThis(&nameDict, &count, "PONCE")
	trackThis(&nameDict, &count, "FAULKNER")
	trackThis(&nameDict, &count, "DONALDSON")
	trackThis(&nameDict, &count, "SCHMITT")
	trackThis(&nameDict, &count, "NOVAK")
	trackThis(&nameDict, &count, "COSTA")
	trackThis(&nameDict, &count, "MONTES")
	trackThis(&nameDict, &count, "BOOKER")
	trackThis(&nameDict, &count, "CORDOVA")
	trackThis(&nameDict, &count, "WALLER")
	trackThis(&nameDict, &count, "ARELLANO")
	trackThis(&nameDict, &count, "MADDOX")
	trackThis(&nameDict, &count, "MATA")
	trackThis(&nameDict, &count, "BONILLA")
	trackThis(&nameDict, &count, "STANTON")
	trackThis(&nameDict, &count, "COMPTON")
	trackThis(&nameDict, &count, "KAUFMAN")
	trackThis(&nameDict, &count, "DUDLEY")
	trackThis(&nameDict, &count, "MCPHERSON")
	trackThis(&nameDict, &count, "BELTRAN")
	trackThis(&nameDict, &count, "DICKSON")
	trackThis(&nameDict, &count, "MCCANN")
	trackThis(&nameDict, &count, "VILLEGAS")
	trackThis(&nameDict, &count, "PROCTOR")
	trackThis(&nameDict, &count, "HESTER")
	trackThis(&nameDict, &count, "CANTRELL")
	trackThis(&nameDict, &count, "DAUGHERTY")
	trackThis(&nameDict, &count, "CHERRY")
	trackThis(&nameDict, &count, "BRAY")
	trackThis(&nameDict, &count, "DAVILA")
	trackThis(&nameDict, &count, "ROWLAND")
	trackThis(&nameDict, &count, "MADDEN")
	trackThis(&nameDict, &count, "LEVINE")
	trackThis(&nameDict, &count, "SPENCE")
	trackThis(&nameDict, &count, "GOOD")
	trackThis(&nameDict, &count, "IRWIN")
	trackThis(&nameDict, &count, "WERNER")
	trackThis(&nameDict, &count, "KRAUSE")
	trackThis(&nameDict, &count, "PETTY")
	trackThis(&nameDict, &count, "WHITNEY")
	trackThis(&nameDict, &count, "BAIRD")
	trackThis(&nameDict, &count, "HOOPER")
	trackThis(&nameDict, &count, "POLLARD")
	trackThis(&nameDict, &count, "ZAVALA")
	trackThis(&nameDict, &count, "JARVIS")
	trackThis(&nameDict, &count, "HOLDEN")
	trackThis(&nameDict, &count, "HENDRIX")
	trackThis(&nameDict, &count, "HAAS")
	trackThis(&nameDict, &count, "MCGRATH")
	trackThis(&nameDict, &count, "BIRD")
	trackThis(&nameDict, &count, "LUCERO")
	trackThis(&nameDict, &count, "TERRELL")
	trackThis(&nameDict, &count, "RIGGS")
	trackThis(&nameDict, &count, "JOYCE")
	trackThis(&nameDict, &count, "ROLLINS")
	trackThis(&nameDict, &count, "MERCER")
	trackThis(&nameDict, &count, "GALLOWAY")
	trackThis(&nameDict, &count, "DUKE")
	trackThis(&nameDict, &count, "ODOM")
	trackThis(&nameDict, &count, "ANDERSEN")
	trackThis(&nameDict, &count, "DOWNS")
	trackThis(&nameDict, &count, "HATFIELD")
	trackThis(&nameDict, &count, "BENITEZ")
	trackThis(&nameDict, &count, "ARCHER")
	trackThis(&nameDict, &count, "HUERTA")
	trackThis(&nameDict, &count, "TRAVIS")
	trackThis(&nameDict, &count, "MCNEIL")
	trackThis(&nameDict, &count, "HINTON")
	trackThis(&nameDict, &count, "ZHANG")
	trackThis(&nameDict, &count, "HAYS")
	trackThis(&nameDict, &count, "MAYO")
	trackThis(&nameDict, &count, "FRITZ")
	trackThis(&nameDict, &count, "BRANCH")
	trackThis(&nameDict, &count, "MOONEY")
	trackThis(&nameDict, &count, "EWING")
	trackThis(&nameDict, &count, "RITTER")
	trackThis(&nameDict, &count, "ESPARZA")
	trackThis(&nameDict, &count, "FREY")
	trackThis(&nameDict, &count, "BRAUN")
	trackThis(&nameDict, &count, "GAY")
	trackThis(&nameDict, &count, "RIDDLE")
	trackThis(&nameDict, &count, "HANEY")
	trackThis(&nameDict, &count, "KAISER")
	trackThis(&nameDict, &count, "HOLDER")
	trackThis(&nameDict, &count, "CHANEY")
	trackThis(&nameDict, &count, "MCKNIGHT")
	trackThis(&nameDict, &count, "GAMBLE")
	trackThis(&nameDict, &count, "VANG")
	trackThis(&nameDict, &count, "COOLEY")
	trackThis(&nameDict, &count, "CARNEY")
	trackThis(&nameDict, &count, "COWAN")
	trackThis(&nameDict, &count, "FORBES")
	trackThis(&nameDict, &count, "FERRELL")
	trackThis(&nameDict, &count, "DAVIES")
	trackThis(&nameDict, &count, "BARAJAS")
	trackThis(&nameDict, &count, "SHEA")
	trackThis(&nameDict, &count, "OSBORN")
	trackThis(&nameDict, &count, "BRIGHT")
	trackThis(&nameDict, &count, "CUEVAS")
	trackThis(&nameDict, &count, "BOLTON")
	trackThis(&nameDict, &count, "MURILLO")
	trackThis(&nameDict, &count, "LUTZ")
	trackThis(&nameDict, &count, "DUARTE")
	trackThis(&nameDict, &count, "KIDD")
	trackThis(&nameDict, &count, "KEY")
	trackThis(&nameDict, &count, "COOKE")
}
