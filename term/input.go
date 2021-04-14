// +build linux

package term

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type InputOpt struct {
	Default  interface{}
	Echo     EchoMode
	Limit    uint8
	ConvFunc func(string) (interface{}, error)
}

var nilOpt = &InputOpt{}

func Input(prompt string, in interface{}, opt *InputOpt) error {
	checkIsTerminal()
	if opt == nil {
		opt = nilOpt
	}
	var s string
	var err error
	for {
		fmt.Print(prompt)
		s, err = GetString(opt.Echo, opt.Limit)
		fmt.Println()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		if s == "" {
			if opt.Default != nil {
				setValue(in, opt.Default)
				break
			} else {
				resetPrompt()
				continue
			}
		}
		if opt.ConvFunc == nil {
			_, err = fmt.Sscan(s, in)
			if err != nil {
				resetPrompt()
				continue
			}
			break
		} else {
			v, err := opt.ConvFunc(s)
			if err != nil {
				resetPrompt()
				continue
			}
			setValue(in, v)
			break
		}
	}
	return err
}

func setValue(in interface{}, v interface{}) {
	reflect.Indirect(reflect.ValueOf(in)).Set(reflect.ValueOf(v))
}

// ANSI escape codes: Cursor Up (CUU: ESC[A),
//                    Cursor Horizontal Absolute (CHA: ESC[G),
//                    Erase in Line (EL: ESC[K).

func resetPrompt() {
	fmt.Print("\x1b[A\x1b[G\x1b[K")
}

func moveCursorUp() {
	fmt.Print("\x1b[A")
}

func YesNo(prompt, options string) (bool, error) {
	checkIsTerminal()
	if len(options) != 2 {
		panic("exactly 2 options required")
	}
	prompt = fmt.Sprintf("%s [%s] ", strings.TrimRight(prompt, " "), options)
	idx, err := Select(prompt, options)
	if err != nil {
		return false, err
	}
	return idx == 0, nil
}

func Select(prompt, options string) (uint, error) {
	checkIsTerminal()
	opt := &InputOpt{Limit: 1}
	for i, r := range options {
		if unicode.IsUpper(r) {
			if opt.Default != nil {
				panic("only one default option allowed")
			}
			opt.Default = uint(i)
		}
	}
	options = strings.ToLower(options)
	opt.ConvFunc = func(s string) (interface{}, error) {
		s = strings.ToLower(s)
		i := strings.Index(options, s)
		if i < 0 {
			return 0, errors.New("")
		}
		return uint(i), nil
	}
	var idx uint
	err := Input(prompt, &idx, opt)
	return idx, err
}

const (
	menuFieldSep = " | "
	menuOptSep   = ") "
)

func Menu(prompt, title string, options []string, columns uint) (uint, error) {
	checkIsTerminal()
	width, height := getTermSize()
	optCnt := len(options)
	rowCnt, colCnt := getRowAndColCounts(optCnt, int(columns), height, title != "")
	maxIdxWidth := len(strconv.Itoa(optCnt))
	maxOptWidth := (width-len(menuFieldSep)*(colCnt-1))/colCnt - maxIdxWidth - len(menuOptSep)
	if w := getMaxOptionWidth(options); w < maxOptWidth {
		maxOptWidth = w
	}
	if title != "" {
		menuWidth := (maxIdxWidth+len(menuOptSep)+maxOptWidth)*colCnt + len(menuFieldSep)*(colCnt-1)
		fmt.Println(center(title, menuWidth))
		fmt.Println(strings.Repeat("=", maxInt(menuWidth, utf8.RuneCountInString(title))))
	}
	fmtStr := fmt.Sprintf("%%%dd) %%-%d.%ds", maxIdxWidth, maxOptWidth, maxOptWidth)
	for row := 0; row < rowCnt; row++ {
		for col := 0; col < colCnt; col++ {
			i := col*rowCnt + row
			if i >= optCnt {
				break
			}
			fmt.Printf(fmtStr, i+1, options[i])
			if col+1 < colCnt {
				fmt.Print(menuFieldSep)
			}
		}
		fmt.Println()
	}
	fmt.Println()
	moveCursorUp()
	opt := &InputOpt{}
	opt.ConvFunc = func(s string) (interface{}, error) {
		i, err := strconv.ParseUint(s, 10, 0)
		if err != nil {
			return 0, err
		}
		if i == 0 || i > uint64(optCnt) {
			return 0, errors.New("")
		}
		return uint(i - 1), nil
	}
	var idx uint
	err := Input(prompt, &idx, opt)
	return idx, err
}

func getRowAndColCounts(optCnt, columns, height int, withTitle bool) (int, int) {
	var rowCnt, colCnt int
	if columns == 0 {
		var h int
		if withTitle {
			h = 4
		} else {
			h = 2
		}
		if rowCnt = height - h; rowCnt > optCnt {
			rowCnt = optCnt
		}
		colCnt = optCnt / rowCnt
		if optCnt%rowCnt != 0 {
			colCnt += 1
		}
	} else {
		colCnt = columns
		rowCnt = optCnt / colCnt
		if optCnt%colCnt != 0 {
			rowCnt += 1
		}
	}
	return rowCnt, colCnt
}

func getTermSize() (int, int) {
	width, height, err := GetSize(os.Stdout.Fd())
	if err != nil {
		width = 80
		height = 24
	}
	return int(width), int(height)
}

func getMaxOptionWidth(options []string) int {
	var w int
	for _, s := range options {
		w = maxInt(w, utf8.RuneCountInString(s))
	}
	return w
}
