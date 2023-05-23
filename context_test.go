package tel

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallers(t *testing.T) {
	a := callers()
	assert.Len(t, a, 4)
}

func L19() { callers() }
func L18() { L19() }
func L17() { L18() }
func L16() { L17() }
func L15() { L16() }
func L14() { L15() }
func L13() { L14() }
func L12() { L13() }
func L11() { L12() }
func L10() { L11() }
func L9()  { L10() }
func L8()  { L9() }
func L7()  { L8() }
func L6()  { L7() }
func L5()  { L6() }
func L4()  { L5() }
func L3()  { L4() }
func L2()  { L3() }
func L1()  { L2() }
func L0()  { L1() }
func BenchmarkL(b *testing.B) {
	L0()
}

func callersDirect() []string {
	var t = make([]string, 0, 100)
	for c := 1; c <= 100; c++ {
		_, file, line, ok := runtime.Caller(c)
		if !ok {
			break
		}
		t = append(t, fmt.Sprintf("%s:%d", file, line))
	}
	return t
}
func M19() { callersDirect() }
func M18() { M19() }
func M17() { M18() }
func M16() { M17() }
func M15() { M16() }
func M14() { M15() }
func M13() { M14() }
func M12() { M13() }
func M11() { M12() }
func M10() { M11() }
func M9()  { M10() }
func M8()  { M9() }
func M7()  { M8() }
func M6()  { M7() }
func M5()  { M6() }
func M4()  { M5() }
func M3()  { M4() }
func M2()  { M3() }
func M1()  { M2() }
func M0()  { M1() }
func BenchmarkM(b *testing.B) {
	M0()
}
