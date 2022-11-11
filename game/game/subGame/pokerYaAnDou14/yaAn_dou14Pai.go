package pokerYaAnD14

import (
	"encoding/json"
	"github.com/golang/glog"
	"io/ioutil"
	"math/rand"
	"strings"
)

const InvalidPai int8 = 0

const MinDianShu int8 = 0x01
const MaxDianShu int8 = 0x0D

const HeiTao int8 = 0x30
const HongTao int8 = 0x20
const MeiHua int8 = 0x10
const FangKuai int8 = 0x00

const XiaoWang int8 = 0x4E
const DaWang int8 = 0x4F
const LaiZi int8 = 0x70

type paiData struct {
	value  int8 // 牌值
	isUsed bool // 是否已经用了
}

type Dou14PaiBaseMgr struct {
	allPaiArr      []paiData // 牌
	allPaiArrIndex int       // 下标
	paiCount_      int

	//usePaiMap map[int8]struct{}

	testPaiArr []int8 // 测试牌
	onTest     bool   // 开启测试
	testIndex  int    // 测试牌 下标

	reserveFaPaiMap map[int8]struct{} // 测试牌
	reserveMoPaiMap map[int8]struct{} // 测试牌
}

func NewDou14PaiBaseMgr(test bool) *Dou14PaiBaseMgr {
	return &Dou14PaiBaseMgr{
		//usePaiMap:       make(map[int8]struct{}),
		reserveFaPaiMap: make(map[int8]struct{}),
		reserveMoPaiMap: make(map[int8]struct{}),
		onTest:          test}
}

// 洗牌
func (this *Dou14PaiBaseMgr) XiPai(wangHua int32) {

	if len(this.allPaiArr) == 0 {
		this.allPaiArr = make([]paiData, 0, 55)

		for huaSe := FangKuai; huaSe <= HeiTao; huaSe += 0x10 {
			for j := MinDianShu; j <= MaxDianShu; j++ {
				this.allPaiArr = append(this.allPaiArr, paiData{value: huaSe | j})
			}
		}

		for i := wangHua / 3; i > 0; i-- {
			this.allPaiArr = append(this.allPaiArr, paiData{value: DaWang})
			this.allPaiArr = append(this.allPaiArr, paiData{value: XiaoWang})
			this.allPaiArr = append(this.allPaiArr, paiData{value: LaiZi})
		}
	}

	//this.usePaiMap = make(map[int8]struct{})

	for k, _ := range this.allPaiArr {
		this.allPaiArr[k].isUsed = false
	}

	//rand.Shuffle(len(this.allPaiArr)/2, func(i, j int) {
	//	this.allPaiArr[i], this.allPaiArr[j] = this.allPaiArr[j], this.allPaiArr[i]
	//})
	rand.Shuffle(len(this.allPaiArr), func(i, j int) {
		this.allPaiArr[i], this.allPaiArr[j] = this.allPaiArr[j], this.allPaiArr[i]
	})

	// 读取测试牌
	if this.onTest == true {
		this.ReadTestPai()
	}

	this.paiCount_ = len(this.allPaiArr)
	this.allPaiArrIndex = 0
	this.testIndex = 0
}
func (this *Dou14PaiBaseMgr) getPai(v int8) int {
	for i := this.allPaiArrIndex; i < len(this.allPaiArr); i++ {
		if this.allPaiArr[i].value == v && this.allPaiArr[i].isUsed == false {
			if _, ok := this.reserveFaPaiMap[this.allPaiArr[i].value]; ok == true {
				continue
			}
			this.allPaiArr[i].isUsed = true
			this.paiCount_ -= 1
			this.allPaiArrIndex = i + 1
			return i
		}
	}

	for i := 0; i < this.allPaiArrIndex && i < len(this.allPaiArr); i++ {
		if this.allPaiArr[i].value == v && this.allPaiArr[i].isUsed == false {
			if _, ok := this.reserveFaPaiMap[this.allPaiArr[i].value]; ok == true {
				continue
			}
			this.allPaiArr[i].isUsed = true
			this.paiCount_ -= 1
			this.allPaiArrIndex = i + 1
			return i
		}
	}

	for i := 0; i < len(this.allPaiArr); i++ {
		if this.allPaiArr[i].isUsed == false {
			this.allPaiArr[i].isUsed = true
			this.paiCount_ -= 1
			this.allPaiArrIndex = i + 1
			return i
		}
	}
	return -1
}

