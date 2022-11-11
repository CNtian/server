package mjXZDDTable

import (
	"qpGame/game/gameMaJiang"
)

type xzddHuLogic struct {
	paiArr        [gameMaJiang.MaxHuaSe + 1][gameMaJiang.MaxDianShu_9 + 1]int8
	paiArrBak     [gameMaJiang.MaxHuaSe + 1][gameMaJiang.MaxDianShu_9 + 1]int8
	laiZiCountBak int8

	laiZiPai        int8
	laiZiCount      int8
	isSupportZiShun bool
	huPaiCount      int8

	ziMoPai int8 // 自摸的牌
	playPai int8 // 打出的牌

	gameRule *XZDDPlayRule
}

func (this *xzddHuLogic) SetShouPaiInfo(huPaiCount int8, paiArr *[gameMaJiang.MaxHuaSe + 1][gameMaJiang.MaxDianShu_9 + 1]int8) {
	this.paiArr = *paiArr
	//this.laiZiCount = laiZiCount
	this.huPaiCount = huPaiCount

	this.paiArrBak = *paiArr
	//this.laiZiCountBak = laiZiCount
}

func (this *xzddHuLogic) IsDianPaoHu(gameSeat *XZDDSeat, playPai int8) bool {

	var huPaiCount int8

	this.playPai, this.ziMoPai = playPai, gameMaJiang.InvalidPai

	mjSeat := gameSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	paiArr := mjSeat.ShouPai

	this.laiZiCountBak = 0
	if this.laiZiPai != gameMaJiang.InvalidPai {
		this.laiZiCountBak = paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F]
	}

	if playPai != gameMaJiang.InvalidPai {
		paiType := uint8(playPai) >> 4
		paiValue := playPai & 0x0F
		paiArr[paiType][0] += 1
		paiArr[paiType][paiValue] += 1
	}
	for i := gameMaJiang.MinHuaSe; i <= gameMaJiang.MaxHuaSe; i++ {
		huPaiCount += paiArr[i][0]
	}
	this.SetShouPaiInfo(huPaiCount, &paiArr)

	if this.is7Dui() == true {
		return true
	}
	if this.IsHu332() == true {
		return true
	}
	return false
}

func (this *xzddHuLogic) IsZiMoHu(gameSeat *XZDDSeat, moPai int8) bool {

	var huPaiCount int8

	this.playPai, this.ziMoPai = gameMaJiang.InvalidPai, moPai

	mjSeat := gameSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
	paiArr := mjSeat.ShouPai

	this.laiZiCountBak = 0
	if this.laiZiPai != gameMaJiang.InvalidPai {
		this.laiZiCountBak = paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F]
	}

	for i := gameMaJiang.MinHuaSe; i <= gameMaJiang.MaxHuaSe; i++ {
		huPaiCount += paiArr[i][0]
	}

	this.SetShouPaiInfo(huPaiCount, &paiArr)

	if this.is7Dui() == true {
		return true
	}
	if this.IsHu332() == true {
		return true
	}
	return false
}

func (this *xzddHuLogic) isHuPrimaryFindJiang() bool {

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 2 {
				continue
			}
			this.paiArr[huaSe][0] -= 2
			this.paiArr[huaSe][dianShu] -= 2

			//this.paiXingRec.Init(14)
			//this.paiXingRec.PushMul(huaSe*0x10|dianShu, 2)

			if this.combinationAnKeOrShunZi(gameMaJiang.MinHuaSe, gameMaJiang.MinDianShu_1) == 0 {
				return true
			}

			this.paiArr = this.paiArrBak
			this.laiZiCount = this.laiZiCountBak
		}
	}
	return false
}

