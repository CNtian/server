package gameMaJiang

func AbsInt8(value int8) int8 {
	v := (value ^ value>>7) - value>>7
	return v
}

type MJHuPaiLogic interface {
	SetRule(value ...interface{})
	IsHu() bool
	GetHuPaiXing() []int8
}

type MJHuBaseLogic struct {
	paiArr        [MaxHuaSe + 1][MaxDianShu_9 + 1]int8
	paiArrBak     [MaxHuaSe + 1][MaxDianShu_9 + 1]int8
	laiZiCountBak int8

	laiZiPai        int8
	laiZiCount      int8
	isSupportZiShun bool
	huPaiCount      int8

	paiXingRec Stack
}

//(supportZiShun bool, laiziPai int8)
func (this *MJHuBaseLogic) SetRule(value ...interface{}) {
	this.isSupportZiShun = value[0].(bool)
	this.laiZiPai = value[1].(int8)
}

func (this *MJHuBaseLogic) SetShouPaiInfo(laiZiCount, huPaiCount int8, paiArr *[MaxHuaSe + 1][MaxDianShu_9 + 1]int8) {
	this.paiArr = *paiArr
	this.laiZiCount = laiZiCount
	this.huPaiCount = huPaiCount

	this.paiArrBak = *paiArr
	this.laiZiCountBak = laiZiCount
}

func (this *MJHuBaseLogic) IsHu() bool {
	if this.is7Dui() == true {
		return true
	}
	return this.IsHu332()
}

func (this *MJHuBaseLogic) GetHuPaiXing() []int8 {
	return this.paiXingRec.GetPaiArr()
}

func (this *MJHuBaseLogic) isHuPrimaryFindJiang() bool {
	for huaSe := MinHuaSe; huaSe <= MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := MinDianShu_1; dianShu <= MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 2 {
				continue
			}
			this.paiArr[huaSe][0] -= 2
			this.paiArr[huaSe][dianShu] -= 2

			this.paiXingRec.Init(14)
			this.paiXingRec.PushMul((huaSe*0x10)|dianShu, 2)

			if this.combinationAnKeOrShunZi(MinHuaSe, MinDianShu_1) == 0 {
				return true
			}

			this.paiArr = this.paiArrBak
			this.laiZiCount = this.laiZiCountBak
		}
	}
	return false
}

func (this *MJHuBaseLogic) combinationAnKeOrShunZi(huaSe, dianShu int8) int8 {
	if huaSe > MaxHuaSe {
		return 0
	}

	if dianShu > MaxDianShu_9 || this.paiArr[huaSe][0] < 1 {
		return this.combinationAnKeOrShunZi(huaSe+1, MinDianShu_1)
	}

	if this.paiArr[huaSe][dianShu] < 1 {
		return this.combinationAnKeOrShunZi(huaSe, dianShu+1)
	}

	var resCombination int8 = -1
	useLaiZiCount := this.isAnKe(huaSe, dianShu)
	if useLaiZiCount >= 0 {
		this.laiZiCount -= useLaiZiCount

		this.paiXingRec.PushMul(this.laiZiPai, useLaiZiCount*-1)
		this.paiXingRec.PushMul(huaSe*0x10|dianShu, 3+useLaiZiCount)

		resCombination = this.combinationAnKeOrShunZi(huaSe, dianShu)
		if resCombination < 0 {
			this.laiZiCount += useLaiZiCount
			this.paiArr[huaSe][0] += 3 - useLaiZiCount
			this.paiArr[huaSe][dianShu] += 3 - useLaiZiCount

			this.paiXingRec.Pop(3)
		}
	}

	if resCombination < 0 {
		var realUseIndex []int8

		realUseIndex, useLaiZiCount = this.isShunZi(huaSe, dianShu)
		if useLaiZiCount < 0 {
			return -2
		}
		this.laiZiCount -= useLaiZiCount

		this.paiXingRec.PushMul(this.laiZiPai, useLaiZiCount*-1)
		for _, v := range realUseIndex {
			this.paiXingRec.Push(huaSe*0x10 | v)
		}

		resCombination = this.combinationAnKeOrShunZi(huaSe, dianShu)
		if resCombination < 0 {
			this.laiZiCount += useLaiZiCount
			this.paiArr[huaSe][0] += int8(len(realUseIndex))
			for _, v := range realUseIndex {
				this.paiArr[huaSe][v] += 1
			}
			this.paiXingRec.Pop(3)
		}
	}

	return resCombination
}

