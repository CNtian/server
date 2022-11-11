package pokerPDKTable

import (
	pokerTable "qpGame/game/poker"
	"sort"
)

const PDK_PX_ZhaDan int32 = 10
const PDK_PX_SiDaiEr int32 = 9
const PDK_PX_SiDaiSan int32 = 8
const PDK_PX_FeiJi int32 = 7

const PDK_PX_SanDai_Yi int32 = 6
const PDK_PX_SanDai_Er int32 = 5

const PDK_PX_LianDui int32 = 4
const PDK_PX_YiDui int32 = 3
const PDK_PX_ShunZi int32 = 2
const PDK_PX_DandZhang int32 = 1
const PDK_PX_Invalid = 0

type paoDeKuaiLogic struct {
	rule *PDKRule

	Pai                 []int8 // 打出的牌
	PaiXing             int32
	PaiXingStartDianShu int8 // 牌型起始牌的点数
	shouPaiCount        int  // 手牌数量

	SequenceCount int  // 连续的数量
	_4W2IsDui     bool // 4带2是否是 带1对
	_4W3Is3       bool // 4带3是否是 带3张
}

func (A *paoDeKuaiLogic) IsGreaterX(B *paoDeKuaiLogic) int32 {

	if A.PaiXing == PDK_PX_ZhaDan &&
		B.PaiXing != PDK_PX_ZhaDan {
		return 0
	}
	if A.PaiXing != B.PaiXing {
		return -1
	}

	if A.PaiXingStartDianShu <= B.PaiXingStartDianShu {
		return -1
	}

	// 牌的数量
	if len(A.Pai) > len(B.Pai) {
		return -1
	} else if len(A.Pai) < len(B.Pai) {

		switch A.PaiXing {
		case PDK_PX_ZhaDan, PDK_PX_SiDaiEr, PDK_PX_SiDaiSan:
			if A.PaiXingStartDianShu == pokerTable.ADianShu &&
				len(A.Pai) == len(B.Pai)-1 {
				return 0
			}
		default:
		}
		//if A.PaiXing == PDK_PX_ZhaDan &&
		//	len(A.Pai) == 3 &&
		//	A.PaiXingStartDianShu == pokerTable.ADianShu {
		//	return 0
		//}

		return 1
	}

	return 0
}

func (this *paoDeKuaiLogic) ParsePaiXing(shouPaiCount int32, playPai []int8, assignPaiXing int32) bool {
	this.Pai = playPai
	this.shouPaiCount = int(shouPaiCount)
	this.PaiXing = PDK_PX_Invalid

	funcIsZhaDan := func() bool {
		if this.Is_ZhaDan() == true {
			this.PaiXing = PDK_PX_ZhaDan
			return true
		}
		return false
	}
	funcIsSiDaiSang := func() bool {
		if this.Is_SiDaiSang() == true {
			this.PaiXing = PDK_PX_SiDaiSan
			return true
		}
		return false
	}

	funcIsSiDaiEr := func() bool {
		if this.Is_SiDaiEr() == true {
			this.PaiXing = PDK_PX_SiDaiEr
			return true
		}
		return false
	}

	funcIsFeiJi := func() bool {
		if this.Is_FeiJi() == true {
			this.PaiXing = PDK_PX_FeiJi
			return true
		}
		return false
	}

	funIsSanDaiEr := func() bool {
		if this.Is_SanDai_Er() == true {
			this.PaiXing = PDK_PX_SanDai_Er
			return true
		}
		return false
	}

	funcIsSanDaiYi := func() bool {
		if this.Is_SanDai_Yi() == true {
			this.PaiXing = PDK_PX_SanDai_Yi
			return true
		}
		return false
	}

	funcIsLianDui := func() bool {
		if this.Is_Lian_Dui() == true {
			this.PaiXing = PDK_PX_LianDui
			return true
		}
		return false
	}

	funcIsYiDui := func() bool {
		if this.Is_Yi_Dui() == true {
			this.PaiXing = PDK_PX_YiDui
			return true
		}
		return false
	}

	funcIsShunZi := func() bool {
		if this.Is_Shun_Zi() == true {
			this.PaiXing = PDK_PX_ShunZi
			return true
		}
		return false
	}

	funcIsDanZhan := func() bool {
		if len(playPai) == 1 {
			this.PaiXingStartDianShu = playPai[0] & 0x0F
			this.PaiXing = PDK_PX_DandZhang
			return true
		}
		return false
	}

	if funcIsZhaDan() == true {
		return true
	}

	switch assignPaiXing {
	case PDK_PX_SiDaiEr:
		return funcIsSiDaiSang()
	case PDK_PX_SiDaiSan:
		return funcIsSiDaiEr()
	case PDK_PX_FeiJi:
		return funcIsFeiJi()
	case PDK_PX_ShunZi:
		return funcIsShunZi()
	case PDK_PX_SanDai_Er:
		return funIsSanDaiEr()
	case PDK_PX_SanDai_Yi:
		return funcIsSanDaiYi()
	case PDK_PX_LianDui:
		return funcIsLianDui()
	case PDK_PX_YiDui:
		return funcIsYiDui()
	case PDK_PX_DandZhang:
		return funcIsDanZhan()
	default:
	}

	if funcIsSiDaiSang() == true {
		return true
	}
	if funcIsSiDaiEr() == true {
		return true
	}
	if funcIsFeiJi() == true {
		return true
	}
	if funcIsShunZi() == true {
		return true
	}
	if funIsSanDaiEr() == true {
		return true
	}
	if funcIsSanDaiYi() == true {
		return true
	}
	if funcIsLianDui() == true {
		return true
	}
	if funcIsYiDui() == true {
		return true
	}
	if funcIsDanZhan() == true {
		return true
	}

	return false
}