func (this *xzddHuLogic) combinationAnKeOrShunZi(huaSe, dianShu int8) int8 {
	if huaSe > gameMaJiang.MaxHuaSe {
		return 0
	}

	if dianShu > gameMaJiang.MaxDianShu_9 || this.paiArr[huaSe][0] < 1 {
		return this.combinationAnKeOrShunZi(huaSe+1, gameMaJiang.MinDianShu_1)
	}

	if this.paiArr[huaSe][dianShu] < 1 {
		return this.combinationAnKeOrShunZi(huaSe, dianShu+1)
	}

	var resCombination int8 = -1
	useLaiZiCount := this.isAnKe(huaSe, dianShu)
	if useLaiZiCount >= 0 {
		this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] -= useLaiZiCount
		this.paiArr[this.laiZiPai>>4][0] -= useLaiZiCount
		this.laiZiCount -= useLaiZiCount

		//this.paiXingRec.PushMul(this.laiZiPai, useLaiZiCount*-1)
		//this.paiXingRec.PushMul(huaSe*0x10|dianShu, 3+useLaiZiCount)

		resCombination = this.combinationAnKeOrShunZi(huaSe, dianShu)
		if resCombination < 0 {
			this.paiArr[huaSe][0] += 3 - useLaiZiCount
			this.paiArr[huaSe][dianShu] += 3 - useLaiZiCount

			this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] += useLaiZiCount
			this.paiArr[this.laiZiPai>>4][0] += useLaiZiCount
			this.laiZiCount += useLaiZiCount

			//this.paiXingRec.Pop(3)
		}
	}

	if resCombination < 0 {
		var realUseIndex []int8

		realUseIndex, useLaiZiCount = this.isShunZi(huaSe, dianShu)
		if useLaiZiCount < 0 {
			return -2
		}
		this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] -= useLaiZiCount
		this.paiArr[this.laiZiPai>>4][0] -= useLaiZiCount
		this.laiZiCount -= useLaiZiCount

		//this.paiXingRec.PushMul(this.laiZiPai, useLaiZiCount*-1)
		//for _, v := range realUseIndex {
		//	this.paiXingRec.Push(huaSe*0x10 | v)
		//}

		resCombination = this.combinationAnKeOrShunZi(huaSe, dianShu)
		if resCombination < 0 {
			this.paiArr[huaSe][0] += int8(len(realUseIndex))
			for _, v := range realUseIndex {
				this.paiArr[huaSe][v] += 1
			}
			this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] += useLaiZiCount
			this.paiArr[this.laiZiPai>>4][0] += useLaiZiCount
			this.laiZiCount += useLaiZiCount
			//this.paiXingRec.Pop(3)
		}
	}

	return resCombination
}

func (this *xzddHuLogic) isAnKe(huaSe, dianShu int8) int8 {
	useLaiZiCount := this.paiArr[huaSe][dianShu] - 3
	if useLaiZiCount < 0 {
		if this.laiZiCount+useLaiZiCount < 0 {
			return -1
		}

		this.paiArr[huaSe][0] -= this.paiArr[huaSe][dianShu]
		this.paiArr[huaSe][dianShu] = 0

		return gameMaJiang.AbsInt8(useLaiZiCount)
	}

	this.paiArr[huaSe][0] -= 3
	this.paiArr[huaSe][dianShu] -= 3

	return 0
}

func (this *xzddHuLogic) isShunZi(huaSe, dianShu int8) ([]int8, int8) {
	const ziIndex = gameMaJiang.Zi / 0x10
	if huaSe == ziIndex {
		if this.isSupportZiShun == true {
			return this.isZiShunZi(huaSe, dianShu)
		}
		return make([]int8, 0), -1
	}

	return this.isTSWShunZi(huaSe, dianShu)
}

