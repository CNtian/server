package NiuNiu_mpQiangZhuang

import (
	"sort"
)

// 顺子11    同花12   葫芦13   炸弹14  五花15  五小16  同花顺17
const (
	TongHuaShun = 17 // 同花顺
	WuXiaoNiu   = 16 // 五小牛
	WuHuaNiu    = 15 // 五花牛
	ZhaDan      = 14 // 炸弹
	HuLuNiu     = 13 // 葫芦牛
	TongHuaNiu  = 12 // 同花牛
	ShunZiNiu   = 11 // 顺子牛
	NiuNiu      = 10 // 牛牛
	Niu_9       = 9
	Niu_8       = 8
	Niu_7       = 7
	Niu_6       = 6
	Niu_5       = 5
	Niu_4       = 4
	Niu_3       = 3
	Niu_2       = 2
	Niu_1       = 1
	Niu_0       = 0
	NULL        = -1
)

type niuNiuLogic struct {
	//PaiXing int32
	//MaxPai  int8
	ArrangePai []int8
	lzPai      int8

	maxLaiZiPaiXing struct {
		_type      int32
		_paiArr    []int8
		_maxPai    int8
		_lzChanged []int8
	}
	lzChanged []int8
}

// ():牌型,最大牌
func (this *niuNiuLogic) GetPaiXing(paiArr []int8) (int32, []int8, int8) {
	this.ArrangePai = paiArr
	if ok, maxPai := this.isTongHuaShun(paiArr); ok == true {
		return TongHuaShun, this.ArrangePai, maxPai
	}
	if this.IsWuXiaoNiu(paiArr) == true {
		return WuXiaoNiu, this.ArrangePai, getMaxPai(paiArr)
	}
	if ok, maxPai := this.IsZhaDan(paiArr); ok == true {
		return ZhaDan, this.ArrangePai, maxPai
	}
	if this.IsWuHuaNiu(paiArr) == true {
		return WuHuaNiu, this.ArrangePai, getMaxPai(paiArr)
	}
	if ok, maxPai := this.IsTongHuaNiu(paiArr); ok == true {
		return TongHuaNiu, this.ArrangePai, maxPai
	}
	if ok, maxPai := this.isShunZiNiu(paiArr); ok == true {
		return ShunZiNiu, this.ArrangePai, maxPai
	}
	if ok, maxPai := this.isHuLuNiu(paiArr); ok == true {
		return HuLuNiu, this.ArrangePai, maxPai
	}
	//if this.IsNiuNiu(paiArr) == true {
	//	return NiuNiu, this.ArrangePai, getMaxPai(paiArr)
	//}
	switch this.IsNiuX(paiArr) {
	case 10:
		return NiuNiu, this.ArrangePai, getMaxPai(paiArr)
	case 9:
		return Niu_9, this.ArrangePai, getMaxPai(paiArr)
	case 8:
		return Niu_8, this.ArrangePai, getMaxPai(paiArr)
	case 7:
		return Niu_7, this.ArrangePai, getMaxPai(paiArr)
	case 6:
		return Niu_6, this.ArrangePai, getMaxPai(paiArr)
	case 5:
		return Niu_5, this.ArrangePai, getMaxPai(paiArr)
	case 4:
		return Niu_4, this.ArrangePai, getMaxPai(paiArr)
	case 3:
		return Niu_3, this.ArrangePai, getMaxPai(paiArr)
	case 2:
		return Niu_2, this.ArrangePai, getMaxPai(paiArr)
	case 1:
		return Niu_1, this.ArrangePai, getMaxPai(paiArr)
	case 0:
		return Niu_0, this.ArrangePai, getMaxPai(paiArr)
	}

	return 0, this.ArrangePai, 0
}

