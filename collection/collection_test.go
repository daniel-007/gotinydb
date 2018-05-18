package collection

import (
	"os"
	"testing"

	internalTesting "gitea.interlab-net.com/alexandre/db/testing"
)

func insertAndCheck(t *testing.T, col *Collection, values []internalTesting.TestValue) {
	insert(t, col, values)
	if t.Failed() {
		return
	}

	ids := []string{}
	for _, value := range values {
		ids = append(ids, value.GetID())
	}

	check(t, col, ids, values)
}

func load(t *testing.T, col *Collection, values []internalTesting.TestValue) {
	loadedCol, loadErr := NewCollection(col.path)
	if loadErr != nil {
		t.Error(loadErr)
	}

	for _, value := range values {
		loadedValue1 := value.New()
		loadedValue2 := value.New()

		get1Err := col.Get(value.GetID(), loadedValue1)
		if get1Err != nil {
			t.Error(get1Err)
			return
		}
		get2Err := loadedCol.Get(value.GetID(), loadedValue2)
		if get2Err != nil {
			t.Error(get2Err)
			return
		}

		if !value.IsEqual(loadedValue1) || !value.IsEqual(loadedValue2) {
			t.Errorf("%v and %v are not equal to %v", loadedValue1, loadedValue2, value.GetContent())
			return
		}
	}
}

func updateAndCheck(t *testing.T, col *Collection, values []internalTesting.TestValue) {
	insert(t, col, values)
	if t.Failed() {
		return
	}

	ids := []string{}
	for i, value := range values {
		y := len(values) - 1 - i
		ids = append(ids, values[y].GetID())
		putErr := col.Put(value.GetID(), values[y].GetContent())
		if putErr != nil {
			t.Error(putErr)
			return
		}
	}

	check(t, col, ids, values)
}

func delete(t *testing.T, col *Collection, values []internalTesting.TestValue) {
	for _, value := range values {
		delErr := col.Delete(value.GetID())
		if delErr != nil {
			t.Error(delErr)
			return
		}
	}
}

func insert(t *testing.T, col *Collection, values []internalTesting.TestValue) {
	for _, value := range values {
		putErr := col.Put(value.GetID(), value.GetContent())
		if putErr != nil {
			t.Error(putErr)
			return
		}
	}
}

func check(t *testing.T, col *Collection, ids []string, values []internalTesting.TestValue) {
	for i, value := range values {
		gettedValue := value.New()
		getErr := col.Get(ids[i], gettedValue)
		if getErr != nil {
			t.Error(getErr)
			return
		}

		if !value.IsEqual(gettedValue) {
			t.Errorf("%v and %v are not equal", value.GetContent(), gettedValue)
			return
		}
	}
}

func runTest(t *testing.T, col *Collection, values []internalTesting.TestValue) {
	insertAndCheck(t, col, values)
	if t.Failed() {
		return
	}

	load(t, col, values)
	if t.Failed() {
		return
	}

	delete(t, col, values)
	if t.Failed() {
		return
	}

	updateAndCheck(t, col, values)
	if t.Failed() {
		return
	}
	delete(t, col, values)
	if t.Failed() {
		return
	}
}

func TestCollectionObject(t *testing.T) {
	defer os.RemoveAll(internalTesting.Path)
	col, newColErr := NewCollection(internalTesting.Path)
	if newColErr != nil {
		t.Error(newColErr)
		return
	}

	runTest(t, col, internalTesting.GetUsersExample())
}

func TestCollectionBin(t *testing.T) {
	defer os.RemoveAll(internalTesting.Path)
	col, newColErr := NewCollection(internalTesting.Path)
	if newColErr != nil {
		t.Error(newColErr)
		return
	}

	runTest(t, col, internalTesting.GetRawExample())
}
