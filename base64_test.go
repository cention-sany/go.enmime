package enmime

import (
	"bytes"
	"encoding/base64"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBase64Cleaner(t *testing.T) {
	input := strings.NewReader("\tA B\r\nC")
	cleaner := NewBase64Cleaner(input)
	buf := new(bytes.Buffer)
	buf.ReadFrom(cleaner)

	assert.Equal(t, "ABC", buf.String())
}

// Base64 combiner and padding reader testing part
type randomSplitter []byte

var split = []byte{'\t', '\r', '\n'}

func (r randomSplitter) Generate(rand *rand.Rand, size int) reflect.Value {
	s := rand.Int() % 8
	b := make([]byte, s)
	for i, _ := range b {
		b[i] = split[rand.Int()%3]
	}
	return reflect.ValueOf(randomSplitter(b))
}

func TestBase64PadCombinerRandom(t *testing.T) {
	f := func(as []string, split string) bool {
		aenc := make([]string, len(as))
		for i, s := range as {
			aenc[i] = base64.StdEncoding.EncodeToString([]byte(s))
		}
		whole := strings.Join(aenc, split)
		br := bytes.NewReader([]byte(whole))
		c := NewBase64Combiner(br)
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(c)
		if err != nil {
			t.Error("error in ReadFrom combiner")
			return false
		}
		whole = strings.Join(as, "")
		result := reflect.DeepEqual([]byte(whole), buf.Bytes())
		if !result {
			t.Error("\nwhole->", []byte(whole))
			t.Error("\nbytes->", buf.Bytes())
		}
		return result
	}
	c := quick.Config{
		Rand: rand.New(rand.NewSource(time.Now().Unix())),
		Values: func(v []reflect.Value, r *rand.Rand) {
			typ := reflect.TypeOf(randomSplitter{})
			as := make([]string, r.Intn(9))   // small integer value
			bs := make([]byte, r.Intn(23456)) // big integer value
			for j, _ := range as {
				for i, _ := range bs {
					bs[i] = byte(r.Int() % 0x100)
				}
				as[j] = string(bs)
			}
			v[0] = reflect.ValueOf(as)
			va, ok := quick.Value(typ, r)
			if ok {
				v[1] = reflect.ValueOf(string(va.Bytes()))
			} else {
				v[1] = reflect.ValueOf("")
			}
		},
	}
	if err := quick.Check(f, &c); err != nil {
		t.Error("failed on combiner black box test", err)
	}
}
