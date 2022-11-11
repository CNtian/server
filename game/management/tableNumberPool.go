package management

/*
	rand.Seed(time.Now().UnixNano())
	const roomNumberSize = 10000
	constCharNumber := []uint8{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}

	this.idleRoomNumberMap = make(map[int32]int32)
	textRoomNumber := make([]uint8, 6)

	var lastRoomNumber int64 = 0
	for i := 0; i < roomNumberSize; {

		randNum := rand.Int31n(10)
		if randNum != int32(len(constCharNumber)-1) {
			randNum += 1
		}
		textRoomNumber[0] = constCharNumber[randNum]
		for j := 1; j < 6; j++ {
			textRoomNumber[j] = constCharNumber[rand.Int31n(10)]
		}

		temp, _ := strconv.ParseInt(string(textRoomNumber), 10, 32)

		_, ok := this.idleRoomNumberMap[int32(temp)]
		if !ok {
			this.idleRoomNumberMap[int32(temp)] = int32(temp)
			i++
			lastRoomNumber = temp
		} else {
			rand.Seed(time.Now().UnixNano() + lastRoomNumber)
		}
	}
*/

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"qpGame/localConfig"
	"strconv"
	"sync"
)

type TableNumberPool struct {
	m               sync.Mutex
	curIndex        int
	tableNumberPool []int32
}

var tableNumberPool TableNumberPool

func Init(cfg *localConfig.TableNumberRange) error {

	tableNumberPool.tableNumberPool = make([]int32, 0, 4096)

	file, err := os.Open("./tableNumber.list")
	if err != nil {
		return err
	}
	defer file.Close()

	tableNumberArr := make([]string, 0, cfg.End-cfg.Begin)
	br := bufio.NewReader(file)

	i, j := 0, 0
	for ; i < cfg.Begin; i++ {
		_, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
	}

	for ; i < cfg.End; i++ {
		text, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		tableNumberArr = append(tableNumberArr, string(text))
		j++
	}

	if len(tableNumberArr) < cfg.End-cfg.Begin {
		return fmt.Errorf("int32(len(beans.Roomlist)) < cfg.RoomNumberEndIndex.%d<%d", len(tableNumberArr), cfg.End)
	}

	for j = 0; j < len(tableNumberArr); j++ {
		var roomNumber int
		roomNumber, err = strconv.Atoi(tableNumberArr[j])
		if err != nil {
			return fmt.Errorf("strconv.ParseInt() err.err:=%s,text:=%s", err.Error(), tableNumberArr[j])
		}
		tableNumberPool.tableNumberPool = append(tableNumberPool.tableNumberPool, int32(roomNumber))
	}

	tableNumberPool.curIndex = rand.Intn(len(tableNumberPool.tableNumberPool))
	if tableNumberPool.curIndex < 0 {
		tableNumberPool.curIndex = 0
	}

	return nil
}

// 获取桌子编号
func GetIdleTableNumber() int32 {

	tableNumberPool.m.Lock()
	defer tableNumberPool.m.Unlock()

	for tableNumberPool.curIndex < len(tableNumberPool.tableNumberPool) {
		temp := tableNumberPool.tableNumberPool[tableNumberPool.curIndex]
		tableNumberPool.curIndex += 1
		return temp
	}

	temp := tableNumberPool.tableNumberPool[0]
	tableNumberPool.curIndex = 1
	return temp
}

func GetTableNumberCount() int32 {
	return int32(len(tableNumberPool.tableNumberPool))
}
