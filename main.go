//go:generate goversioninfo
package main

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	Scale         = 2
	BorderWidth   = 10
	StateBarWidth = 160
	MaxBoardSize  = 40
)

var windowWidth, windowHeight int

//go:embed assets/*
var fs embed.FS

const FocusDelay = 100 * time.Millisecond

type Cell struct {
	isMine    bool
	neighbor  int
	isOpen    bool
	isFlagged bool
	isFocused bool
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

	startTime time.Time
	elapsed   time.Duration

	// 按钮相关字段
	isButtonPressed bool
	buttonX         int
	buttonY         int
	buttonWidth     int
	buttonHeight    int

	// 设置面板字段
	currentRows  int
	currentCols  int
	currentMines int

	// 输入状态
	isEditingRows  bool
	isEditingCols  bool
	isEditingMines bool
	inputBuffer    string
	editField      string // "rows", "cols", "mines", ""
	// 光标闪烁相关
	lastBlinkTime time.Time
	isCursorOn    bool

	// 设置面板位置
	settingsX int
	settingsY int
}

func NewBoard(rows, cols, cellSize, mines int) *Board {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := &Board{
		screen:     ebiten.NewImage(cols*cellSize, rows*cellSize),
		rows:       rows,
		cols:       cols,
		cellSize:   cellSize,
		cells:      make([][]Cell, rows),
		flags:      mines,
		mines:      mines,
		open:       0,
		isGameOver: false,
		isWin:      false,
		startTime:  time.Now(),
		elapsed:    0,

		// 设置面板初始化
		currentRows:    rows,
		currentCols:    cols,
		currentMines:   mines,
		isEditingRows:  false,
		isEditingCols:  false,
		isEditingMines: false,
		inputBuffer:    "",
		editField:      "",
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
		}
		b.images[name] = img
	}

	for i := 0; i <= 8; i++ {
		load(strconv.Itoa(i))
	}
	load("unknown")
	load("mine")
	load("flag")
	load("cross")
	load("focus")

	load("button")
	load("button_pressing")
	load("button_dead")
	load("button_dead_pressing")

	// 初始化按钮位置和大小
	b.buttonX = b.cols*b.cellSize + BorderWidth + 10
	b.buttonY = 20
	b.buttonWidth = b.images["button"].Bounds().Dx()
	b.buttonHeight = b.images["button"].Bounds().Dy()

	// 初始化设置面板位置
	b.settingsX = b.cols*b.cellSize + BorderWidth + 10
	b.settingsY = b.buttonY + b.buttonHeight + 50

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
	if b.isGameOver { // 游戏结束时显示所有格子
		b.screen.Clear()
		for y := 0; y < b.rows; y++ {
			for x := 0; x < b.cols; x++ {
				cell := b.cells[y][x]
				cx := x * b.cellSize
				cy := y * b.cellSize

				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(cx), float64(cy))

				// 绘制数字、旗子或雷
				if cell.isMine {
					b.screen.DrawImage(b.images["mine"], op)
				} else if cell.neighbor >= 0 {
					b.screen.DrawImage(b.images[strconv.Itoa(cell.neighbor)], op)
				}
				if cell.isFlagged {
					b.screen.DrawImage(b.images["cross"], op)
				}
			}
		}
	} else { // 游戏尚未结束
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
					} else if cell.isFocused {
						b.screen.DrawImage(b.images["focus"], op)
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

// 检查周围旗帜数量是否匹配
func (b *Board) checkSurroundFlags(x, y int) bool {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < b.cols && ny >= 0 && ny < b.rows {
				if b.cells[ny][nx].isFlagged {
					count++
				}
			}
		}
	}
	return count == b.cells[y][x].neighbor
}

// 展开周围未标记的格子
func (b *Board) expandAround(x, y int) {
	if b.checkSurroundFlags(x, y) {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx >= 0 && nx < b.cols && ny >= 0 && ny < b.rows {
					cell := &b.cells[ny][nx]
					if !cell.isFlagged && !cell.isOpen {
						if cell.isMine {
							cell.isOpen = true
							b.isGameOver = true
							b.isWin = false
							return
						}
						b.openAndExpand(nx, ny)
					}
				}
			}
		}
	} else {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx >= 0 && nx < b.cols && ny >= 0 && ny < b.rows {
					cell := &b.cells[ny][nx]
					if !cell.isOpen && !cell.isFlagged {
						cell.isFocused = true
						time.AfterFunc(FocusDelay, func() {
							cell.isFocused = false
						})
					}
				}
			}
		}
	}
}