// 发牌
func (this *Dou14PaiBaseMgr) FaPai(takeout int32, reserveShouPai []int8) []int8 {

	paiArr := make([]int8, 0, takeout)

	getPaiSuccessFunc := func(pai int8) {
		paiArr = append(paiArr, pai)
		takeout -= 1
		//this.usePaiMap[pai] = struct{}{}
	}

	// 是否开启了测试
	if this.onTest == true {
		for i := this.testIndex; i < len(this.testPaiArr) && takeout > 0; i++ {
			getPaiSuccessFunc(this.testPaiArr[i])
			this.testIndex += 1
		}

		return paiArr
	}

	// 是否有预定
	for _, v := range reserveShouPai {
		delete(this.reserveFaPaiMap, v)
		i_ := this.getPai(v)
		if i_ >= 0 {
			getPaiSuccessFunc(v)
		}
		if takeout < 1 {
			break
		}
	}

	for j := 0; j < 100 && takeout > 0; j++ {
		i_ := this.getPai(InvalidPai)
		if i_ >= 0 {
			getPaiSuccessFunc(this.allPaiArr[i_].value)
		}
	}

	return paiArr
}

func (this *Dou14PaiBaseMgr) FaPaiOver() {
	this.reserveFaPaiMap = map[int8]struct{}{}
}

// 摸牌
func (this *Dou14PaiBaseMgr) MoPai(reserveMoPai int8) int8 {

	// 是否开启了测试
	if this.onTest == true {
		for i := this.testIndex; i < len(this.testPaiArr); i++ {
			this.testIndex += 1
			return this.testPaiArr[i]
		}

		return InvalidPai
	}

	// 是否有预定
	if reserveMoPai != InvalidPai {
		delete(this.reserveMoPaiMap, reserveMoPai)
		i_ := this.getPai(reserveMoPai)
		if i_ >= 0 {
			return this.allPaiArr[i_].value
		}
	}

	i_ := this.getPai(InvalidPai)
	if i_ >= 0 {
		return this.allPaiArr[i_].value
	}

	return InvalidPai
}

// 剩余牌的数量
func (this *Dou14PaiBaseMgr) GetTheRestOfPaiCount() int32 {
	if this.onTest == true {
		return int32(len(this.testPaiArr) - this.testIndex)
	}

	return int32(this.paiCount_)
}

// 获取剩下所有的牌
func (this *Dou14PaiBaseMgr) GetSurplusPai() []int8 {
	arr := make([]int8, 0, 30)
	for i := 0; i < len(this.allPaiArr); i++ {
		if this.allPaiArr[i].isUsed == false {
			arr = append(arr, this.allPaiArr[i].value)
		}
	}

	return arr
}

// 预定
func (this *Dou14PaiBaseMgr) ReserveShouPai(paiArr []int8) (int8, int32) {

	for _, v := range paiArr {
		if v == DaWang || v == XiaoWang || v == LaiZi {
			continue
		}

		pv := v & 0x0F
		switch v & 0x70 {
		case FangKuai, MeiHua, HongTao, HeiTao:
			if pv >= MinDianShu && pv <= MaxDianShu {
				continue
			}
		default:
		}
		return v, -2
	}
	for _, v := range paiArr {
		this.reserveFaPaiMap[v] = struct{}{}
	}

	return 0, 0
}

func (this *Dou14PaiBaseMgr) ReserveMoPai(pai int8) int32 {

	if pai == DaWang || pai == XiaoWang || pai == LaiZi {
		for i := 0; i < len(this.allPaiArr); i++ {
			if this.allPaiArr[i].value == pai {
				if this.allPaiArr[i].isUsed == false {
					this.reserveMoPaiMap[pai] = struct{}{}
					return 0
				}
			}
		}
	} else {
		for i := 0; i < len(this.allPaiArr); i++ {
			if this.allPaiArr[i].value == pai {
				if this.allPaiArr[i].isUsed == false {
					this.reserveMoPaiMap[pai] = struct{}{}
					return 0
				}
				return -2
			}
		}
	}

	return -1
}

