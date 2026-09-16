//go:build linux

package mister

import (
	"errors"
	"reflect"
	"testing"
)

func TestResizeRecovery(t *testing.T) {
	for _, failures := range []int{0, 1, 2} {
		var calls [][2]int
		err := resizeWithRollback(480, 270, 320, 240, func(w, h int) error {
			calls = append(calls, [2]int{w, h})
			if len(calls) <= failures {
				return errors.New("request or mapping failed")
			}
			return nil
		})
		want := [][2]int{{320, 240}}
		if failures > 0 {
			want = append(want, [2]int{480, 270})
		}
		if !reflect.DeepEqual(calls, want) || (err != nil) != (failures > 0) {
			t.Fatalf("failures%d calls%v err%v", failures, calls, err)
		}
	}
}