func main() {
	board := NewDefaultBoard()
	windowWidth = board.cols*board.cellSize + 2*BorderWidth + StateBarWidth
	windowHeight = board.rows*board.cellSize + 2*BorderWidth
	ebiten.SetWindowSize(windowWidth*Scale, windowHeight*Scale)
	ebiten.SetWindowTitle("MineBuster")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

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
	mx, my := ebiten.CursorPosition()
	leftDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	rightDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)

	// 处理按钮点击事件
	isButtonArea := mx >= g.board.buttonX && mx <= g.board.buttonX+g.board.buttonWidth &&
		my >= g.board.buttonY && my <= g.board.buttonY+g.board.buttonHeight

	// 处理设置面板点击事件
	isSettingsArea := mx >= g.board.settingsX && mx <= g.board.settingsX+100 &&
		my >= g.board.settingsY && my <= g.board.settingsY+120

	// 计算设置项的点击区域
	rowsTextArea := mx >= g.board.settingsX && mx <= g.board.settingsX+100 &&
		my >= g.board.settingsY+20 && my <= g.board.settingsY+35

	colsTextArea := mx >= g.board.settingsX && mx <= g.board.settingsX+100 &&
		my >= g.board.settingsY+40 && my <= g.board.settingsY+55

	minesTextArea := mx >= g.board.settingsX && mx <= g.board.settingsX+100 &&
		my >= g.board.settingsY+60 && my <= g.board.settingsY+75

	// 光标闪烁逻辑
	if g.board.editField != "" {
		// 初始化闪烁时间
		if g.board.lastBlinkTime.IsZero() {
			g.board.lastBlinkTime = time.Now()
		}
		// 每500毫秒闪烁一次
		if time.Since(g.board.lastBlinkTime) > 500*time.Millisecond {
			g.board.isCursorOn = !g.board.isCursorOn
			g.board.lastBlinkTime = time.Now()
		}
	} else {
		// 非编辑状态下重置光标状态
		g.board.isCursorOn = false
		g.board.lastBlinkTime = time.Time{}
	}

	// 处理编辑状态
	if isSettingsArea && leftDown && !g.prevLeftDown {
		// 切换编辑字段前，保存当前输入
		if rowsTextArea && g.board.editField != "rows" {
			// 保存之前的输入
			g.saveInputAndExitEdit()
			// 开始编辑行
			g.board.isEditingRows = true
			g.board.isEditingCols = false
			g.board.isEditingMines = false
			g.board.editField = "rows"
			g.board.inputBuffer = strconv.Itoa(g.board.currentRows)
			// 重置光标状态
			g.board.isCursorOn = true
			g.board.lastBlinkTime = time.Now()
		} else if colsTextArea && g.board.editField != "cols" {
			// 保存之前的输入
			g.saveInputAndExitEdit()
			// 开始编辑列
			g.board.isEditingRows = false
			g.board.isEditingCols = true
			g.board.isEditingMines = false
			g.board.editField = "cols"
			g.board.inputBuffer = strconv.Itoa(g.board.currentCols)
			// 重置光标状态
			g.board.isCursorOn = true
			g.board.lastBlinkTime = time.Now()
		} else if minesTextArea && g.board.editField != "mines" {
			// 保存之前的输入
			g.saveInputAndExitEdit()
			// 开始编辑地雷数
			g.board.isEditingRows = false
			g.board.isEditingCols = false
			g.board.isEditingMines = true
			g.board.editField = "mines"
			g.board.inputBuffer = strconv.Itoa(g.board.currentMines)
			// 重置光标状态
			g.board.isCursorOn = true
			g.board.lastBlinkTime = time.Now()
		} else if !rowsTextArea && !colsTextArea && !minesTextArea {
			// 点击设置面板其他区域，取消编辑
			g.saveInputAndExitEdit()
		}
	}

	// 点击非设置区域，完成输入
	if leftDown && !g.prevLeftDown && !isSettingsArea && g.board.editField != "" {
		g.saveInputAndExitEdit()
	}

	// 处理键盘输入
	if g.board.editField != "" {
		// 获取释放的键
		for _, key := range inpututil.AppendJustReleasedKeys(nil) {
			switch {
			case key >= ebiten.Key0 && key <= ebiten.Key9:
				// 数字键
				g.board.inputBuffer += string(key - ebiten.Key0 + '0')
			case key >= ebiten.KeyKP0 && key <= ebiten.KeyKP9:
				// 数字小键盘键
				g.board.inputBuffer += string(key - ebiten.KeyKP0 + '0')
			case key == ebiten.KeyBackspace:
				// 退格键
				if len(g.board.inputBuffer) > 0 {
					g.board.inputBuffer = g.board.inputBuffer[:len(g.board.inputBuffer)-1]
				}
			case key == ebiten.KeyEnter || key == ebiten.KeyKPEnter:
				// 回车键确认
				g.saveInputAndExitEdit()
			case key == ebiten.KeyEscape:
				// ESC键取消，不保存输入
				g.board.isEditingRows = false
				g.board.isEditingCols = false
				g.board.isEditingMines = false
				g.board.editField = ""
				g.board.inputBuffer = ""
				g.board.isCursorOn = false
				g.board.lastBlinkTime = time.Time{}
			}
		}
	}
	if isButtonArea {
		ebiten.SetCursorShape(ebiten.CursorShapePointer)
		if leftDown && !g.prevLeftDown {
			// 按钮按下
			g.board.isButtonPressed = true
		} else if !leftDown && g.prevLeftDown {
			// 按钮释放，重启游戏
			// 自动确认输入
			g.saveInputAndExitEdit()
			g.board = NewBoard(g.board.currentRows, g.board.currentCols, 25, g.board.currentMines)
			// 更新窗口大小
			windowWidth = g.board.cols*g.board.cellSize + 2*BorderWidth + StateBarWidth
			windowHeight = g.board.rows*g.board.cellSize + 2*BorderWidth
			ebiten.SetWindowSize(windowWidth*Scale, windowHeight*Scale)
			return nil
		}
	} else {
		// 转换为棋盘坐标
		var cx = int(math.Floor(float64(mx-BorderWidth) / float64(g.board.cellSize)))
		var cy = int(math.Floor(float64(my-BorderWidth) / float64(g.board.cellSize)))

		// 检查坐标是否有效
		if cx >= 0 && cx < g.board.cols && cy >= 0 && cy < g.board.rows {
			ebiten.SetCursorShape(ebiten.CursorShapePointer)
			if !g.board.isGameOver {
				cell := &g.board.cells[cy][cx]

				if leftDown && !g.prevLeftDown && !cell.isFlagged {
					if cell.isMine {
						// 踩中地雷
						cell.isOpen = true
						g.board.isGameOver = true
						g.board.isWin = false
					} else if cell.isOpen {
						// 安全区域自动扩展
						g.board.expandAround(cx, cy)
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

				if !g.board.isGameOver && g.board.open == g.board.rows*g.board.cols-g.board.mines {
					g.board.isGameOver = true
					g.board.isWin = true
				}
			}
		} else {
			ebiten.SetCursorShape(ebiten.CursorShapeDefault)
		}
	}

	// 更新按钮状态
	if !leftDown {
		g.board.isButtonPressed = false
	}

	// 保存当前按键状态
	g.prevLeftDown = leftDown
	g.prevRightDown = rightDown
	return nil
}

// 保存当前输入值并取消编辑状态
func (g *Game) saveInputAndExitEdit() {
	if g.board.editField != "" {
		if val, err := strconv.Atoi(g.board.inputBuffer); err == nil {
			// 验证并设置值
			switch g.board.editField {
			case "rows":
				if val >= 2 && val <= MaxBoardSize {
					g.board.currentRows = val
				}
			case "cols":
				if val >= 2 && val <= MaxBoardSize {
					g.board.currentCols = val
				}
			case "mines":
				maxMines := g.board.currentRows*g.board.currentCols - 1
				if val >= 1 && val <= maxMines {
					g.board.currentMines = val
				}
			}
		}
		// 取消编辑状态
		g.board.isEditingRows = false
		g.board.isEditingCols = false
		g.board.isEditingMines = false
		g.board.editField = ""
		g.board.inputBuffer = ""
		g.board.isCursorOn = false
		g.board.lastBlinkTime = time.Time{}
	}
}
func (g *Game) Draw(screen *ebiten.Image) {
	g.board.Draw()
	screen.DrawImage(g.board.screen, g.board.op)

	// 绘制重启按钮
	buttonOp := &ebiten.DrawImageOptions{}
	buttonOp.GeoM.Translate(float64(g.board.buttonX), float64(g.board.buttonY))

	// 根据游戏状态选择按钮图片
	var buttonImage string
	if g.board.isButtonPressed {
		if g.board.isGameOver {
			buttonImage = "button_dead_pressing"
		} else {
			buttonImage = "button_pressing"
		}
	} else {
		if g.board.isGameOver {
			buttonImage = "button_dead"
		} else {
			buttonImage = "button"
		}
	}
	screen.DrawImage(g.board.images[buttonImage], buttonOp)

	// 调整文本位置，在按钮下方显示
	buttonBottom := g.board.buttonY + g.board.buttonHeight + 10
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Flags: %d", g.board.flags), g.board.cols*g.board.cellSize+BorderWidth+10, buttonBottom)

	if !g.board.isGameOver {
		g.board.elapsed = time.Since(g.board.startTime)
	}
	minutes := int(g.board.elapsed.Minutes())
	seconds := int(g.board.elapsed.Seconds()) % 60
	timerText := fmt.Sprintf("Time: %02d:%02d", minutes, seconds)
	ebitenutil.DebugPrintAt(screen, timerText, g.board.cols*g.board.cellSize+BorderWidth+10, buttonBottom+16)

	// 绘制设置面板
	ebitenutil.DebugPrintAt(screen, "Settings: ", g.board.settingsX, g.board.settingsY)

	// 行设置
	rowsText := fmt.Sprintf("Rows: %d", g.board.currentRows)
	if g.board.isEditingRows {
		if g.board.isCursorOn {
			rowsText = fmt.Sprintf("Rows: %s_", g.board.inputBuffer)
		} else {
			rowsText = fmt.Sprintf("Rows: %s", g.board.inputBuffer)
		}
	}
	ebitenutil.DebugPrintAt(screen, rowsText, g.board.settingsX, g.board.settingsY+20)

	// 列设置
	colsText := fmt.Sprintf("Cols: %d", g.board.currentCols)
	if g.board.isEditingCols {
		if g.board.isCursorOn {
			colsText = fmt.Sprintf("Cols: %s_", g.board.inputBuffer)
		} else {
			colsText = fmt.Sprintf("Cols: %s", g.board.inputBuffer)
		}
	}
	ebitenutil.DebugPrintAt(screen, colsText, g.board.settingsX, g.board.settingsY+40)

	// 雷数设置
	minesText := fmt.Sprintf("Mines: %d", g.board.currentMines)
	if g.board.isEditingMines {
		if g.board.isCursorOn {
			minesText = fmt.Sprintf("Mines: %s_", g.board.inputBuffer)
		} else {
			minesText = fmt.Sprintf("Mines: %s", g.board.inputBuffer)
		}
	}
	ebitenutil.DebugPrintAt(screen, minesText, g.board.settingsX, g.board.settingsY+60)

	// 操作提示
	ebitenutil.DebugPrintAt(screen, "Click number to edit", g.board.settingsX, g.board.settingsY+80)
	ebitenutil.DebugPrintAt(screen, "Press Enter to confirm, ", g.board.settingsX, g.board.settingsY+100)
	ebitenutil.DebugPrintAt(screen, "ESC to cancel", g.board.settingsX, g.board.settingsY+120)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return windowWidth, windowHeight
}
