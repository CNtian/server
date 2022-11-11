package gameMaJiang

import (
	"encoding/json"
	"github.com/golang/glog"
	"io/ioutil"
	"math/rand"
	"strings"
)

const InvalidPai int8 = 0

const MinDianShu_1 int8 = 0x01
const MaxDianShu_9 int8 = 0x09

const Tong int8 = 0x00
const Suo int8 = 0x10
const Wan int8 = 0x20
const Zi int8 = 0x30

const MinHuaSe int8 = 0
const MaxHuaSe int8 = 3

const MinZiPai int8 = 0x01
const MaxZiPai int8 = 0x07

const Zhong int8 = 0x35
const Fa int8 = 0x36
const Bai int8 = 0x37

//       1     2      3     4    5     6     7     8     9
//0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09	//筒 0x00
//0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19	//条 0x10
//0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29	//万 0x20
//0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39	//东南西北中发白 0x30
//        东    南    西    北     中   发     白

func PaiStringToValue(pai string) int8 {
	MjPaiStringToValueMap := map[string]int8{
		"1T": 0x01, "2T": 0x02, "3T": 0x03, "4T": 0x04, "5T": 0x05, "6T": 0x06, "7T": 0x07, "8T": 0x08, "9T": 0x09,
		"1S": 0x11, "2S": 0x12, "3S": 0x13, "4S": 0x14, "5S": 0x15, "6S": 0x16, "7S": 0x17, "8S": 0x18, "9S": 0x19,
		"1W": 0x21, "2W": 0x22, "3W": 0x23, "4W": 0x24, "5W": 0x25, "6W": 0x26, "7W": 0x27, "8W": 0x28, "9W": 0x29,
		"DF": 0x31, "NF": 0x32, "XF": 0x33, "BF": 0x34, "HZ": 0x35, "FC": 0x36, "BB": 0x37,
	}

	if v, ok := MjPaiStringToValueMap[pai]; ok == true {
		return v
	}
	return InvalidPai
}

func PaiValueToString(pai uint8) string {
	MjPaiValueToStringMap := map[uint8]string{
		0x01: "1T", 0x02: "2T", 0x03: "3T", 0x04: "4T", 0x05: "5T", 0x06: "6T", 0x07: "7T", 0x08: "8T", 0x09: "9T",
		0x11: "1S", 0x12: "2S", 0x13: "3S", 0x14: "4S", 0x15: "5S", 0x16: "6S", 0x17: "7S", 0x18: "8S", 0x19: "9S",
		0x21: "1W", 0x22: "2W", 0x23: "3W", 0x24: "4W", 0x25: "5W", 0x26: "6W", 0x27: "7W", 0x28: "8W", 0x29: "9W",
		0x31: "DF", 0x32: "NF", 0x33: "XF", 0x34: "BF", 0x35: "HZ", 0x36: "FC", 0x37: "BB",
	}

	if v, ok := MjPaiValueToStringMap[pai]; ok == true {
		return v
	}
	return "?"
}

type MJPaiMgr interface {
	XiPai()
	GetGroupPai(takeout int32) []int8
	GetPai() int8
	GetTheRestOfPaiCount() int32
}

type MJPaiBaseMgr struct {
	isIncludeZi   bool
	isIncludeTong bool
	isIncludeSuo  bool
	isIncludeWan  bool
	paiArr        []int8
	faPaiIndex    int32

	test bool
}

func NewMJPaiBaseMgr(zi, tong, suo, wan, test bool) MJPaiBaseMgr {
	return MJPaiBaseMgr{
		isIncludeZi:   zi,
		isIncludeTong: tong,
		isIncludeSuo:  suo,
		isIncludeWan:  wan,
		test:          test}
}

// 洗牌
func (this *MJPaiBaseMgr) XiPai() {

	if this.paiArr == nil {
		this.paiArr = make([]int8, 0)

		if this.isIncludeZi == true {
			for j := MinZiPai; j <= MaxZiPai; j++ {
				for k := 0; k < 4; k++ {
					this.paiArr = append(this.paiArr, 0x30+j)
				}
			}
		}

		if this.isIncludeTong == true {
			paiType := Tong
			for j := MinDianShu_1; j <= MaxDianShu_9; j++ {
				for k := 0; k < 4; k++ {
					this.paiArr = append(this.paiArr, paiType+j)
				}
			}
		}
		if this.isIncludeSuo == true {
			paiType := Suo
			for j := MinDianShu_1; j <= MaxDianShu_9; j++ {
				for k := 0; k < 4; k++ {
					this.paiArr = append(this.paiArr, paiType+j)
				}
			}
		}
		if this.isIncludeWan == true {
			paiType := Wan
			for j := MinDianShu_1; j <= MaxDianShu_9; j++ {
				for k := 0; k < 4; k++ {
					this.paiArr = append(this.paiArr, paiType+j)
				}
			}
		}

		rand.Shuffle(len(this.paiArr), func(i, j int) {
			this.paiArr[i], this.paiArr[j] = this.paiArr[j], this.paiArr[i]
		})
	}

	rand.Shuffle(len(this.paiArr), func(i, j int) {
		this.paiArr[i], this.paiArr[j] = this.paiArr[j], this.paiArr[i]
	})

	// 读取测试牌
	if this.test == true {
		data, err := ioutil.ReadFile("./testMaJiangPai.json")
		if err != nil {
			glog.Fatal("read testMaJiangPai.json error....", err.Error())
		}
		var testPaiArr []string
		err = json.Unmarshal(data, &testPaiArr)
		if err != nil {
			glog.Fatal("read json.Unmarshal(data, pai) error....", err.Error())
		}

		this.paiArr = make([]int8, 0)

		for _, v := range testPaiArr {
			value := PaiStringToValue(strings.ToUpper(v))
			if value == InvalidPai {
				glog.Fatal("PaiStringToValue() error....", v)
			}
			this.paiArr = append(this.paiArr, value)
		}
	}

	this.faPaiIndex = 0
}

// 发牌
func (this *MJPaiBaseMgr) GetGroupPai(takeout int32) []int8 {
	if int(this.faPaiIndex+takeout) >= len(this.paiArr) {
		takeout = int32(len(this.paiArr)-int(this.faPaiIndex)) - 1
	}
	if takeout < 1 {
		return nil
	}

	paiArr := make([]int8, takeout)
	for takeout = takeout - 1; takeout >= 0; takeout-- {
		paiArr[takeout] = this.paiArr[this.faPaiIndex]
		this.faPaiIndex += 1
	}
	return paiArr
}

// 发牌
func (this *MJPaiBaseMgr) GetPai() int8 {
	if int(this.faPaiIndex) < len(this.paiArr) {
		pai := this.paiArr[this.faPaiIndex]
		this.faPaiIndex += 1
		return pai
	}
	return InvalidPai
}

// 剩余牌的数量
func (this *MJPaiBaseMgr) GetTheRestOfPaiCount() int32 {
	return int32(len(this.paiArr)) - this.faPaiIndex + 1
}

/*
let arr = [1,2,3,4,5]

for (let i = arr.length - 1; i >= 0; i--) {
	let randomIndex = Math.floor(Math.random() * i) // 确定候选区域的随机索引
	let temp = arr[i] // 进行交换
	arr[i] = arr[randomIndex]
	arr[randomIndex] = temp
}
*/
