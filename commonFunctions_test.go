package gotinydb

// func TestTypeName(t *testing.T) {
// 	if StringIndex.TypeName() != "StringIndex" {
// 		t.Error("returned name is not correct")
// 		return
// 	}
// 	if IntIndex.TypeName() != "IntIndex" {
// 		t.Error("returned name is not correct")
// 		return
// 	}
// 	if TimeIndex.TypeName() != "TimeIndex" {
// 		t.Error("returned name is not correct")
// 		return
// 	}

// 	if IndexType(-1).TypeName() != "" {
// 		t.Error("returned name is not correct")
// 		return
// 	}
// }

// func TestBuildSelectorHash(t *testing.T) {
// 	selectors := [][]string{
// 		{"userName"},
// 		{"auth", "ssh"},
// 		{"email"},
// 	}
// 	expectedResults := []uint16{
// 		7087,
// 		48658,
// 		4996,
// 	}

// 	for i := range selectors {
// 		if ret := buildSelectorHash(selectors[i]); ret != expectedResults[i] {
// 			t.Errorf("wrong result expected %d but had %d", expectedResults[i], ret)
// 		}
// 	}

// }