func (this *niuNiuLogic) GetLaiZiPaiXing(paiArr []int8) (int32, []int8, int8, []int8) {

	this.maxLaiZiPaiXing._maxPai = 0
	this.maxLaiZiPaiXing._paiArr = make([]int8, len(paiArr))
	this.maxLaiZiPaiXing._type = 0
	this.maxLaiZiPaiXing._lzChanged = nil
	this.lzChanged = []int8{}

	this.calculateLaiZiPaiXing(paiArr, 0)

	return this.maxLaiZiPaiXing._type, this.maxLaiZiPaiXing._paiArr, this.maxLaiZiPaiXing._maxPai, this.maxLaiZiPaiXing._lzChanged
}

func (this *niuNiuLogic) calculateLaiZiPaiXing(paiArr []int8, index int) {

	t_ := int32(0)
	arragePai := make([]int8, 0, len(paiArr))
	maxV := int8(0)

	c := this.lzPai & 0x0F
	for i := index; i < len(paiArr); i++ {
		isXPai := false
		if DaWang == paiArr[i] || paiArr[i] == XiaoWang {
			isXPai = true
		} else if (paiArr[i] & 0x0F) == c {
			isXPai = true
		}
		if isXPai == false {
			continue
		}

		bak := paiArr[i]
		huaSeArr := [4]int8{FangKuai, MeiHua, HongTao, HeiTao}
		for v := MinDianShu; v <= MaxDianShu; v++ {
			for huaSe := 0; huaSe < len(huaSeArr); huaSe++ {
				paiArr[i] = huaSeArr[huaSe] | v
				this.lzChanged = append(this.lzChanged, bak)
				this.calculateLaiZiPaiXing(paiArr, i+1)
			}
		}
		paiArr[i] = bak
	}

	isRecover := false
	t_, arragePai, maxV = this.GetPaiXing(paiArr)
	if t_ > this.maxLaiZiPaiXing._type {
		isRecover = true
	} else if t_ == this.maxLaiZiPaiXing._type && maxV > this.maxLaiZiPaiXing._maxPai {
		isRecover = true
	}
	if isRecover {
		this.maxLaiZiPaiXing._type, this.maxLaiZiPaiXing._maxPai = t_, maxV
		copy(this.maxLaiZiPaiXing._paiArr, arragePai)
		this.maxLaiZiPaiXing._lzChanged = this.lzChanged
	}
	this.lzChanged = []int8{}
}

// A > B:true
func (this *niuNiuLogic) Compare(APaiXing int32, AmaxPai int8, BPaiXing int32, BmaxPai int8) bool {
	if APaiXing < BPaiXing {
		return false
	}

	if APaiXing > BPaiXing {
		return true
	}

	AdianShu := AmaxPai & 0x0F
	AType := AmaxPai >> 4

	BdianShu := BmaxPai & 0x0F
	BType := BmaxPai >> 4

	if AdianShu < BdianShu {
		return false
	}

	if AdianShu > BdianShu {
		return true
	}

	if AType >= BType {
		return true
	}

	return false
}

func (this *niuNiuLogic) isTongHuaShun(paiArr []int8) (bool, int8) {
	sortPaiArr := make([]int, len(paiArr))
	sortAKQJ10PaiArr := make([]int, len(paiArr))

	huaSe := paiArr[0] >> 4
	for i, v := range paiArr {
		if huaSe != v>>4 {
			return false, 0
		}

		if v&0x0F == ADianShu {
			temp := int(v >> 4)

			sortPaiArr[i] = temp*0x10 + 1

			sortAKQJ10PaiArr[i] = int(v)
			continue
		}
		if v&0x0F == _2DianShu {
			temp := int(v >> 4)
			sortPaiArr[i] = temp*0x10 + 2

			sortAKQJ10PaiArr[i] = sortPaiArr[i]
			continue
		}

		sortPaiArr[i] = int(v)
		sortAKQJ10PaiArr[i] = int(v)
	}

	sort.Ints(sortPaiArr)
	sort.Ints(sortAKQJ10PaiArr)

	ok := true
	for i := 0; i < len(sortPaiArr)-1; i++ {
		if sortPaiArr[i]+1 != sortPaiArr[i+1] {
			ok = false
			break
		}
	}

	if ok == true {
		return true, int8(sortPaiArr[len(sortPaiArr)-1])
	}

	for i := 0; i < len(sortAKQJ10PaiArr)-1; i++ {
		if sortAKQJ10PaiArr[i]+1 != sortAKQJ10PaiArr[i+1] {
			ok = false
			return false, 0
		}
	}
	return true, int8(sortAKQJ10PaiArr[len(sortAKQJ10PaiArr)-1])
}

