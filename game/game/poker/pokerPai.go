package pokerTable

import (
	"encoding/json"
	"github.com/golang/glog"
	"io/ioutil"
	"math/rand"
	"strings"
)

const InvalidPai int8 = 0

const MinDianShu int8 = 0x03
const MaxDianShu int8 = 0x0F

const MinHuaSe int8 = 0x00
const MaxHuaSe int8 = 0x40

const HeiTao int8 = 0x30
const HongTao int8 = 0x20
const MeiHua int8 = 0x10
const FangKuai int8 = 0x00

const ADianShu int8 = 0x0E
const XiaoWang int8 = 0x43
const DaWang int8 = 0x44

type PokerPaiBaseMgr struct {
	allPaiArr      []int8 // 牌
	allPaiArrIndex int    // 下标

	notUsePaiMap map[int8]struct{}
	usePaiMap    map[int8]struct{}

	faPaiCount    int    // 发了多少牌
	players       int32  // 玩家人数
	groupPaiCount int32  // 一手牌的数量
	surplusPaiArr []int8 // 剩余牌

	testPaiArr []int8 // 测试牌
	onTest     bool   // 开启测试
	testIndex  int    // 测试牌 下标

	seatReserve   [4][]int8         // 测试牌
	reservePaiMap map[int8]struct{} // 测试牌
}

func NewPokerPaiBaseMgr(test bool) *PokerPaiBaseMgr {
	return &PokerPaiBaseMgr{
		notUsePaiMap:  make(map[int8]struct{}),
		usePaiMap:     make(map[int8]struct{}),
		reservePaiMap: make(map[int8]struct{}),
		onTest:        test}
}

// 洗牌
func (this *PokerPaiBaseMgr) XiPai(players, groupPaiCount int32) {
	this.players, this.groupPaiCount = players, groupPaiCount

	if len(this.notUsePaiMap) == 0 && len(this.usePaiMap) == 0 {
		for huaSe := FangKuai; huaSe <= HeiTao; huaSe += 0x10 {
			for j := MinDianShu; j <= MaxDianShu-3; j++ {
				this.notUsePaiMap[huaSe|j] = struct{}{}
			}
		}

		// K
		this.notUsePaiMap[HeiTao|(MinDianShu+10)] = struct{}{}
		this.notUsePaiMap[HongTao|(MinDianShu+10)] = struct{}{}
		this.notUsePaiMap[MeiHua|(MinDianShu+10)] = struct{}{}
		if this.groupPaiCount == 16 {
			this.notUsePaiMap[FangKuai|(MinDianShu+10)] = struct{}{}
		}
		// A
		this.notUsePaiMap[HeiTao|(MinDianShu+11)] = struct{}{}
		if this.groupPaiCount == 16 {
			this.notUsePaiMap[HongTao|(MinDianShu+11)] = struct{}{}
			this.notUsePaiMap[MeiHua|(MinDianShu+11)] = struct{}{}
		}
		// 2
		this.notUsePaiMap[HeiTao|(MinDianShu+12)] = struct{}{}

		this.allPaiArr = make([]int8, len(this.notUsePaiMap))
	}

	for k, _ := range this.usePaiMap {
		delete(this.usePaiMap, k)
		this.notUsePaiMap[k] = struct{}{}
	}

	index := 0
	for k, _ := range this.notUsePaiMap {
		this.allPaiArr[index] = k
		index++
	}
	rand.Shuffle(len(this.allPaiArr)/2, func(i, j int) {
		this.allPaiArr[i], this.allPaiArr[j] = this.allPaiArr[j], this.allPaiArr[i]
	})
	rand.Shuffle(len(this.allPaiArr), func(i, j int) {
		this.allPaiArr[i], this.allPaiArr[j] = this.allPaiArr[j], this.allPaiArr[i]
	})

	// 读取测试牌
	if this.onTest == true {
		this.ReadTestPai()
	}

	this.allPaiArrIndex = 0
	this.faPaiCount = 0
	this.testIndex = 0
}

