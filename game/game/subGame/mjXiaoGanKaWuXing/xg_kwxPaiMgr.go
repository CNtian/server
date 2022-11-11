package mjXiaoGanKWXTable

import (
	"encoding/json"
	"github.com/golang/glog"
	"io/ioutil"
	"math/rand"
	"qpGame/game/gameMaJiang"
	"strings"
)

type kwxPaiMgr struct {
	isIncludeFeng  bool
	isIncludeBaJiu bool

	paiArr     []int8
	faPaiIndex int32

	test bool
}

func NewKWXPaiMgr(feng, baJiu, test bool) *kwxPaiMgr {
	return &kwxPaiMgr{
		isIncludeFeng:  feng,
		isIncludeBaJiu: baJiu,
		test:           test}
}

// 洗牌
func (this *kwxPaiMgr) XiPai() {

	if this.paiArr == nil {
		this.paiArr = make([]int8, 0, 84)

		if this.isIncludeFeng == true {
			for k := 0; k < 4; k++ {
				this.paiArr = append(this.paiArr, gameMaJiang.Zhong)
				this.paiArr = append(this.paiArr, gameMaJiang.Fa)
				this.paiArr = append(this.paiArr, gameMaJiang.Bai)
			}
		}

		paiType := gameMaJiang.Tong
		// 1 -7
		for j := gameMaJiang.MinDianShu_1; j < gameMaJiang.MaxDianShu_9-1; j++ {
			for k := 0; k < 4; k++ {
				this.paiArr = append(this.paiArr, paiType+j)
			}
		}
		if this.isIncludeBaJiu == true {
			dianShu8 := gameMaJiang.MaxDianShu_9 - 1
			for k := 0; k < 4; k++ {
				this.paiArr = append(this.paiArr, paiType+dianShu8)
				this.paiArr = append(this.paiArr, paiType+gameMaJiang.MaxDianShu_9)
			}
		}

		paiType = gameMaJiang.Suo
		// 1 -7
		for j := gameMaJiang.MinDianShu_1; j < gameMaJiang.MaxDianShu_9-1; j++ {
			for k := 0; k < 4; k++ {
				this.paiArr = append(this.paiArr, paiType+j)
			}
		}
		if this.isIncludeBaJiu == true {
			dianShu8 := gameMaJiang.MaxDianShu_9 - 1
			for k := 0; k < 4; k++ {
				this.paiArr = append(this.paiArr, paiType+dianShu8)
				this.paiArr = append(this.paiArr, paiType+gameMaJiang.MaxDianShu_9)
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
			value := gameMaJiang.PaiStringToValue(strings.ToUpper(v))
			if value == gameMaJiang.InvalidPai {
				glog.Fatal("PaiStringToValue() error....", v)
			}
			this.paiArr = append(this.paiArr, value)
		}
	}

	this.faPaiIndex = 0
}

// 发牌
func (this *kwxPaiMgr) GetGroupPai(takeout int32) []int8 {
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
func (this *kwxPaiMgr) GetPai() int8 {
	if int(this.faPaiIndex) < len(this.paiArr) {
		pai := this.paiArr[this.faPaiIndex]
		this.faPaiIndex += 1
		return pai
	}
	return gameMaJiang.InvalidPai
}

// 发牌
func (this *kwxPaiMgr) GetNextPai(pai int8) int8 {
	for i := int(this.faPaiIndex); i < len(this.paiArr); i++ {
		if pai == this.paiArr[i] {
			this.paiArr[this.faPaiIndex], this.paiArr[i] = this.paiArr[i], this.paiArr[this.faPaiIndex]
			this.faPaiIndex += 1
			return pai
		}
	}

	return this.GetPai()
}

// 剩余牌的数量
func (this *kwxPaiMgr) GetTheRestOfPaiCount() int32 {
	return int32(len(this.paiArr)) - this.faPaiIndex
}

func (this *kwxPaiMgr) GetRemainPai() map[int8]int8 {
	paiMap := make(map[int8]int8)
	for i := int(this.faPaiIndex); i < len(this.paiArr); i++ {
		if v, ok := paiMap[this.paiArr[i]]; ok == false {
			paiMap[this.paiArr[i]] = 1
		} else {
			paiMap[this.paiArr[i]] = v + 1
		}
	}
	return paiMap
}
