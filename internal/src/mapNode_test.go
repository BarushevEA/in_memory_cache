package src

import (
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

// TestMapNode_SetRemoveCallback tests that the remove callback is correctly set and invoked when the node is removed.
func TestMapNode_SetRemoveCallback(t *testing.T) {
	var called bool
	removeFunc := func() { called = true }

	node := NewMapNode[int](42)
	node.SetRemoveCallback(removeFunc)

	// Note: Directly calling node.remove() here to test the callback setting
	node.remove()
	assert.True(t, called, "remove callback should be called")
}

// TestMapNode_SetTTL tests the SetTTL method of MapNode to ensure it correctly updates the TTL and duration values.
func TestMapNode_SetTTL(t *testing.T) {
	node := NewMapNode[int](42)

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
			assert.Equal(t, tc.expTTL, node.ttl)
			assert.Equal(t, tc.expTTL, node.duration)
		})
	}
}

// TestMapNode_SetTTLDecrement tests the SetTTLDecrement method of MapNode.
func TestMapNode_SetTTLDecrement(t *testing.T) {
	node := NewMapNode[int](42)

	testCases := []struct {
		name         string
		input        time.Duration
		expDecrement time.Duration
	}{
		{"set positive decrement", 1 * time.Second, 1 * time.Second},
		{"set zero decrement", 0, 0},
		{"set negative decrement", -1 * time.Second, -1 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node.SetTTLDecrement(tc.input)
			assert.Equal(t, tc.expDecrement, node.ttlDecrement)
		})
	}
}

// TestMapNode_Tick validates the behavior of the Tick method in the MapNode implementation.
func TestMapNode_Tick(t *testing.T) {
	t.Run("duration reduces but not expired", func(t *testing.T) {
		var callbackCalled bool
		node := NewMapNode[int](42)
		node.SetRemoveCallback(func() { callbackCalled = true })
		node.SetTTL(5 * time.Second)
		node.SetTTLDecrement(1 * time.Second)
		// node.isDeleted.Store(false) // NewMapNode already sets it to false

		node.Tick() // Call Tick

		assert.Equal(t, 4*time.Second, node.duration, "duration should match after tick")
		assert.False(t, callbackCalled, "remove callback should not be called")
		assert.False(t, node.IsDeleted(), "node should not be marked as deleted by Tick method itself")
		assert.NotNil(t, node.remove, "remove callback should still be set by Tick method itself")
	})

	t.Run("duration expires exactly", func(t *testing.T) {
		var callbackCalled bool
		node := NewMapNode[int](42)
		node.SetRemoveCallback(func() { callbackCalled = true })
		node.SetTTL(2 * time.Second)
		node.SetTTLDecrement(2 * time.Second)
		// node.isDeleted.Store(false) // NewMapNode already sets it to false

		node.Tick() // Call Tick

		assert.Equal(t, time.Duration(0), node.duration, "duration should be 0 after expiration")
		assert.True(t, callbackCalled, "remove callback should be called on expiration")
		assert.False(t, node.IsDeleted(), "node should not be marked as deleted by Tick method itself")
		assert.NotNil(t, node.remove, "remove callback should still be set by Tick method itself")
	})

	t.Run("duration already expired", func(t *testing.T) {
		var callbackCalled bool
		node := NewMapNode[int](42)
		node.SetRemoveCallback(func() { callbackCalled = true })
		node.SetTTL(1 * time.Second)
		node.SetTTLDecrement(5 * time.Second)
		// node.isDeleted.Store(false) // NewMapNode already sets it to false

		node.Tick() // Call Tick

		assert.Equal(t, -4*time.Second, node.duration, "duration should be negative after expiration")
		assert.True(t, callbackCalled, "remove callback should be called on expiration")
		assert.False(t, node.IsDeleted(), "node should not be marked as deleted by Tick method itself")
		assert.NotNil(t, node.remove, "remove callback should still be set by Tick method itself")
	})

	t.Run("no remove callback set", func(t *testing.T) {
		node := NewMapNode[int](42)
		node.SetTTL(1 * time.Second)
		node.SetTTLDecrement(5 * time.Second)
		// node.isDeleted.Store(false) // NewMapNode already sets it to false

		node.Tick() // Call Tick

		// If remove == nil, callbackCalled will not be changed.
		// Assertions should reflect that nothing happened except duration change.
		assert.Equal(t, -4*time.Second, node.duration, "duration should be negative even without callback")
		assert.False(t, node.IsDeleted(), "node should not be marked as deleted")
		assert.Nil(t, node.remove, "remove callback should remain nil")
	})
}

// TestMapNode_MultipleTicks verifies the behavior of sequential TTL decrements
// and ensures the removal callback is triggered on expiration, understanding that
// MapNode.Tick() itself does not clear the callback.
func TestMapNode_MultipleTicks(t *testing.T) {
	var callbackCalledCount atomic.Int32 // Used for atomic increment

	node := NewMapNode[int](42)
	node.SetRemoveCallback(func() {
		callbackCalledCount.Add(1) // Increment atomically
	})

	node.SetTTL(5 * time.Second)
	node.SetTTLDecrement(1 * time.Second)
	// node.isDeleted.Store(false) // NewMapNode already sets it to false

	expectedDurations := []time.Duration{4 * time.Second, 3 * time.Second, 2 * time.Second, 1 * time.Second, 0}

	for i, expected := range expectedDurations {
		node.Tick() // Call Tick

		assert.Equal(t, expected, node.duration, "duration should match after tick %d", i+1)

		// Check callback call count based on current state of duration
		if expected <= 0 { // Callback should have been called at this point (or already)
			assert.Equal(t, int32(1), callbackCalledCount.Load(), "remove callback should be called once per expiration event (at least)")
			callbackCalledCount.Store(0) // Reset for the next check of a single event, if the loop continues
		} else { // Callback should not have been called yet
			assert.Equal(t, int32(0), callbackCalledCount.Load(), "remove callback should not be called before expiration")
		}

		assert.False(t, node.IsDeleted(), "node should not be marked as deleted by Tick method itself")
		assert.NotNil(t, node.remove, "remove callback should still be set, as MapNode.Tick() does not clear it")
	}

	// Additional tick after expiration: Callback will be called again because MapNode does not clear it itself.
	node.Tick()
	assert.Equal(t, -1*time.Second, node.duration, "duration should continue to decrease after expiration")
	// The expectation here is that the callback was called *again* after the previous reset, so it should be 1.
	assert.Equal(t, int32(1), callbackCalledCount.Load(), "remove callback should be called again because MapNode does not clear it itself")
	assert.False(t, node.IsDeleted(), "node should not be marked as deleted by Tick method itself")
	assert.NotNil(t, node.remove, "remove callback should still be set, as MapNode.Tick() does not clear it")
}