// 发牌
func (this *PokerPaiBaseMgr) GetGroupPai(seatNum, takeout int32, cb func(int8)) []int8 {

	paiArr := make([]int8, 0, takeout)

	getPaiSuccessFunc := func(pai int8) {
		paiArr = append(paiArr, pai)
		takeout -= 1
		this.faPaiCount += 1
		this.usePaiMap[pai] = struct{}{}
		delete(this.notUsePaiMap, pai)
		if cb != nil {
			cb(pai)
		}
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
	for _, v := range this.seatReserve[seatNum] {
		if _, ok := this.notUsePaiMap[v]; ok == true {
			getPaiSuccessFunc(v)
		}
	}

	var pai int8
	for len(this.notUsePaiMap) > 0 && takeout > 0 {
		for this.allPaiArrIndex < len(this.allPaiArr) {
			pai = this.allPaiArr[this.allPaiArrIndex]
			this.allPaiArrIndex++
			if _, ok := this.reservePaiMap[pai]; ok == true {
				continue
			}
			if _, ok := this.notUsePaiMap[pai]; ok {
				getPaiSuccessFunc(pai)
				if takeout < 1 {
					break
				}
			}
		}
	}

	if this.faPaiCount >= int(this.groupPaiCount*this.players) {
		this.surplusPaiArr = make([]int8, 0, 46)

		for k, _ := range this.notUsePaiMap {
			this.surplusPaiArr = append(this.surplusPaiArr, k)
		}
		this.reservePaiMap = map[int8]struct{}{}
	}

	this.seatReserve[seatNum] = nil

	return paiArr
}

// 剩余牌的数量
func (this *PokerPaiBaseMgr) GetTheRestOfPaiCount() int32 {
	return int32(len(this.notUsePaiMap))
}

// 获取剩下所有的牌
func (this *PokerPaiBaseMgr) GetSurplusPai() []int8 {
	return this.surplusPaiArr
}

// 随机一张牌
func (this *PokerPaiBaseMgr) RandomPai() int8 {
	if this.onTest == true {
		index := rand.Intn(len(this.testPaiArr))
		if index < 0 {
			index = 0
		}
		return this.testPaiArr[index]
	}

	for k, _ := range this.notUsePaiMap {
		return k
	}
	return InvalidPai
}

// 预定
func (this *PokerPaiBaseMgr) Reserve(seatNum int32, paiArr []int8) (int8, int32) {

	for _, v := range paiArr {
		if _, ok := this.notUsePaiMap[v]; ok == true {
			continue
		}

		if _, ok := this.usePaiMap[v]; ok == true {
			continue
		}
		return v, -2
	}
	for _, v := range paiArr {
		this.reservePaiMap[v] = struct{}{}
	}
	this.seatReserve[seatNum] = paiArr
	return 0, 0
}

func (this *PokerPaiBaseMgr) ReadTestPai() {
	this.testPaiArr = make([]int8, 0, 54)

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
		for _, v := range testPaiArrInt {
			this.testPaiArr = append(this.testPaiArr, v)
		}
	} else {
		tempPaiArr := [16]map[int8]int8{}
		for i := MinDianShu; i <= MaxDianShu; i++ {
			tempPaiArr[i] = make(map[int8]int8, 0)
			tempPaiArr[i][HeiTao] = HeiTao
			tempPaiArr[i][HongTao] = HongTao
			tempPaiArr[i][MeiHua] = MeiHua
			tempPaiArr[i][FangKuai] = FangKuai
		}
		for i, v := range testPaiArr {
			value := PaiStringToValue(strings.ToUpper(v))
			if value == InvalidPai {
				glog.Fatal("PaiStringToValue() error....", v)
			}
			huaSe := int8(-1)
			for k, _ := range tempPaiArr[value] {
				huaSe = k
				delete(tempPaiArr[value], huaSe)
				break
			}
			if huaSe == -1 {
				glog.Fatal("PaiStringToValue() error....", v, ",pos:=", i)
			}
			this.testPaiArr = append(this.testPaiArr, huaSe|value)
		}
	}
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
		"3": 0x03, "4": 0x04, "5": 0x05, "6": 0x06, "7": 0x07, "8": 0x08, "9": 0x09, "10": 0x0A, "J": 0x0B, "Q": 0x0C, "K": 0x0D, "A": 0x0E, "2": 0x0F,
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