func (this *xzddHuLogic) isTSWShunZi(huaSe, dianShu int8) ([]int8, int8) {
	laiZiCountBak := this.laiZiCount

	realUseIndex := make([]int8, 0, 3)
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
	if dianShu == gameMaJiang.MaxDianShu_9-1 {
		if checkFunc(dianShu-1) != 0 || checkFunc(dianShu+1) != 0 {
			return realUseIndex, -1
		}
		return realUseIndex, callBackFunc()
	} else if dianShu == gameMaJiang.MaxDianShu_9 { //9  789
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

func (this *xzddHuLogic) isZiShunZi(huaSe, dianShu int8) ([]int8, int8) {

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
	for i := dianShu; i <= gameMaJiang.MaxZiPai; i++ {
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

func (this *xzddHuLogic) is7Dui() bool {
	if this.huPaiCount != 14 {
		return false
	}

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	//this.paiXingRec.Init(14)

	var duiZiCount int8 = 0
	for huaSe := int8(0); huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] == 1 || this.paiArr[huaSe][dianShu] == 3 {
				if this.laiZiCount < 1 {
					return false
				}
				this.paiArr[this.laiZiPai>>4][0] -= 1
				this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] -= 1
				this.laiZiCount -= 1

				duiZiCount += (this.paiArr[huaSe][dianShu] + 1) / 2

				if this.paiArr[huaSe][dianShu] == 1 {
					//this.paiXingRec.PushMul(this.laiZiPai, 1)
					//this.paiXingRec.PushMul(huaSe*0x10|dianShu, 1)
				} else {
					//this.paiXingRec.PushMul(huaSe*0x10|dianShu, 3)
					//this.paiXingRec.PushMul(this.laiZiPai, 1)
				}
			} else {
				duiZiCount += this.paiArr[huaSe][dianShu] / 2

				//this.paiXingRec.PushMul(huaSe*0x10|dianShu, 2)
				//if duiZiCount > 1 {
				//	this.paiXingRec.PushMul(huaSe*0x10|dianShu, 2)
				//}
			}
		}
	}

	if duiZiCount != 7 {
		return false
	}
	return true
}

func (this *xzddHuLogic) IsHu332() bool {

	if this.isHuPrimaryFindJiang() == true {
		return true
	}

	if this.isHuPrimaryFindJiangUseLaiZi1() == true {
		return true
	}

	if this.isHuPrimaryFindJiangUseLaiZi2() == true {
		return true
	}

	return false
}

func (this *xzddHuLogic) isHuPrimaryFindJiangUseLaiZi1() bool {

	if this.laiZiCountBak < 1 {
		return false
	}

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 1 {
				continue
			} else {
				this.paiArr[huaSe][0] -= 1
				this.paiArr[huaSe][dianShu] -= 1

				this.paiArr[this.laiZiPai>>4][0] -= 1
				this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] -= 1
				this.laiZiCount -= 1

				//this.paiXingRec.Init(14)
				//this.paiXingRec.PushMul(this.laiZiPai, 1)
				//this.paiXingRec.PushMul(huaSe*0x10|dianShu, 1)
			}

			res := this.combinationAnKeOrShunZi(gameMaJiang.MinHuaSe, gameMaJiang.MinDianShu_1)
			if res == 0 {
				return true
			}

			this.paiArr = this.paiArrBak
			this.laiZiCount = this.laiZiCountBak
		}
	}
	return false
}

func (this *xzddHuLogic) isHuPrimaryFindJiangUseLaiZi2() bool {
	if this.laiZiCountBak < 2 {
		return false
	}

	//this.paiXingRec.Init(14)
	//this.paiXingRec.PushMul(this.laiZiPai, 2)

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	this.paiArr[this.laiZiPai>>4][0] -= 2
	this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] -= 2
	this.laiZiCount -= 2

	res := this.combinationAnKeOrShunZi(gameMaJiang.MinHuaSe, gameMaJiang.MinDianShu_1)
	if res == 0 {
		return true
	}

	return false
}

func (this *xzddHuLogic) isPengPengHu(info *[]*gameMaJiang.OperationPaiInfo) bool {
	var jiangCount, threeCount int32

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	if this.laiZiCount > 0 {
		this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] = 0
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 1 {
				continue
			}
			if this.paiArr[huaSe][dianShu] == 2 {
				jiangCount += 1
			} else if this.paiArr[huaSe][dianShu] == 3 {
				threeCount += 1
			} else {
				return false
			}

			if jiangCount > 1 {
				return false
			}
		}
	}

	if jiangCount != 1 {
		return false
	}

	if threeCount > 0 {
		return true
	}

	for _, v := range *info {
		if v.OperationPXItem == gameMaJiang.OPX_PENG {
			return true
		}
	}

	return false
}

