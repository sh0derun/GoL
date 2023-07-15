package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

type (
	cell struct {
		x     int
		y     int
		value int
	}

	coord struct {
		x uint16
		y uint16
	}

	ConsoleCursorInfo struct {
		size    uint32
		visible int32
	}
)

const (
	WIDTH  int = 20
	HEIGHT int = 40
)

var grid [HEIGHT][WIDTH]int

func handleErrno(result uintptr, errno syscall.Errno) {
	if int(result) == 0 {
		var err error
		if errno != 0 {
			err = error(errno)
		} else {
			err = syscall.EINVAL
		}
		panic(err)
	}
}

func setCursorVisibility(visible int32) {
	cursorInfo := ConsoleCursorInfo{
		size:    uint32(unsafe.Sizeof(ConsoleCursorInfo{})),
		visible: visible,
	}

	var kernel32 = syscall.NewLazyDLL("kernel32.dll")

	var procSetConsoleCursorInfo = kernel32.NewProc("SetConsoleCursorInfo")
	result, _, errno := syscall.SyscallN(
		procSetConsoleCursorInfo.Addr(),
		uintptr(syscall.Handle(os.Stdout.Fd())),
		uintptr(unsafe.Pointer(&cursorInfo)),
	)
	handleErrno(result, errno)
}

func moveCursor(position coord) {
	var kernel32 = syscall.NewLazyDLL("kernel32.dll")
	var procSetConsoleCursorPosition = kernel32.NewProc("SetConsoleCursorPosition")
	result, _, errno := syscall.SyscallN(
		procSetConsoleCursorPosition.Addr(),
		uintptr(syscall.Handle(os.Stdout.Fd())),
		uintptr(*(*int32) /*because 16+16*/ (unsafe.Pointer(&position))),
	)

	handleErrno(result, errno)
}

func clamp(n *int, l, h int) {
	if *n < l {
		*n = l
	}
	if *n > h {
		*n = h
	}
}

func color(r, g, b int) int {
	clamp(&r, 0, 5)
	clamp(&g, 0, 5)
	clamp(&b, 0, 5)
	return 16 + 36*r + 6*g + b
}

func printGrid(grid [HEIGHT][WIDTH]int) {
	for y := 0; y < HEIGHT; y++ {
		for x := 0; x < WIDTH; x++ {
			moveCursor(coord{uint16(y), uint16(x)})
			if grid[y][x] == 1 {
				fmt.Print("\033[38;5;", color(5, 2, 0), "m\u2726 \033[0m")
			} else {
				// fmt.Print("\033[38;5;", color(1, 1, 1), "m\u00B7 \033[0m")
			}
		}
	}
}

func gridSetupRandom(grid *[HEIGHT][WIDTH]int) {
	rand.Seed(time.Now().UnixNano())
	for y := 0; y < HEIGHT; y++ {
		for x := 0; x < WIDTH; x++ {
			(*grid)[y][x] = rand.Intn(2)
		}
	}
}

func gridSetupGlider(grid *[HEIGHT][WIDTH]int) {
	(*grid)[HEIGHT/2][WIDTH/2-1] = 1
	(*grid)[HEIGHT/2][WIDTH/2] = 1
	(*grid)[HEIGHT/2][WIDTH/2+1] = 1
	(*grid)[HEIGHT/2-1][WIDTH/2+1] = 1
	(*grid)[HEIGHT/2-2][WIDTH/2] = 1
}

func gridSetupPiheptominoToPulsar(grid *[HEIGHT][WIDTH]int) {
	(*grid)[HEIGHT/2-1][WIDTH/2] = 1
	(*grid)[HEIGHT/2+1][WIDTH/2] = 1
	(*grid)[HEIGHT/2-1][WIDTH/2-1] = 1
	(*grid)[HEIGHT/2][WIDTH/2-1] = 1
	(*grid)[HEIGHT/2+1][WIDTH/2-1] = 1
	(*grid)[HEIGHT/2-1][WIDTH/2+1] = 1
	(*grid)[HEIGHT/2+1][WIDTH/2+1] = 1
}

func getNeighbours(y int, x int) (int, []cell) {
	sum := 0
	neighbours := []cell{}
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			if i == 0 && j == 0 {
				continue
			}
			row := y + j
			if row < 0 {
				row = HEIGHT - 1
			} else if row >= HEIGHT {
				row = 0
			}
			col := x + i
			if col < 0 {
				col = WIDTH - 1
			} else if col >= WIDTH {
				col = 0
			}
			sum += grid[row][col]
			neighbours = append(neighbours, cell{col, row, grid[row][col]})
		}
	}
	return sum, neighbours
}

func calculateNextState(grid *[HEIGHT][WIDTH]int) {
	var tmpGrid [HEIGHT][WIDTH]int
	for y := 0; y < HEIGHT; y++ {
		for x := 0; x < WIDTH; x++ {
			sum, _ := getNeighbours(y, x)
			if grid[y][x] == 1 && sum < 2 {
				tmpGrid[y][x] = 0
			} else if grid[y][x] == 1 && (sum == 2 || sum == 3) {
				tmpGrid[y][x] = 1
			} else if grid[y][x] == 0 && sum == 3 {
				tmpGrid[y][x] = 1
			} else {
				tmpGrid[y][x] = 0
			}
		}
	}
	*grid = tmpGrid
}

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		setCursorVisibility(1)
		os.Exit(0)
	}()

	setCursorVisibility(0)
	gridSetupPiheptominoToPulsar(&grid)
	for {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
		printGrid(grid)
		calculateNextState(&grid)
		time.Sleep(time.Nanosecond)
	}
}
