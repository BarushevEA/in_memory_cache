package src

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMapNode_SetRemoveCallback(t *testing.T) {
	var called bool
	removeFunc := func() { called = true }

	node := NewMapNode[int](42).(*MapNode[int])
	node.SetRemoveCallback(removeFunc)

	node.remove()
	assert.True(t, called, "remove callback should be called")
}

func TestMapNode_SetTTL(t *testing.T) {
	node := NewMapNode[int](42).(*MapNode[int])

	testCases := []struct {
		name   string
		input  time.Duration
		expTTL time.Duration
	}{
		{"set positive ttl", 5 * time.Second, 5 * time.Second},
		{"set zero ttl", 0, 0},
		{"set negative ttl", -1 * time.Second, -1 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node.SetTTL(tc.input)
			assert.Equal(t, tc.expTTL, node.ttl, "ttl should match")
			assert.Equal(t, tc.expTTL, node.duration, "duration should match")
		})
	}
}

func TestMapNode_SetTTLDecrement(t *testing.T) {
	node := NewMapNode[int](42).(*MapNode[int])

	testCases := []struct {
		name   string
		input  time.Duration
		expDec time.Duration
	}{
		{"set positive decrement", 1 * time.Second, 1 * time.Second},
		{"set zero decrement", 0, 0},
		{"set negative decrement", -1 * time.Second, -1 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node.SetTTLDecrement(tc.input)
			assert.Equal(t, tc.expDec, node.ttlDecrement, "TTL decrement should match")
		})
	}
}

func TestMapNode_Tick(t *testing.T) {
	var callbackCalled bool
	node := NewMapNode[int](42).(*MapNode[int])
	node.SetRemoveCallback(func() { callbackCalled = true })

	testCases := []struct {
		name         string
		initialTTL   time.Duration
		decrement    time.Duration
		expDuration  time.Duration
		shouldRemove bool
	}{
		{"duration reduces but not expired", 5 * time.Second, 1 * time.Second, 4 * time.Second, false},
		{"duration expires exactly", 2 * time.Second, 2 * time.Second, 0, true},
		{"duration already expired", 1 * time.Second, 5 * time.Second, -4 * time.Second, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callbackCalled = false
			node.SetTTL(tc.initialTTL)
			node.SetTTLDecrement(tc.decrement)

			node.Tick()
			assert.Equal(t, tc.expDuration, node.duration, "duration should match after tick")

			if tc.shouldRemove {
				assert.True(t, callbackCalled, "remove callback should be called")
			} else {
				assert.False(t, callbackCalled, "remove callback should not be called")
			}
		})
	}
}

func TestMapNode_GetData(t *testing.T) {
	testCases := []struct {
		name        string
		data        string
		initialTTL  time.Duration
		expectedTTL time.Duration
	}{
		{"reset duration on access", "test1", 5 * time.Second, 5 * time.Second},
		{"reset zero duration", "test2", 0, 0},
		{"reset negative duration", "test3", -1 * time.Second, -1 * time.Second},
		{"get data after tick", "test4", 5 * time.Second, 5 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := NewMapNode(tc.data).(*MapNode[string])
			node.SetTTL(tc.initialTTL)

			if tc.name == "get data after tick" {
				node.SetTTLDecrement(1 * time.Second)
				node.Tick() // уменьшаем duration
			}

			data := node.GetData()
			assert.Equal(t, tc.data, data, "data should match")
			assert.Equal(t, tc.expectedTTL, node.duration, "duration should reset to ttl after GetData")
		})
	}
}

func TestNewMapNode(t *testing.T) {
	testCases := []struct {
		name     string
		data     interface{}
		expected interface{}
	}{
		{"integer data", 42, 42},
		{"string data", "test", "test"},
		{"struct data", struct{ value int }{42}, struct{ value int }{42}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := NewMapNode(tc.data).(*MapNode[interface{}])
			assert.Equal(t, tc.expected, node.data)
			assert.Zero(t, node.ttl)
			assert.Zero(t, node.duration)
			assert.Zero(t, node.ttlDecrement)
			assert.Nil(t, node.remove)
		})
	}
}

func TestMapNode_MultipleTicks(t *testing.T) {
	var callbackCalled bool
	node := NewMapNode[int](42).(*MapNode[int])
	node.SetRemoveCallback(func() { callbackCalled = true })

	node.SetTTL(5 * time.Second)
	node.SetTTLDecrement(1 * time.Second)

	// Проверяем последовательное уменьшение duration
	expectedDurations := []time.Duration{4 * time.Second, 3 * time.Second, 2 * time.Second, 1 * time.Second, 0}

	for i, expected := range expectedDurations {
		node.Tick()
		assert.Equal(t, expected, node.duration, "duration should match after tick %d", i+1)

		if expected == 0 {
			assert.True(t, callbackCalled, "remove callback should be called on last tick")
		} else {
			assert.False(t, callbackCalled, "remove callback should not be called before last tick")
		}
	}
}

func TestMapNode_UpdateRemoveCallback(t *testing.T) {
	var called1, called2 bool
	node := NewMapNode[int](42).(*MapNode[int])

	// Устанавливаем первый callback
	node.SetRemoveCallback(func() { called1 = true })

	// Обновляем на второй callback
	node.SetRemoveCallback(func() { called2 = true })

	node.remove()

	assert.False(t, called1, "first callback should not be called")
	assert.True(t, called2, "second callback should be called")
}