func (this *xzddHuLogic) isJiangYiSe(info *[]*gameMaJiang.OperationPaiInfo) bool {
	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	if this.laiZiCount > 0 {
		this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] = 0
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 1 {
				continue
			}
			// 2  5  8
			if dianShu != gameMaJiang.MinDianShu_1+1 &&
				dianShu != gameMaJiang.MinDianShu_1+4 &&
				dianShu != gameMaJiang.MinDianShu_1+7 {
				return false
			}
		}
	}

	for _, v := range *info {
		dianShu := v.PaiArr[0] & 0x0F
		if dianShu != gameMaJiang.MinDianShu_1+1 &&
			dianShu != gameMaJiang.MinDianShu_1+4 &&
			dianShu != gameMaJiang.MinDianShu_1+7 {
			return false
		}
	}

	return true
}

func (this *xzddHuLogic) isYaoJiu(info *[]*gameMaJiang.OperationPaiInfo) bool {
	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	if this.laiZiCount > 0 {
		this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] = 0
	}

	for _, v := range *info {
		dianShu := v.PaiArr[0] & 0x0F
		if dianShu != gameMaJiang.MinDianShu_1 &&
			dianShu != gameMaJiang.MaxDianShu_9 {
			return false
		}
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 1 {
				continue
			}
			// 4  5  6
			if dianShu == gameMaJiang.MinDianShu_1+3 ||
				dianShu == gameMaJiang.MinDianShu_1+4 ||
				dianShu == gameMaJiang.MinDianShu_1+5 {
				return false
			}
		}
	}

	if this.is19() == true {
		return true
	}

	return false
}

func (this *xzddHuLogic) isJiangDui(info *[]*gameMaJiang.OperationPaiInfo) bool {
	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	if this.laiZiCount > 0 {
		this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] = 0
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 1 {
				continue
			}
			// 2  5  8
			if dianShu != gameMaJiang.MinDianShu_1+1 &&
				dianShu != gameMaJiang.MinDianShu_1+4 &&
				dianShu != gameMaJiang.MinDianShu_1+7 {
				return false
			}
		}
	}

	for _, v := range *info {
		dianShu := v.PaiArr[0] & 0x0F
		if dianShu != gameMaJiang.MinDianShu_1+1 &&
			dianShu != gameMaJiang.MinDianShu_1+4 &&
			dianShu != gameMaJiang.MinDianShu_1+7 {
			return false
		}
	}

	return true
}

func (this *xzddHuLogic) isQingYiSe(info *[]*gameMaJiang.OperationPaiInfo) bool {

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	if this.laiZiCount > 0 {
		this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] = 0
	}

	mainPaiType := uint8(0xFF)

	for _, v := range *info {
		paiType := uint8(v.PaiArr[0]) >> 4
		if mainPaiType == 0xFF {
			mainPaiType = paiType
			continue
		}
		if mainPaiType != paiType {
			return false
		}
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		if mainPaiType == 0xFF {
			mainPaiType = uint8(huaSe)
			continue
		}
		if mainPaiType != uint8(huaSe) {
			return false
		}
	}

	return true
}

func (this *xzddHuLogic) isHaoHua7Dui() bool {

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	if this.laiZiCount > 0 {
		this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] = 0
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 1 {
				continue
			}
			if this.paiArr[huaSe][dianShu]+this.laiZiCount >= 4 {
				return true
			}
		}
	}
	return false
}

func (this *xzddHuLogic) isKaWuXing(moPai, playPai int8) bool {
	tempPai := gameMaJiang.InvalidPai
	if moPai != gameMaJiang.InvalidPai {
		tempPai = moPai
	}
	if playPai != gameMaJiang.InvalidPai {
		tempPai = playPai
	}

	paiType := uint8(tempPai) >> 4
	if int8(paiType) == gameMaJiang.Zi {
		return false
	}
	paiValue := tempPai & 0x0F
	if paiValue != 5 {
		return false
	}

	tempBak := this.paiArrBak
	this.paiArr = this.paiArrBak

	if this.paiArr[paiType][paiValue-1] > 0 {
		this.paiArr[paiType][paiValue-1] -= 1
	} else {
		return false
	}
	if this.paiArr[paiType][paiValue+1] > 0 {
		this.paiArr[paiType][paiValue+1] -= 1
	} else {
		return false
	}

	this.paiArr[paiType][paiValue] -= 1
	this.paiArr[paiType][0] -= 3

	this.paiArrBak = this.paiArr
	isHu := this.IsHu332()
	this.paiArrBak = tempBak
	return isHu
}

