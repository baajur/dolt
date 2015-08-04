package types

import (
	"testing"

	"github.com/attic-labs/noms/chunks"
	"github.com/stretchr/testify/assert"
)

var (
	testVals = []Value{
		Bool(true),
		Int16(1),
		Int32(1),
		Int64(1),
		UInt16(1),
		UInt32(1),
		UInt64(1),
		Float32(1),
		Float64(1),
		NewString("hi"),
		newBlobLeaf([]byte("hi")),
		// compoundBlob
		NewSet(NewString("hi")),
		NewList(NewString("hi")),
		NewMap(NewString("hi"), NewString("hi")),
	}
)

func isEncodedOutOfLine(v Value) int {
	switch v.(type) {
	case blobLeaf, compoundBlob, Set, List, Map:
		return 1
	}
	return 0
}

func TestIncrementalLoadList(t *testing.T) {
	assert := assert.New(t)
	cs := &chunks.TestStore{}

	expected := NewList(testVals...)
	ref, err := WriteValue(expected, cs)
	assert.NoError(err)

	actualVar, err := ReadValue(ref, cs)
	assert.NoError(err)
	actual := actualVar.(List)

	expectedCount := cs.Reads
	assert.Equal(1, expectedCount)
	for i := uint64(0); i < expected.Len(); i++ {
		v := actual.Get(i)
		assert.True(expected.Get(i).Equals(v))

		expectedCount += isEncodedOutOfLine(v)
		assert.Equal(expectedCount, cs.Reads)

		// Do it again to make sure multiple derefs don't do multiple loads.
		v = actual.Get(i)
		assert.Equal(expectedCount, cs.Reads)
	}
}

func TestIncrementalLoadSet(t *testing.T) {
	assert := assert.New(t)
	cs := &chunks.TestStore{}

	expected := NewSet(testVals...)
	ref, err := WriteValue(expected, cs)
	assert.NoError(err)

	actualVar, err := ReadValue(ref, cs)
	assert.NoError(err)
	actual := actualVar.(Set)

	expectedCount := cs.Reads
	assert.Equal(1, expectedCount)
	actual.Iter(func(v Value) (stop bool) {
		expectedCount += isEncodedOutOfLine(v)
		assert.Equal(expectedCount, cs.Reads)
		return
	})
}

func TestIncrementalLoadMap(t *testing.T) {
	assert := assert.New(t)
	cs := &chunks.TestStore{}

	expected := NewMap(testVals...)
	ref, err := WriteValue(expected, cs)
	assert.NoError(err)

	actualVar, err := ReadValue(ref, cs)
	assert.NoError(err)
	actual := actualVar.(Map)

	expectedCount := cs.Reads
	assert.Equal(1, expectedCount)
	actual.Iter(func(k, v Value) (stop bool) {
		expectedCount += isEncodedOutOfLine(k)
		expectedCount += isEncodedOutOfLine(v)
		assert.Equal(expectedCount, cs.Reads)
		return
	})
}

func TestIncrementalAddRef(t *testing.T) {
	assert := assert.New(t)
	cs := &chunks.TestStore{}

	expectedItem := UInt32(42)
	ref, err := WriteValue(expectedItem, cs)
	assert.NoError(err)

	expected := NewList(Ref{ref})
	ref, err = WriteValue(expected, cs)
	actualVar, err := ReadValue(ref, cs)
	assert.NoError(err)

	assert.Equal(1, cs.Reads)
	assert.True(expected.Equals(actualVar))

	actual := actualVar.(List)
	actualItem := actual.Get(0)
	assert.Equal(2, cs.Reads)
	assert.True(expectedItem.Equals(actualItem))

	// do it again to make sure caching works.
	actualItem = actual.Get(0)
	assert.Equal(2, cs.Reads)
	assert.True(expectedItem.Equals(actualItem))
}
