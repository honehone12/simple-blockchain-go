package common

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/json"
	"os"
)

func Encode(data interface{}) ([]byte, error) {
	buff := new(bytes.Buffer)
	encoder := json.NewEncoder(buff)
	err := encoder.Encode(data)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func Decode[T interface{}](bs []byte) (*T, error) {
	buff := new(bytes.Buffer)
	var data T
	buff.Write(bs)
	decoder := json.NewDecoder(buff)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func ToHex[T comparable](num T) ([]byte, error) {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func FromHex[T comparable](hex []byte) (T, error) {
	var num T
	err := binary.Read(bytes.NewBuffer(hex), binary.BigEndian, &num)
	return num, err
}

func FindAll[T interface{}](
	s []T, f func(e T) bool,
) []T {
	found := []T{}
	for _, elem := range s {
		if f(elem) {
			found = append(found, elem)
		}
	}
	return found
}

func ExistFile(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func QuickVerify(sig []byte, pubKey []byte, content []byte) bool {
	return ed25519.Verify(pubKey, content, sig)
}

func IsPowerOf2(n int) bool {
	return n != 0 && n&(n-1) == 0
}

func NextPowerOf2(n int) int {
	i := 0
	for {
		m := 1 << i
		if m >= n {
			return m
		}
		i++
	}
}

func LastPowerOf2(n int) int {
	i := 0
	l := 0
	for {
		m := 1 << i
		if m > n {
			break
		}
		l = m
		i++
	}
	return l
}