func (this *xzddHuLogic) genCount(info *[]*gameMaJiang.OperationPaiInfo) int64 {
	var count int64
	for _, v := range *info {
		if v.OperationPXItem == gameMaJiang.OPX_AN_GANG ||
			v.OperationPXItem == gameMaJiang.OPX_BU_GANG ||
			v.OperationPXItem == gameMaJiang.OPX_MING_GANG {
			count += 1
		}
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] > 3 {
				count += 1
			}
		}
	}
	return count
}

func (this *xzddHuLogic) is19() bool {

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for _, v := range []int8{gameMaJiang.MinDianShu_1, gameMaJiang.MaxDianShu_9} {
			this.paiArr = this.paiArrBak
			this.laiZiCount = this.laiZiCountBak

			if this.paiArr[huaSe][v] >= 2 {
				this.paiArr[huaSe][v] -= 2
				this.paiArr[huaSe][0] -= 2
			} else {
				continue
			}

			if this.combinationAnKeOrShunZi(gameMaJiang.MinHuaSe, gameMaJiang.MinDianShu_1) == 0 {
				return true
			}
		}
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for _, v := range []int8{gameMaJiang.MinDianShu_1, gameMaJiang.MaxDianShu_9} {
			this.paiArr = this.paiArrBak
			this.laiZiCount = this.laiZiCountBak

			if this.paiArr[huaSe][v] > 0 && this.laiZiCount > 0 {
				this.paiArr[huaSe][v] -= 1
				this.paiArr[huaSe][0] -= 1
				this.laiZiCount -= 1 // 单张 + 1张赖子
			} else {
				continue
			}

			if this.combinationAnKeOrShunZi(gameMaJiang.MinHuaSe, gameMaJiang.MinDianShu_1) == 0 {
				return true
			}
		}
	}

	if this.laiZiCountBak >= 2 {
		this.paiArr = this.paiArrBak
		this.laiZiCount = this.laiZiCountBak
		this.laiZiCount -= 2 // 一对赖子将

		if this.combinationAnKeOrShunZi(gameMaJiang.MinHuaSe, gameMaJiang.MinDianShu_1) == 0 {
			for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
				if this.paiArr[huaSe][0] < 1 {
					continue
				}
				if this.paiArr[huaSe][gameMaJiang.MinDianShu_1+1] > 0 || // 2
					this.paiArr[huaSe][gameMaJiang.MinDianShu_1+2] > 0 || // 3
					this.paiArr[huaSe][gameMaJiang.MaxDianShu_9-1] > 0 || // 8
					this.paiArr[huaSe][gameMaJiang.MaxDianShu_9-2] > 0 { // 7
					return false
				}

				return true
			}
		}
	}

	return false
}

func (this *xzddHuLogic) isMenQing(info *[]*gameMaJiang.OperationPaiInfo) bool {

	for _, v := range *info {
		if v.OperationPXItem != gameMaJiang.OPX_AN_GANG {
			return false
		}
	}
	return true
}

func (this *xzddHuLogic) isZhongZhang(info *[]*gameMaJiang.OperationPaiInfo) bool {
	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	if this.laiZiCount > 0 {
		this.paiArr[this.laiZiPai>>4][this.laiZiPai&0x0F] = 0
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}
		if this.paiArr[huaSe][gameMaJiang.MinDianShu_1] > 0 ||
			this.paiArr[huaSe][gameMaJiang.MaxDianShu_9] > 0 {
			return false
		}
	}

	for _, v := range *info {
		dianShu := v.PaiArr[0] & 0x0F
		if dianShu == gameMaJiang.MinDianShu_1 ||
			dianShu == gameMaJiang.MaxDianShu_9 {
			return false
		}
	}

	return true
}

func (this *xzddHuLogic) isTingPai(seat *XZDDSeat) bool {

	for huase := gameMaJiang.MinHuaSe; huase <= gameMaJiang.MaxHuaSe; huase++ {
		for dianshu := gameMaJiang.MinDianShu_1; dianshu <= gameMaJiang.MaxDianShu_9; dianshu++ {
			pai := huase*0x10 | dianshu
			if this.IsZiMoHu(seat, pai) == true {
				return true
			}
		}
	}
	return false
}
