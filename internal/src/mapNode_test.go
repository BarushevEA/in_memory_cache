package src

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestMapNode_SetRemoveCallback tests that the remove callback is correctly set and invoked when the node is removed.
func TestMapNode_SetRemoveCallback(t *testing.T) {
	var called bool
	removeFunc := func() { called = true }

	node := NewMapNode[int](42).(*MapNode[int])
	node.SetRemoveCallback(removeFunc)

	node.remove()
	assert.True(t, called, "remove callback should be called")
}

// TestMapNode_SetTTL tests the SetTTL method of MapNode to ensure it correctly updates the TTL and duration values.
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

// TestMapNode_SetTTLDecrement tests the SetTTLDecrement method by verifying the ttlDecrement field updates correctly.
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

// TestMapNode_Tick is a test function that validates the behavior of the Tick method in the MapNode implementation.
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

// TestMapNode_GetData tests the GetData function of MapNode, ensuring data retrieval resets TTL and handling various scenarios.
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
				node.Tick()
			}

			data := node.GetData()
			assert.Equal(t, tc.data, data, "data should match")
			assert.Equal(t, tc.expectedTTL, node.duration, "duration should reset to ttl after GetData")
		})
	}
}

// TestNewMapNode verifies the NewMapNode function by testing its behavior with various data types and initial state values.
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

// TestMapNode_MultipleTicks verifies the behavior of sequential TTL decrements and ensures the removal callback is triggered on expiration.
func TestMapNode_MultipleTicks(t *testing.T) {
	var callbackCalled bool
	node := NewMapNode[int](42).(*MapNode[int])
	node.SetRemoveCallback(func() { callbackCalled = true })

	node.SetTTL(5 * time.Second)
	node.SetTTLDecrement(1 * time.Second)

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

// TestMapNode_UpdateRemoveCallback verifies that the remove callback can be updated and only the latest callback is invoked.
func TestMapNode_UpdateRemoveCallback(t *testing.T) {
	var called1, called2 bool
	node := NewMapNode[int](42).(*MapNode[int])

	node.SetRemoveCallback(func() { called1 = true })

	node.SetRemoveCallback(func() { called2 = true })

	node.remove()

	assert.False(t, called1, "first callback should not be called")
	assert.True(t, called2, "second callback should be called")
}

// TestMapNode_SetData tests the behavior of the SetData method on a MapNode instance with various input data scenarios.
func TestMapNode_SetData(t *testing.T) {
	testCases := []struct {
		name        string
		initialData interface{}
		newData     interface{}
	}{
		{"update integer", 42, 84},
		{"update string", "hello", "world"},
		{"update to same value", 42, 42},
		{"update struct",
			struct{ value int }{42},
			struct{ value int }{84}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := NewMapNode(tc.initialData).(*MapNode[interface{}])
			assert.Equal(t, tc.initialData, node.data, "initial data should match")

			node.SetData(tc.newData)
			assert.Equal(t, tc.newData, node.data, "data should be updated")
		})
	}
}

// TestMapNode_SetDataWithTTL tests that the data of a MapNode can be updated without affecting its TTL or duration.
func TestMapNode_SetDataWithTTL(t *testing.T) {
	node := NewMapNode[int](42).(*MapNode[int])
	node.SetTTL(5 * time.Second)

	initialTTL := node.ttl
	initialDuration := node.duration

	node.SetData(84)

	assert.Equal(t, 84, node.data, "data should be updated")
	assert.Equal(t, initialTTL, node.ttl, "TTL should remain unchanged")
	assert.Equal(t, initialDuration, node.duration, "duration should remain unchanged")
}