func (this *MJHuBaseLogic) isAnKe(huaSe, dianShu int8) int8 {
	useLaiZiCount := this.paiArr[huaSe][dianShu] - 3
	if useLaiZiCount < 0 {
		if this.laiZiCount+useLaiZiCount < 0 {
			return -1
		}

		this.paiArr[huaSe][0] -= this.paiArr[huaSe][dianShu]
		this.paiArr[huaSe][dianShu] = 0

		return AbsInt8(useLaiZiCount)
	}

	this.paiArr[huaSe][0] -= 3
	this.paiArr[huaSe][dianShu] -= 3

	return 0
}

func (this *MJHuBaseLogic) isShunZi(huaSe, dianShu int8) ([]int8, int8) {
	const ziIndex = Zi / 0x10
	if huaSe == ziIndex {
		if this.isSupportZiShun == true {
			return this.isZiShunZi(huaSe, dianShu)
		}
		return make([]int8, 0), -1
	}

	return this.isTSWShunZi(huaSe, dianShu)
}

func (this *MJHuBaseLogic) isTSWShunZi(huaSe, dianShu int8) ([]int8, int8) {
	laiZiCountBak := this.laiZiCount

	realUseIndex := make([]int8, 0)
	realUseIndex = append(realUseIndex, dianShu)

	callBackFunc := func() int8 {
		this.paiArr[huaSe][0] -= int8(len(realUseIndex))

		for _, v := range realUseIndex {
			this.paiArr[huaSe][v] -= 1
		}
		return this.laiZiCount - laiZiCountBak //init count - surplus count = use count
	}

	checkFunc := func(index int8) int8 {
		if this.paiArr[huaSe][index] < 1 {
			if laiZiCountBak-1 < 0 {
				return -1
			}
			laiZiCountBak -= 1
		} else {
			realUseIndex = append(realUseIndex, index)
		}
		return 0
	}

	//8    789
	if dianShu == MaxDianShu_9-1 {
		if checkFunc(dianShu-1) != 0 || checkFunc(dianShu+1) != 0 {
			return realUseIndex, -1
		}
		return realUseIndex, callBackFunc()
	} else if dianShu == MaxDianShu_9 { //9  789
		if checkFunc(dianShu-2) != 0 || checkFunc(dianShu-1) != 0 {
			return realUseIndex, -1
		}

		return realUseIndex, callBackFunc()
	}

	//[1,7]
	if checkFunc(dianShu+1) != 0 || checkFunc(dianShu+2) != 0 {
		return realUseIndex, -1
	}

	return realUseIndex, callBackFunc()
}

func (this *MJHuBaseLogic) isZiShunZi(huaSe, dianShu int8) ([]int8, int8) {

	laiZiCountBak := this.laiZiCount

	realUseIndex := make([]int8, 0)

	callBackFun := func() int8 {
		this.paiArr[huaSe][0] -= int8(len(realUseIndex))

		for _, v := range realUseIndex {
			this.paiArr[huaSe][v] -= 1
		}
		return this.laiZiCount - laiZiCountBak //init count - surplus count = use count
	}

	//东南西北
	if dianShu < 5 {
		for i := dianShu; i < 5; i++ {
			if this.paiArr[huaSe][i] > 0 {
				realUseIndex = append(realUseIndex, i)
			}
			if len(realUseIndex) >= 3 {
				return realUseIndex, callBackFun()
			}
		}
		if laiZiCountBak < int8(3-len(realUseIndex)) {
			return realUseIndex, -1
		}
		laiZiCountBak -= int8(3 - len(realUseIndex))

		return realUseIndex, callBackFun()
	}
	//中发白
	for i := dianShu; i <= MaxZiPai; i++ {
		if this.paiArr[huaSe][i] > 0 {
			realUseIndex = append(realUseIndex, i)
		}
		if len(realUseIndex) >= 3 {
			return realUseIndex, callBackFun()
		}
	}

	if laiZiCountBak < int8(3-len(realUseIndex)) {
		return realUseIndex, -1
	}
	laiZiCountBak -= int8(3 - len(realUseIndex))

	return realUseIndex, callBackFun()
}