func (this *Dou14PaiBaseMgr) ReadTestPai() {
	this.testPaiArr = make([]int8, 0, 54)

	//tempPaiArr[0x0E] = map[int8]int8{0x40: 0}
	//tempPaiArr[0x0F] = map[int8]int8{0x40: 0}
	//tempPaiArr[0] = map[int8]int8{0x70: 0}

	data, err := ioutil.ReadFile("./testPokerPai.json")
	if err != nil {
		glog.Fatal("read testPokerPai.json error....", err.Error())
	}
	var testPaiArr []string
	err = json.Unmarshal(data, &testPaiArr)
	if err != nil {
		errBak := err

		var testPaiArrInt []int8
		err = json.Unmarshal(data, &testPaiArrInt)
		if err != nil {
			err = errBak
			glog.Fatal("read json.Unmarshal(data, pai) error....", err.Error())
		}

		//tempPaiArr := [16]map[int8]int8{}
		//for i := MinDianShu; i <= MaxDianShu; i++ {
		//	tempPaiArr[i] = make(map[int8]int8, 0)
		//	tempPaiArr[i][HeiTao] = HeiTao
		//	tempPaiArr[i][HongTao] = HongTao
		//	tempPaiArr[i][MeiHua] = MeiHua
		//	tempPaiArr[i][FangKuai] = FangKuai
		//}

		this.testPaiArr = testPaiArrInt
	} else {

		huaSeMap := [16]map[int8]int8{}
		for i := MinDianShu; i <= MaxDianShu; i++ {
			huaSeMap[i] = make(map[int8]int8, 0)
			huaSeMap[i][HeiTao] = HeiTao
			huaSeMap[i][HongTao] = HongTao
			huaSeMap[i][MeiHua] = MeiHua
			huaSeMap[i][FangKuai] = FangKuai
		}

		if len(testPaiArr[0]) < 3 {

			for i, v := range testPaiArr {
				switch v {
				case "w":
					this.testPaiArr = append(this.testPaiArr, 0x4E)
					continue
				case "W":
					this.testPaiArr = append(this.testPaiArr, 0x4F)
					continue
				case "l", "L":
					this.testPaiArr = append(this.testPaiArr, 0x70)
					continue
				default:
				}

				dianShu, ok := _vMap[strings.ToUpper(v)]
				if ok == false {
					glog.Fatal("PaiStringToValue() error....", v)
				}

				huaSe := int8(-1)
				for k, _ := range huaSeMap[dianShu] {
					huaSe = k
					delete(huaSeMap[dianShu], huaSe)
					break
				}
				if huaSe == -1 {
					glog.Fatal("PaiStringToValue() error....", v, ",pos:=", i)
				}
				this.testPaiArr = append(this.testPaiArr, huaSe|dianShu)
			}

		} else {
			for _, v := range testPaiArr {
				value := PaiStringToValue(strings.ToUpper(v))
				if value == InvalidPai {
					glog.Fatal("PaiStringToValue() error....", v)
				}
				huaSe := int8(-1)
				huaSe = 0
				//for k, _ := range huaSeMap[value] {
				//	huaSe = k
				//	delete(huaSeMap[value], huaSe)
				//	break
				//}
				//if huaSe == -1 {
				//	glog.Fatal("PaiStringToValue() error....", v, ",pos:=", i)
				//}
				this.testPaiArr = append(this.testPaiArr, huaSe|value)
			}
		}
	}
}

var _vMap = map[string]int8{
	"A": 0x01, "2": 0x02, "3": 0x03, "4": 0x04, "5": 0x05, "6": 0x06, "7": 0x07, "8": 0x08, "9": 0x09, "10": 0x0A, "J": 0x0B, "Q": 0x0C, "K": 0x0D,
}

func PaiStringToValue(pai string) int8 {
	//pokerPaiStringToValueMap := map[string]int8{
	//	"FK3": 0x03, "FK4": 0x04, "FK5": 0x05, "FK6": 0x06, "FK7": 0x07, "FK8": 0x08, "FK9": 0x09, "FK10": 0x0A, "FKJ": 0x0B, "FKQ": 0x0C, "FKK": 0x0D, "FKA": 0x0E, "FK2": 0x0F,
	//	"MH3": 0x03, "MH4": 0x04, "MH5": 0x05, "MH6": 0x06, "MH7": 0x07, "MH8": 0x08, "MH9": 0x09, "MH10": 0x0A, "MHJ": 0x0B, "MHQ": 0x0C, "MHK": 0x0D, "MHA": 0x0E, "MH2": 0x0F,
	//	"HT3": 0x03, "HT4": 0x04, "HT5": 0x05, "HT6": 0x06, "HT7": 0x07, "HT8": 0x08, "HT9": 0x09, "HT10": 0x0A, "HTJ": 0x0B, "HTQ": 0x0C, "HTK": 0x0D, "HTA": 0x0E, "HT2": 0x0F,
	//	"BT3": 0x03, "BT4": 0x04, "BT5": 0x05, "BT6": 0x06, "BT7": 0x07, "BT8": 0x08, "BT9": 0x09, "BT10": 0x0A, "BTJ": 0x0B, "BTQ": 0x0C, "BTK": 0x0D, "BTA": 0x0E, "BT2": 0x0F,
	//	"DW": 0x44, "XW": 0x43,
	//}

	pokerPaiStringToValueMap := map[string]int8{
		"FKA": 0x01, "FK2": 0x02, "FK3": 0x03, "FK4": 0x04, "FK5": 0x05, "FK6": 0x06, "FK7": 0x07, "FK8": 0x08, "FK9": 0x09, "FK10": 0x0A, "FKJ": 0x0B, "FKQ": 0x0C, "FKK": 0x0D,
		"MHA": 0x11, "MH2": 0x12, "MH3": 0x13, "MH4": 0x14, "MH5": 0x15, "MH6": 0x16, "MH7": 0x17, "MH8": 0x18, "MH9": 0x19, "MH10": 0x1A, "MHJ": 0x1B, "MHQ": 0x1C, "MHK": 0x1D,
		"HTA": 0x21, "HT2": 0x22, "HT3": 0x23, "HT4": 0x24, "HT5": 0x25, "HT6": 0x26, "HT7": 0x27, "HT8": 0x28, "HT9": 0x29, "HT10": 0x2A, "HTJ": 0x2B, "HTQ": 0x2C, "HTK": 0x2D,
		"BTA": 0x31, "BT2": 0x32, "BT3": 0x33, "BT4": 0x34, "BT5": 0x35, "BT6": 0x36, "BT7": 0x37, "BT8": 0x38, "BT9": 0x39, "BT10": 0x3A, "BTJ": 0x3B, "BTQ": 0x3C, "BTK": 0x3D,
		"XW": 0x4E, "DW": 0x4F, "LZ": 0x70,
	}

	if v, ok := pokerPaiStringToValueMap[pai]; ok == true {
		return v
	}
	return InvalidPai
}