func (this *paoDeKuaiLogic) CleanStatus() {
	this.Pai = nil
	this.shouPaiCount, this.PaiXingStartDianShu = 0, 0
	this.SequenceCount = 0
	this.PaiXing = PDK_PX_Invalid
}

func (this *paoDeKuaiLogic) GetPaiXing() int32 {
	return this.PaiXing
}

func (this *paoDeKuaiLogic) SetRule(value interface{}) {
	this.rule = value.(*PDKRule)
}

func (this *paoDeKuaiLogic) Is_ZhaDan() bool {

	// A A A
	if this.rule.Is3ABomb == true && len(this.Pai) == 3 {
		for i := 0; i < len(this.Pai); i++ {
			dianShu := this.Pai[i] & 0x0F
			if dianShu != pokerTable.ADianShu {
				return false
			}
		}
		this.PaiXingStartDianShu = pokerTable.ADianShu
		return true
	}

	// 3 3 3 3
	if len(this.Pai) != 4 {
		return false
	}

	dianShu := this.Pai[0] & 0x0F

	for i := 1; i < 4; i++ {
		if (this.Pai[i] & 0x0F) != dianShu {
			return false
		}
	}
	this.PaiXingStartDianShu = dianShu

	return true
}

func (this *paoDeKuaiLogic) Is_SiDaiEr() bool {
	if this.rule.Is4With2 == false {
		return false
	}
	this._4W2IsDui = false

	isOk := false
	// 常规
	if len(this.Pai) == 6 {
		dianShuMap := map[int8]int8{}
		var dianShu int8
		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			dianShuMap[dianShu] += 1

			if v, _ := dianShuMap[dianShu]; v == 4 {
				this.PaiXingStartDianShu = dianShu
				isOk = true
			}
		}
		if isOk == true && len(dianShuMap) == 2 {
			this._4W2IsDui = true
		}
	}

	if this.rule.Is3ABomb == true && len(this.Pai) == 5 {
		dianShuMap := map[int8]int8{}
		var dianShu int8

		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			dianShuMap[dianShu] += 1
			if v, _ := dianShuMap[dianShu]; v == 3 && dianShu == pokerTable.ADianShu {
				this.PaiXingStartDianShu = dianShu
				isOk = true
			}
		}
		if isOk == true && len(dianShuMap) == 2 {
			this._4W2IsDui = true
		}
	}

	if this.rule.IsShaoDaiTouPao == false {
		return isOk
	}

	if isOk == true {
		return isOk
	}
	if this.shouPaiCount != len(this.Pai) {
		return isOk
	}

	// 特殊
	if len(this.Pai) >= 4 && len(this.Pai) < 6 {
		var tempDianShuArr [256]int
		var dianShu int8

		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			tempDianShuArr[dianShu] += 1
			if tempDianShuArr[dianShu] == 4 {
				this.PaiXingStartDianShu = dianShu
				return true
			}
		}
	}
	if this.rule.Is3ABomb == true &&
		len(this.Pai) >= 3 && len(this.Pai) < 5 {
		var tempDianShuArr [256]int
		var dianShu int8

		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			tempDianShuArr[dianShu] += 1
			if tempDianShuArr[dianShu] == 3 && dianShu == pokerTable.ADianShu {
				this.PaiXingStartDianShu = dianShu
				return true
			}
		}
	}

	return isOk
}

