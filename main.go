package main

import (
	"bytes"
	"embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const (
	BorderWidth   = 10
	StateBarWidth = 160
)

var windowWidth, windowHeight int

//go:embed assets/*
var fs embed.FS

type Cell struct {
	isMine    bool
	neighbor  int
	isOpen    bool
	isFlagged bool
}

type Board struct {
	screen   *ebiten.Image
	rows     int
	cols     int
	cellSize int
	cells    [][]Cell
	images   map[string]*ebiten.Image
	op       *ebiten.DrawImageOptions
	
	flags int
	mines int
	open  int
	
	isGameOver bool
	isWin      bool
}

func NewBoard(rows, cols, cellSize, mines int) *Board {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	b := &Board{
		screen:     ebiten.NewImage(rows*cellSize, cols*cellSize),
		rows:       rows,
		cols:       cols,
		cellSize:   cellSize,
		cells:      make([][]Cell, rows),
		flags:      mines,
		mines:      mines,
		open:       0,
		isGameOver: false,
		isWin:      false,
	}
	
	b.op = &ebiten.DrawImageOptions{}
	b.op.GeoM.Translate(BorderWidth, BorderWidth)
	
	// 初始化单元格
	for i := range b.cells {
		b.cells[i] = make([]Cell, cols)
	}
	
	// 放置地雷
	for i := 0; i < mines; {
		x, y := r.Intn(cols), r.Intn(rows)
		if !b.cells[y][x].isMine {
			b.cells[y][x].isMine = true
			i++
		}
	}
	
	// 计算相邻雷数
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if !b.cells[y][x].isMine {
				b.cells[y][x].neighbor = b.countNeighbors(x, y)
			}
		}
	}
	
	// 加载图片资源
	b.images = make(map[string]*ebiten.Image)
	load := func(name string) {
		data, _ := fs.ReadFile(fmt.Sprintf("assets/%s.png", name))
		img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		b.images[name] = img
	}
	
	for i := 0; i <= 8; i++ {
		load(strconv.Itoa(i))
	}
	load("unknown")
	load("mine")
	load("flag")
	
	return b
}

func (b *Board) countNeighbors(x, y int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < b.cols && ny >= 0 && ny < b.rows {
				if b.cells[ny][nx].isMine {
					count++
				}
			}
		}
	}
	return count
}

func (b *Board) Draw() {
	if b.isGameOver {
		b.screen.Clear()
		for y := 0; y < b.rows; y++ {
			for x := 0; x < b.cols; x++ {
				cell := b.cells[y][x]
				cx := x * b.cellSize
				cy := y * b.cellSize
				
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(cx), float64(cy))
				
				// 绘制数字、旗子或雷
				if cell.isFlagged {
					b.screen.DrawImage(b.images["flag"], op)
				}
				if cell.isMine {
					b.screen.DrawImage(b.images["mine"], op)
				} else if cell.neighbor >= 0 {
					b.screen.DrawImage(b.images[strconv.Itoa(cell.neighbor)], op)
				}
			}
		}
	} else {
		b.screen.Clear()
		for y := 0; y < b.rows; y++ {
			for x := 0; x < b.cols; x++ {
				cell := b.cells[y][x]
				cx := x * b.cellSize
				cy := y * b.cellSize
				
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(cx), float64(cy))
				
				if cell.isOpen {
					if cell.isMine {
						b.screen.DrawImage(b.images["mine"], op)
					} else if cell.neighbor >= 0 {
						b.screen.DrawImage(b.images[strconv.Itoa(cell.neighbor)], op)
					}
				} else {
					if cell.isFlagged {
						b.screen.DrawImage(b.images["flag"], op)
					} else {
						b.screen.DrawImage(b.images["unknown"], op)
					}
				}
			}
		}
	}
}

func (b *Board) openAndExpand(x, y int) {
	// 越界检查
	if x < 0 || x >= b.cols || y < 0 || y >= b.rows {
		return
	}
	
	cell := &b.cells[y][x]
	// 跳过已打开、有旗标、或地雷的格子
	if cell.isOpen || cell.isFlagged || cell.isMine {
		return
	}
	
	cell.isOpen = true
	b.open++
	
	// 只有周围无雷时才继续扩展
	if cell.neighbor == 0 {
		// 递归检查8个方向
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				b.openAndExpand(x+dx, y+dy)
			}
		}
	}
}

func main() {
	board := NewDefaultBoard()
	windowWidth = board.cols*board.cellSize + 2*BorderWidth + StateBarWidth
	windowHeight = board.rows*board.cellSize + 2*BorderWidth
	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowTitle("MineBuster")
	
	if err := ebiten.RunGame(&Game{board: board}); err != nil {
		log.Fatal(err)
	}
}

func NewDefaultBoard() *Board {
	return NewBoard(16, 16, 25, 40)
}

type Game struct {
	board         *Board
	prevLeftDown  bool
	prevRightDown bool
}

func (g *Game) Update() error {
	if g.board.isGameOver && !g.prevLeftDown && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.board = NewDefaultBoard()
		g.prevLeftDown = true
		return nil
	}
	
	mx, my := ebiten.CursorPosition()
	
	// 转换为棋盘坐标
	cx := (mx - BorderWidth) / g.board.cellSize
	cy := (my - BorderWidth) / g.board.cellSize
	
	// 检查坐标是否有效
	if cx >= 0 && cx < g.board.cols && cy >= 0 && cy < g.board.rows {
		cell := &g.board.cells[cy][cx]
		
		// 获取当前按键状态
		leftDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
		rightDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
		
		// 左键单击处理
		if leftDown && !g.prevLeftDown && !cell.isFlagged {
			if cell.isMine {
				// 踩中地雷
				cell.isOpen = true
				g.board.isGameOver = true
				g.board.isWin = false
			} else {
				// 安全区域自动扩展
				g.board.openAndExpand(cx, cy)
			}
		}
		
		// 右键标记处理
		if rightDown && !g.prevRightDown && !cell.isOpen {
			cell.isFlagged = !cell.isFlagged
			if cell.isFlagged {
				g.board.flags--
			} else {
				g.board.flags++
			}
		}
		
		// 保存当前按键状态
		g.prevLeftDown = leftDown
		g.prevRightDown = rightDown
		
		if !g.board.isGameOver && g.board.open == g.board.rows*g.board.cols-g.board.mines {
			g.board.isGameOver = true
			g.board.isWin = true
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.board.Draw()
	screen.DrawImage(g.board.screen, g.board.op)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Flags: %d", g.board.flags), g.board.cols*g.board.cellSize+BorderWidth+10, 20)
	if g.board.isGameOver {
		if g.board.isWin {
			ebitenutil.DebugPrintAt(screen, "You Win!\nClick anywhere to restart.", g.board.cols*g.board.cellSize+BorderWidth+10, 40)
		} else {
			ebitenutil.DebugPrintAt(screen, "Game Over!\nClick anywhere to restart.", g.board.cols*g.board.cellSize+BorderWidth+10, 40)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return windowWidth, windowHeight
}