func (this *MJHuBaseLogic) is7Dui() bool {
	if this.huPaiCount != 14 {
		return false
	}

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	this.paiXingRec.Init(14)

	var duiZiCount int8 = 0
	for huaSe := int8(0); huaSe <= MaxHuaSe; huaSe++ {
		for dianShu := MinDianShu_1; dianShu <= MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] == 1 || this.paiArr[huaSe][dianShu] == 3 {
				if this.laiZiCount < 1 {
					return false
				}
				this.laiZiCount -= 1

				duiZiCount += (this.paiArr[huaSe][dianShu] + 1) / 2

				if this.paiArr[huaSe][dianShu] == 1 {
					this.paiXingRec.PushMul(this.laiZiPai, 1)
					this.paiXingRec.PushMul(huaSe*0x10|dianShu, 1)
				} else {
					this.paiXingRec.PushMul(huaSe*0x10|dianShu, 3)
					this.paiXingRec.PushMul(this.laiZiPai, 1)
				}
			} else {
				duiZiCount += this.paiArr[huaSe][dianShu] / 2

				this.paiXingRec.PushMul(huaSe*0x10|dianShu, 2)
				if duiZiCount > 1 {
					this.paiXingRec.PushMul(huaSe*0x10|dianShu, 2)
				}
			}
		}
	}

	if duiZiCount != 7 {
		return false
	}
	return true
}

func (this *MJHuBaseLogic) IsHu332() bool {

	if this.isHuPrimaryFindJiang() == true {
		return true
	}

	if this.laiZiCount < 1 {
		return false
	}

	if this.isHuPrimaryFindJiangUseLaiZi1() == true {
		return true
	}

	if this.isHuPrimaryFindJiangUseLaiZi2() == true {
		return true
	}

	return false
}

func (this *MJHuBaseLogic) isHuPrimaryFindJiangUseLaiZi1() bool {

	if this.laiZiCountBak < 1 {
		return false
	}

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	for huaSe := MinHuaSe; huaSe <= MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := MinDianShu_1; dianShu <= MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 1 {
				continue
			} else {
				this.paiArr[huaSe][0] -= 1
				this.paiArr[huaSe][dianShu] -= 1
				this.laiZiCount -= 1

				this.paiXingRec.Init(14)
				this.paiXingRec.PushMul(this.laiZiPai, 1)
				this.paiXingRec.PushMul(huaSe*0x10|dianShu, 1)
			}

			res := this.combinationAnKeOrShunZi(MinHuaSe, MinDianShu_1)
			if res == 0 {
				return true
			}

			this.paiArr = this.paiArrBak
			this.laiZiCount = this.laiZiCountBak
		}
	}
	return false
}

func (this *MJHuBaseLogic) isHuPrimaryFindJiangUseLaiZi2() bool {
	if this.laiZiCountBak < 1 {
		return false
	}

	this.paiXingRec.Init(14)
	this.paiXingRec.PushMul(this.laiZiPai, 2)

	this.laiZiCount -= 2

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	res := this.combinationAnKeOrShunZi(MinHuaSe, MinDianShu_1)
	if res == 0 {
		return true
	}

	return false
}