func (this *paoDeKuaiLogic) Is_SiDaiSang() bool {
	if this.rule.Is4With3 == false {
		return false
	}
	this._4W3Is3 = false

	isOk := false

	if len(this.Pai) == 7 {
		dianShuMap := map[int8]int8{}
		var dianShu int8

		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			dianShuMap[dianShu] += 1
			if v, _ := dianShuMap[dianShu]; v == 4 {
				this.PaiXingStartDianShu = dianShu
				isOk = true
			}
		}
		if isOk == true && len(dianShuMap) == 2 {
			this._4W3Is3 = true
		}
	}

	if this.rule.Is3ABomb == true && len(this.Pai) == 6 {
		dianShuMap := map[int8]int8{}
		var dianShu int8

		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			dianShuMap[dianShu] += 1
			if v, _ := dianShuMap[dianShu]; v == 3 && dianShu == pokerTable.ADianShu {
				this.PaiXingStartDianShu = dianShu
				isOk = true
			}
		}
		if isOk == true && len(dianShuMap) == 2 {
			this._4W3Is3 = true
		}
	}

	if this.rule.IsShaoDaiTouPao == false {
		return isOk
	}
	if isOk == true {
		return isOk
	}
	if this.shouPaiCount != len(this.Pai) {
		return isOk
	}

	if len(this.Pai) >= 4 && len(this.Pai) < 7 {
		var tempDianShuArr [256]int
		var dianShu int8

		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			tempDianShuArr[dianShu] += 1
			if tempDianShuArr[dianShu] == 4 {
				this.PaiXingStartDianShu = dianShu
				return true
			}
		}
	}
	if this.rule.Is3ABomb == true &&
		len(this.Pai) >= 3 && len(this.Pai) < 6 {

		var tempDianShuArr [256]int
		var dianShu int8

		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			tempDianShuArr[dianShu] += 1
			if tempDianShuArr[dianShu] == 3 && dianShu == pokerTable.ADianShu {
				this.PaiXingStartDianShu = dianShu
				return true
			}
		}
	}

	return isOk
}