func PaiValueToString(pai int8) string {
	MjPaiValueToStringMap := map[int8]string{
		0x03: "方块3", 0x04: "方块4", 0x05: "方块5", 0x06: "方块6", 0x07: "方块7", 0x08: "方块8", 0x09: "方块9", 0x0A: "方块10", 0x0B: "方块J", 0x0C: "方块Q", 0x0D: "方块K", 0xE: "方块A",
		0x13: "梅花3", 0x14: "梅花4", 0x15: "梅花5", 0x16: "梅花6", 0x17: "梅花7", 0x18: "梅花8", 0x19: "梅花9", 0x1A: "梅花10", 0x1B: "梅花J", 0x1C: "梅花Q", 0x1D: "梅花K", 0x1E: "梅花A",
		0x23: "红桃3", 0x24: "红桃4", 0x25: "红桃5", 0x26: "红桃6", 0x27: "红桃7", 0x28: "红桃8", 0x29: "红桃9", 0x2A: "红桃10", 0x2B: "红桃J", 0x2C: "红桃Q", 0x2D: "红桃K", 0x2E: "红桃A",
		0x33: "黑桃3", 0x34: "黑桃4", 0x35: "黑桃5", 0x36: "黑桃6", 0x37: "黑桃7", 0x38: "黑桃8", 0x39: "黑桃9", 0x3A: "黑桃10", 0x3B: "黑桃J", 0x3C: "黑桃Q", 0x3D: "黑桃K", 0x3E: "黑桃A", 0x3F: "黑桃2",
	}

	if v, ok := MjPaiValueToStringMap[pai]; ok == true {
		return v
	}
	return "?"
}

//func PaiValueToString(pai int8) string {
//	MjPaiValueToStringMap := map[int8]string{
//		0x03: "FK3", 0x04: "FK4", 0x05: "FK5", 0x06: "FK6", 0x07: "FK7", 0x08: "FK8", 0x09: "FK9", 0x0A: "FK10", 0x0B: "FKJ", 0x0C: "FKQ", 0x0D: "FKK", 0x0E: "FKA", 0x0F: "FK2",
//		0x13: "MH3", 0x14: "MH4", 0x15: "MH5", 0x16: "MH6", 0x17: "MH7", 0x18: "MH8", 0x19: "MH9", 0x1A: "MH10", 0x1B: "MHJ", 0x1C: "MHQ", 0x1D: "MHK", 0x1E: "MHA", 0x1F: "MH2",
//		0x23: "HT3", 0x24: "HT4", 0x25: "HT5", 0x26: "HT6", 0x27: "HT7", 0x28: "HT8", 0x29: "HT9", 0x2A: "HT10", 0x2B: "HTJ", 0x2C: "HTQ", 0x2D: "HTK", 0x2E: "HTA", 0x2F: "HT2",
//		0x33: "BT3", 0x34: "BT4", 0x35: "BT5", 0x36: "BT6", 0x37: "BT7", 0x38: "BT8", 0x39: "BT9", 0x3A: "BT10", 0x3B: "BTJ", 0x3C: "BTQ", 0x3D: "BTK", 0x3E: "BTA", 0x3F: "BT2",
//		0x44: "DW", 0x43: "XW",
//	}
//
//	if v, ok := MjPaiValueToStringMap[pai]; ok == true {
//		return v
//	}
//	return "?"
//}

// 斗地主\跑得快
//   3	 4	    5	   6	 7	   8	 9	  10	 J	   Q	 K	   A    2
//0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F	//方块 0x00
//0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F	//梅花 0x10
//0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F	//红桃 0x20
//0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D, 0x3E, 0x3F	//黑桃 0x30
//0x43, 0x44                                                                    //小王、大王