// TestMapNode_GetData tests the GetData method of MapNode.
func TestMapNode_GetData(t *testing.T) {
	node := NewMapNode[int](42)
	node.SetTTL(10 * time.Second)
	node.duration = 5 * time.Second // Simulate a reduced duration

	retrievedData := node.GetData()

	assert.Equal(t, 42, retrievedData, "retrieved data should match original data")
	assert.Equal(t, 10*time.Second, node.duration, "duration should reset to TTL after GetData")
	assert.Equal(t, uint32(1), node.getCount, "get count should be 1 after GetData")
}

// TestMapNode_SetData tests the SetData method of MapNode.
func TestMapNode_SetData(t *testing.T) {
	node := NewMapNode[string]("initial")
	node.SetTTL(10 * time.Second)
	node.duration = 5 * time.Second // Simulate a reduced duration

	node.SetData("new value")

	assert.Equal(t, "new value", node.data, "data should be updated")
	assert.Equal(t, 10*time.Second, node.duration, "duration should reset to TTL after SetData")
	assert.Equal(t, uint32(2), node.setCount, "set count should be 2 after SetData (1 from NewMapNode + 1 from SetData)")
}

// TestMapNode_Clear tests the Clear method of MapNode.
func TestMapNode_Clear(t *testing.T) {
	node := NewMapNode[string]("test data")
	node.SetTTL(5 * time.Second)
	node.SetTTLDecrement(1 * time.Second)
	node.SetRemoveCallback(func() {}) // Set a dummy callback
	node.isDeleted.Store(true)        // Set to true to test reset

	node.Clear()

	assert.Equal(t, time.Duration(0), node.duration, "duration should be 0 after clear")
	assert.Nil(t, node.remove, "remove callback should be nil after clear")
	assert.Equal(t, time.Duration(0), node.ttl, "ttl should be 0 after clear")
	assert.Equal(t, time.Duration(0), node.ttlDecrement, "ttlDecrement should be 0 after clear")
	var zeroVal string
	assert.Equal(t, zeroVal, node.data, "data should be zero value after clear")
	assert.Equal(t, uint32(0), node.setCount, "setCount should be 0 after clear")
	assert.Equal(t, uint32(0), node.getCount, "getCount should be 0 after clear")
	assert.False(t, node.IsDeleted(), "isDeleted should be false after clear") // Crucial assertion
}

// TestMapNode_GetMetrics verifies that GetDataWithMetrics returns the correct data, creation time, and counts.
func TestMapNode_GetMetrics(t *testing.T) {
	// Arrange
	node := NewMapNode("test")
	time.Sleep(10 * time.Millisecond) // Simulate some time passing
	expectedCreationTime := node.createdAt
	node.SetData("new value") // Increments setCount
	node.GetData()            // Increments getCount

	// Act
	actualData, actualCreatedAt, actualSetCount, actualGetCount := node.GetDataWithMetrics()

	// Assert
	assert.Equal(t, "new value", actualData)
	assert.Equal(t, expectedCreationTime, actualCreatedAt)
	assert.Equal(t, uint32(2), actualSetCount) // 1 from NewMapNode + 1 from SetData
	assert.Equal(t, uint32(1), actualGetCount)
}

func TestMapNode_SetCount(t *testing.T) {
	t.Run("should increment set count on NewMapNode", func(t *testing.T) {
		// Arrange & Act
		node := NewMapNode("test")
		_, _, actualSetCount, _ := node.GetDataWithMetrics()

		// Assert
		assert.Equal(t, uint32(1), actualSetCount)
	})

	t.Run("should increment set count on SetData", func(t *testing.T) {
		// Arrange
		node := NewMapNode("test")
		expectedSetCount := uint32(3) // Initial 1 + 2 more calls

		// Act
		node.SetData("new value 1")
		node.SetData("new value 2")
		_, _, actualSetCount, _ := node.GetDataWithMetrics()

		// Assert
		assert.Equal(t, expectedSetCount, actualSetCount)
	})

	t.Run("should maintain separate get and set counts", func(t *testing.T) {
		// Arrange
		node := NewMapNode("test")
		expectedGetCount := uint32(3)
		expectedSetCount := uint32(3) // Expected total: 1 from New + 2 from SetData calls

		// Act
		for i := uint32(0); i < expectedGetCount; i++ {
			node.GetData()
		}
		// Change: Make the loop explicitly run (expectedSetCount - 1) times for SetData calls
		// If expectedSetCount is 3, this loop runs 2 times (for i=0 and i=1)
		for i := uint32(0); i < expectedSetCount-1; i++ {
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
		assert.False(t, createdAt.IsZero()) // createdAt is not reset by Clear
	})
}