func (this *paoDeKuaiLogic) Is_FeiJi() bool {
	this.SequenceCount = 0

	if len(this.Pai) < 6 {
		return false
	}

	var dianShuArr [128]int8
	dianShu3Arr := make([]int, 0)
	paiInfoMap := map[int8]int8{}
	var dianShu int8

	for i := 0; i < len(this.Pai); i++ {
		dianShu = this.Pai[i] & 0x0F
		dianShuArr[dianShu] += 1

		v, ok := paiInfoMap[dianShu]
		if ok == false {
			paiInfoMap[dianShu] = 1
		} else {
			paiInfoMap[dianShu] += 1
		}
		v, ok = paiInfoMap[dianShu]

		if v == 3 {
			dianShu3Arr = append(dianShu3Arr, int(dianShu))
			delete(paiInfoMap, dianShu)
		}
	}

	if len(dianShu3Arr) < 2 {
		return false
	}

	//从小到大排序
	sort.Sort(sort.IntSlice(dianShu3Arr))

	// 跑得快 没有 222
	// 2 不可以带入飞机,但可以当作翅膀
	//if int8(dianShu3Arr[len(dianShu3Arr)-1]) == pokerTable.MaxDianShu {
	//	dianShu3Arr = dianShu3Arr[:len(dianShu3Arr)-1]
	//}

	// 3A当作炸弹后， 不能当作飞机
	if this.rule.Is3ABomb == true &&
		int8(dianShu3Arr[len(dianShu3Arr)-1]) == pokerTable.ADianShu {
		dianShu3Arr = dianShu3Arr[:len(dianShu3Arr)-1]
	}

	if len(dianShu3Arr) < 2 {
		return false
	}

	_3ArrBak := dianShu3Arr

	//判断翅膀是否合法
	checkFeiJiPaiXing := func(fjArr []int) bool {
		for len(fjArr) > 1 {
			temp := len(this.Pai) - len(fjArr)*3

			if temp < len(fjArr)*2 &&
				this.rule.IsShaoDaiTouPao == true &&
				this.shouPaiCount == len(this.Pai) {

				this.SequenceCount = fjArr[len(fjArr)-1] - fjArr[0] + 1
				this.PaiXingStartDianShu = int8(fjArr[0])
				return true
			}

			if temp == len(fjArr)*2 {
				this.SequenceCount = fjArr[len(fjArr)-1] - fjArr[0] + 1
				this.PaiXingStartDianShu = int8(fjArr[0])
				return true
			}

			if this.rule.Is3With1 == true {
				if temp < len(fjArr)*1 &&
					this.rule.IsShaoDaiTouPao == true &&
					this.shouPaiCount == len(this.Pai) {

					this.SequenceCount = fjArr[len(fjArr)-1] - fjArr[0] + 1
					this.PaiXingStartDianShu = int8(fjArr[0])
					return true
				}

				if temp == len(fjArr)*1 {
					this.SequenceCount = fjArr[len(fjArr)-1] - fjArr[0] + 1
					this.PaiXingStartDianShu = int8(fjArr[0])
					return true
				}
			}

			fjArr = fjArr[1:]
		}
		return false
	}

	// 从 大 到 小 查找
	{
		maxIndex := -1
		arrIndex := len(dianShu3Arr) - 1

		for arrIndex > 0 {
			if maxIndex == -1 {
				// 记录 末尾位置
				maxIndex = arrIndex
			}

			// 是否 连续
			if dianShu3Arr[arrIndex]-1 == dianShu3Arr[arrIndex-1] {
				arrIndex -= 1
				continue
			}

			// 是否 超过 1个连续
			if maxIndex-arrIndex < 1 {
				maxIndex = -1
				arrIndex -= 1
				continue
			}

			if checkFeiJiPaiXing(dianShu3Arr[arrIndex:maxIndex+1]) == true {
				return true
			}
			// 还原
			dianShu3Arr = _3ArrBak
			maxIndex = -1
		}

		if maxIndex != -1 &&
			maxIndex-arrIndex > 0 &&
			checkFeiJiPaiXing(dianShu3Arr[arrIndex:maxIndex+1]) == true {
			return true
		}
	}

	return false
}

func (this *paoDeKuaiLogic) Is_SanDai_Er() bool {

	if len(this.Pai) < 3 || len(this.Pai) > 5 {
		return false
	}

	var (
		tempDianShuArr [256]int8
		dianShu        int8
		card_3         = pokerTable.InvalidPai
	)

	for i := 0; i < len(this.Pai); i++ {
		dianShu = this.Pai[i] & 0x0F
		tempDianShuArr[dianShu] += 1
		if tempDianShuArr[dianShu] == 3 {
			card_3 = this.Pai[i]
			break
		}
	}

	if card_3 == pokerTable.InvalidPai {
		return false
	}

	if this.rule.Is3ABomb == true && (card_3&0x0F) == pokerTable.ADianShu {
		return false
	}

	if len(this.Pai) < 5 {
		if this.rule.IsShaoDaiTouPao == false {
			return false
		}

		// 少带偷跑 最后一手才能出
		if this.shouPaiCount != len(this.Pai) {
			return false
		}
	}

	this.PaiXingStartDianShu = card_3 & 0x0F

	return true
}

func (this *paoDeKuaiLogic) Is_SanDai_Yi() bool {

	if this.rule.Is3With1 == false {
		return false
	}

	if len(this.Pai) < 3 || len(this.Pai) > 4 {
		return false
	}

	var (
		tempDianShuArr [256]int8
		dianShu        int8
		card_3         = pokerTable.InvalidPai
	)

	for i := 0; i < len(this.Pai); i++ {
		dianShu = this.Pai[i] & 0x0F
		tempDianShuArr[dianShu] += 1
		if tempDianShuArr[dianShu] == 3 {
			card_3 = this.Pai[i]
			break
		}
	}

	if card_3 == pokerTable.InvalidPai {
		return false
	}

	if this.rule.Is3ABomb == true && (card_3&0x0F) == pokerTable.ADianShu {
		return false
	}

	if len(this.Pai) < 4 {
		if this.rule.IsShaoDaiTouPao == false {
			return false
		}

		// 少带偷跑 最后一手才能出
		if this.shouPaiCount != len(this.Pai) {
			return false
		}
	}

	this.PaiXingStartDianShu = card_3 & 0x0F

	return true
}