func (this *niuNiuLogic) isShunZiNiu(paiArr []int8) (bool, int8) {

	tempGetDianShuFunc := func(pai int8) int8 {
		if pai&0x0F == ADianShu {
			return 1
		}
		if pai&0x0F == _2DianShu {
			return 2
		}
		return pai & 0x0F
	}
	sortPaiArr := make([]int, len(paiArr))
	for i, v := range paiArr {
		sortPaiArr[i] = int(tempGetDianShuFunc(v))
	}

	sort.Ints(sortPaiArr)

	ok := true
	for i := 0; i < len(sortPaiArr)-1; i++ {
		if sortPaiArr[i]+1 != sortPaiArr[i+1] {
			ok = false
			break
			//return false, 0
		}
	}

	if ok == true {
		var maxPai int8
		for _, v := range paiArr {
			if sortPaiArr[4] == int(tempGetDianShuFunc(v)) {
				maxPai = v
				break
			}
		}

		return true, maxPai
	}

	sortAKQJ10PaiArr := make([]int, len(paiArr))

	maxPai := int8(0)
	for i, v := range paiArr {
		if v&0x0F == _2DianShu {
			return false, 0
		}
		if (v & 0x0F) == ADianShu {
			maxPai = v
		}
		sortAKQJ10PaiArr[i] = int(v & 0x0F)
	}

	sort.Ints(sortAKQJ10PaiArr)

	for i := 0; i < len(sortAKQJ10PaiArr)-1; i++ {
		if sortAKQJ10PaiArr[i]+1 != sortAKQJ10PaiArr[i+1] {
			return false, 0
		}
	}

	return true, maxPai
}

func (this *niuNiuLogic) isHuLuNiu(paiArr []int8) (bool, int8) {

	tempGetDianShuFunc := func(pai int8) int8 {
		if pai&0x0F == ADianShu {
			return 1
		}
		if pai&0x0F == _2DianShu {
			return 2
		}
		return pai & 0x0F
	}

	dianShuMap := make(map[int8][]int8)
	ds := int8(0)
	for _, v := range paiArr {
		ds = tempGetDianShuFunc(v)
		paiArr, ok := dianShuMap[ds]
		if ok == false {
			paiArr = []int8{v}
		} else {
			paiArr = append(paiArr, v)
		}
		dianShuMap[ds] = paiArr
	}

	if len(dianShuMap) != 2 {
		return false, 0
	}

	_3Pai := int8(0)
	_2Pai := int8(0)
	for _, v := range dianShuMap {
		if len(v) == 3 {
			_3Pai = v[0]
		} else if len(v) == 2 {
			_2Pai = v[0]
		} else {
			return false, 0
		}
	}

	if _3Pai != 0 && _2Pai != 0 {
		return true, _3Pai
	}

	return false, 0
}

func (this *niuNiuLogic) IsWuXiaoNiu(paiArr []int8) bool {
	dianShuCount, ds := int8(0), int8(0)
	for _, v := range paiArr {

		ds = getDianShu(v)
		if ds >= 5 {
			return false
		}
		dianShuCount += ds
	}
	if dianShuCount < 10 {
		return true
	}
	return false
}

func (this *niuNiuLogic) IsZhaDan(paiArr []int8) (bool, int8) {

	dianShuMap := make(map[int8]int)

	for _, v := range paiArr {
		dianShu := v & 0x0F
		dianShuMap[dianShu] += 1

		if dianShuMap[dianShu] == 4 {
			return true, v
		}
	}

	return false, InvalidPai
}