// TestMapNode_ReferenceTypes tests the behavior of MapNode with various reference types, ensuring correctness and immutability.
func TestMapNode_ReferenceTypes(t *testing.T) {
	type TestStruct struct {
		Value int
		Data  string
	}

	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "pointer to struct",
			test: func(t *testing.T) {
				initial := &TestStruct{Value: 42, Data: "initial"}
				node := NewMapNode(initial).(*MapNode[*TestStruct])

				initial.Value = 100
				nodeData := node.GetData()
				assert.Equal(t, 100, nodeData.Value, "node should reference the same memory")

				newData := &TestStruct{Value: 200, Data: "new"}
				node.SetData(newData)
				assert.Equal(t, newData, node.GetData(), "node should store new pointer")
			},
		},
		{
			name: "slice",
			test: func(t *testing.T) {
				initial := []int{1, 2, 3}
				node := NewMapNode(initial).(*MapNode[[]int])

				initial[0] = 10
				nodeData := node.GetData()
				assert.Equal(t, 10, nodeData[0], "node should reference the same slice")

				newData := []int{4, 5, 6}
				node.SetData(newData)
				assert.Equal(t, newData, node.GetData(), "node should store new slice")
			},
		},
		{
			name: "map",
			test: func(t *testing.T) {
				initial := map[string]int{"a": 1, "b": 2}
				node := NewMapNode(initial).(*MapNode[map[string]int])

				initial["a"] = 10
				nodeData := node.GetData()
				assert.Equal(t, 10, nodeData["a"], "node should reference the same map")

				newData := map[string]int{"c": 3, "d": 4}
				node.SetData(newData)
				assert.Equal(t, newData, node.GetData(), "node should store new map")
			},
		},
		{
			name: "nested references",
			test: func(t *testing.T) {
				type NestedStruct struct {
					Ptr   *TestStruct
					Slice []int
					Map   map[string]int
				}

				initial := &NestedStruct{
					Ptr:   &TestStruct{Value: 42, Data: "test"},
					Slice: []int{1, 2, 3},
					Map:   map[string]int{"a": 1},
				}

				node := NewMapNode(initial).(*MapNode[*NestedStruct])

				initial.Ptr.Value = 100
				initial.Slice[0] = 10
				initial.Map["a"] = 10

				nodeData := node.GetData()
				assert.Equal(t, 100, nodeData.Ptr.Value, "nested pointer should be modified")
				assert.Equal(t, 10, nodeData.Slice[0], "nested slice should be modified")
				assert.Equal(t, 10, nodeData.Map["a"], "nested map should be modified")

				newData := &NestedStruct{
					Ptr:   &TestStruct{Value: 200, Data: "new"},
					Slice: []int{4, 5, 6},
					Map:   map[string]int{"b": 2},
				}
				node.SetData(newData)
				assert.Equal(t, newData, node.GetData(), "node should store new nested structure")
			},
		},
		{
			name: "nil values",
			test: func(t *testing.T) {
				var initial *TestStruct = nil
				node := NewMapNode(initial).(*MapNode[*TestStruct])

				assert.Nil(t, node.GetData(), "node should store nil pointer")

				newData := &TestStruct{Value: 42, Data: "new"}
				node.SetData(newData)
				assert.NotNil(t, node.GetData(), "node should store non-nil pointer")

				node.SetData(nil)
				assert.Nil(t, node.GetData(), "node should store nil after reset")
			},
		},
		{
			name: "interface values",
			test: func(t *testing.T) {
				var initial interface{} = &TestStruct{Value: 42, Data: "test"}
				node := NewMapNode(initial).(*MapNode[interface{}])

				if ptr, ok := node.GetData().(*TestStruct); ok {
					assert.Equal(t, 42, ptr.Value, "should preserve original value")
				} else {
					t.Error("failed to cast interface to original type")
				}

				newData := "string value"
				node.SetData(newData)
				assert.Equal(t, newData, node.GetData(), "should store new value of different type")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

// TestMapNode_Clear verifies the behavior of the Clear method, ensuring it resets all fields and removes callbacks.
func TestMapNode_Clear(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{

		{
			name: "clear and verify callback removal",
			test: func(t *testing.T) {
				callbackExecuted := false
				node := NewMapNode(42).(*MapNode[int])

				node.SetRemoveCallback(func() { callbackExecuted = true })
				node.SetTTL(5 * time.Second)
				node.SetTTLDecrement(1 * time.Second)

				node.duration = 0
				node.remove()
				assert.True(t, callbackExecuted, "callback should work before clear")

				callbackExecuted = false
				node.Clear()

				assert.Nil(t, node.remove, "callback should be nil after clear")

				assert.Equal(t, 0, node.data, "data should be zero")
				assert.Equal(t, time.Duration(0), node.ttl, "ttl should be zero")
				assert.Equal(t, time.Duration(0), node.duration, "duration should be zero")
				assert.Equal(t, time.Duration(0), node.ttlDecrement, "ttlDecrement should be zero")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func TestMapNode_Metrics(t *testing.T) {
	t.Run("should correctly track creation time", func(t *testing.T) {
		// Arrange
		beforeCreate := time.Now()
		node := NewMapNode("test")
		afterCreate := time.Now()

		// Act
		_, createdAt, _, _ := node.GetDataWithMetrics()

		// Assert
		assert.True(t, createdAt.After(beforeCreate) || createdAt.Equal(beforeCreate))
		assert.True(t, createdAt.Before(afterCreate) || createdAt.Equal(afterCreate))
	})

	t.Run("should correctly count get operations", func(t *testing.T) {
		// Arrange
		node := NewMapNode("test")
		expectedGetCount := uint32(5)

		// Act
		for i := uint32(0); i < expectedGetCount; i++ {
			node.GetData()
		}
		_, _, _, actualGetCount := node.GetDataWithMetrics()

		// Assert
		assert.Equal(t, expectedGetCount, actualGetCount)
	})

	t.Run("should correctly count set operations", func(t *testing.T) {
		// Arrange
		node := NewMapNode("initial")
		expectedSetCount := uint32(2)

		// Act
		for i := uint32(1); i < expectedSetCount; i++ {
			node.SetData("new value")
		}
		_, _, actualSetCount, _ := node.GetDataWithMetrics()

		// Assert
		assert.Equal(t, expectedSetCount, actualSetCount)
	})

	t.Run("should maintain separate get and set counts", func(t *testing.T) {
		// Arrange
		node := NewMapNode("test")
		expectedGetCount := uint32(3)
		expectedSetCount := uint32(3)

		// Act
		for i := uint32(0); i < expectedGetCount; i++ {
			node.GetData()
		}
		for i := uint32(1); i < expectedSetCount; i++ {
			node.SetData("new value")
		}
		_, _, setCount, getCount := node.GetDataWithMetrics()

		// Assert
		assert.Equal(t, expectedSetCount, setCount)
		assert.Equal(t, expectedGetCount, getCount)
	})

	t.Run("should reset metrics on Clear", func(t *testing.T) {
		// Arrange
		node := NewMapNode("test")
		node.GetData()
		node.SetData("new value")

		// Act
		node.Clear()
		_, createdAt, setCount, getCount := node.GetDataWithMetrics()

		// Assert
		assert.Equal(t, uint32(0), setCount)
		assert.Equal(t, uint32(0), getCount)
		assert.False(t, createdAt.IsZero())
	})

	t.Run("should preserve metrics across data updates", func(t *testing.T) {
		// Arrange
		node := NewMapNode("initial")
		time.Sleep(time.Millisecond) // Ensure some time passes

		// Act
		node.GetData()
		node.SetData("new value")
		data, createdAt, setCount, getCount := node.GetDataWithMetrics()

		// Assert
		assert.Equal(t, "new value", data)
		assert.True(t, createdAt.Before(time.Now()))
		assert.Equal(t, uint32(2), setCount)
		assert.Equal(t, uint32(1), getCount)
	})
}