func (this *paoDeKuaiLogic) Is_Lian_Dui() bool {
	this.SequenceCount = 0
	if len(this.Pai) < 4 ||
		(len(this.Pai)%2) != 0 {
		return false
	}

	var tempDianShuArr [256]int8
	var beginIndexDianShu int8 = 0x7F
	var dianShu int8

	//是否有超过两张
	for i := 0; i < len(this.Pai); i++ {
		dianShu = this.Pai[i] & 0x0F
		if dianShu < beginIndexDianShu {
			beginIndexDianShu = dianShu
		}

		tempDianShuArr[dianShu] += 1
		if tempDianShuArr[dianShu] > 2 {
			return false
		}
	}

	//不允许有2
	if tempDianShuArr[pokerTable.MaxDianShu] > 0 {
		return false
	}

	//是否是连着的
	endIndex := beginIndexDianShu + (int8)((len(this.Pai)/2)-1)
	// 连续性 && 数量
	if (endIndex-beginIndexDianShu) < 1 ||
		int(((endIndex-beginIndexDianShu)+1)*2) != len(this.Pai) {
		return false
	}

	for i := beginIndexDianShu; i < endIndex; i++ {
		//是否 不是 2张
		if tempDianShuArr[i] != 2 {
			return false
		}

		if tempDianShuArr[i] != tempDianShuArr[i+1] {
			return false
		}
	}

	this.SequenceCount = int(endIndex - beginIndexDianShu)
	this.PaiXingStartDianShu = beginIndexDianShu
	return true
}

func (this *paoDeKuaiLogic) Is_Yi_Dui() bool {
	if len(this.Pai) == 2 {
		if (this.Pai[0] & 0x0F) == (this.Pai[1] & 0x0F) {
			this.PaiXingStartDianShu = this.Pai[1] & 0x0F
			return true
		}
	}

	return false
}

func (this *paoDeKuaiLogic) Is_Shun_Zi() bool {
	if len(this.Pai) < 5 {
		return false
	}

	var tempDianShuArr [256]int8
	var beginIndexDianShu int8 = 0x7F
	var dianShu int8

	//是否有超过1张
	for i := 0; i < len(this.Pai); i++ {
		dianShu = this.Pai[i] & 0x0F
		if dianShu < beginIndexDianShu {
			beginIndexDianShu = dianShu
		}

		tempDianShuArr[dianShu] += 1
		if tempDianShuArr[dianShu] > 1 {
			return false
		}
	}

	//不允许有2
	if tempDianShuArr[pokerTable.MaxDianShu] > 0 {
		return false
	}

	//是否是连着的
	endIndex := beginIndexDianShu + int8((len(this.Pai))-1)
	if endIndex-beginIndexDianShu < 4 ||
		int((endIndex-beginIndexDianShu)+1) != len(this.Pai) {
		return false
	}

	for i := beginIndexDianShu; i < endIndex; i++ {
		if tempDianShuArr[i] != tempDianShuArr[i+1] {
			return false
		}
	}
	this.SequenceCount = int(endIndex - beginIndexDianShu)
	this.PaiXingStartDianShu = beginIndexDianShu

	return true
}

func (this *paoDeKuaiLogic) ParseYiShouChu(shouPaiCount int32, playPai []int8) bool {
	if this.ParsePaiXing(shouPaiCount, playPai, PDK_PX_Invalid) == false {
		return false
	}
	// 有炸弹时,不能自动出
	paiArr := [128]int8{}
	for _, v := range this.Pai {
		paiArr[v&0x0F] += 1

		if this.rule.Is3ABomb == true {
			if paiArr[pokerTable.ADianShu] >= 3 && shouPaiCount > 3 {
				return false
			}
		}
		if paiArr[v&0x0F] > 3 && shouPaiCount > 4 {
			return false
		}
	}

	return true
}