func (this *niuNiuLogic) IsWuHuaNiu(paiArr []int8) bool {
	for _, v := range paiArr {
		dianShu := v & 0x0F
		if dianShu < jDianShu || dianShu > kDianShu {
			return false
		}
	}
	return true
}

func (this *niuNiuLogic) IsTongHuaNiu(paiArr []int8) (bool, int8) {

	maxPai := int8(0)
	huaSe := paiArr[0] >> 4
	for _, v := range paiArr {
		if huaSe != v>>4 {
			return false, 0
		}

		if getDianShu(v) > maxPai&0x0F {
			maxPai = v
		}
	}

	return true, maxPai
}

//func (this *niuNiuLogic) IsNiuNiu(paiArr []int8) bool {
//	var dianshuCount int8
//	for _, v := range paiArr {
//		dianshuCount += getDianShu(v)
//	}
//	if dianshuCount%10 == 0 {
//		return true
//	}
//	return false
//}

func getDianShu(pai int8) int8 {
	dianShu := pai & 0x0F
	//_type := (pai >> 4) * 0x10

	if dianShu == ADianShu { // A
		return 1
	} else if dianShu == MaxDianShu { // 2
		return 2
	} else if dianShu >= _10DianShu && dianShu <= kDianShu { // 10,J,Q,K
		return 10
	}
	return dianShu
}

func getMaxPai(paiArr []int8) int8 {
	var maxPai int8
	for _, v := range paiArr {
		dianShu := v & 0x0F

		if dianShu == ADianShu { // A
			dianShu = 1
		} else if dianShu == MaxDianShu { // 2
			dianShu = 2
		}

		if dianShu > maxPai&0x0F {
			maxPai = (v>>4)*0x10 | dianShu
		} else if dianShu == maxPai&0x0F {
			if v>>4 > maxPai>>4 {
				maxPai = (v>>4)*0x10 | dianShu
			}
		}
	}

	if maxPai&0x0F == 1 {
		maxPai = (maxPai>>4)*0x10 | ADianShu
	} else if maxPai&0x0F == 2 {
		maxPai = (maxPai>>4)*0x10 | MaxDianShu
	}

	return maxPai
}

func (this *niuNiuLogic) IsNiuX(paiArr []int8) int8 {

	var indexArr [10][5]int8
	indexArr[0] = [5]int8{0, 1, 2, 3, 4}
	indexArr[1] = [5]int8{0, 1, 3, 2, 4}
	indexArr[2] = [5]int8{0, 1, 4, 3, 2}
	indexArr[3] = [5]int8{0, 2, 3, 4, 1}
	indexArr[4] = [5]int8{0, 2, 4, 3, 1}
	indexArr[5] = [5]int8{0, 3, 4, 1, 2}
	indexArr[6] = [5]int8{1, 2, 3, 4, 0}
	indexArr[7] = [5]int8{1, 2, 4, 3, 0}
	indexArr[8] = [5]int8{1, 3, 4, 2, 0}
	indexArr[9] = [5]int8{2, 3, 4, 0, 1}

	var paiXingDianShu, dianShu, maxDianShu int8

	for _, v := range indexArr {
		paiXingDianShu = getDianShu(paiArr[v[0]]) + getDianShu(paiArr[v[1]]) + getDianShu(paiArr[v[2]])
		dianShu = getDianShu(paiArr[v[3]]) + getDianShu(paiArr[v[4]])
		if paiXingDianShu%10 == 0 && dianShu%10 == 0 {
			this.ArrangePai = []int8{paiArr[v[0]], paiArr[v[1]], paiArr[v[2]], paiArr[v[3]], paiArr[v[4]]}
			return 10
		} else if paiXingDianShu%10 == 0 && dianShu%10 > maxDianShu {
			maxDianShu = dianShu % 10
			this.ArrangePai = []int8{paiArr[v[0]], paiArr[v[1]], paiArr[v[2]], paiArr[v[3]], paiArr[v[4]]}
		}
	}

	return maxDianShu
}
