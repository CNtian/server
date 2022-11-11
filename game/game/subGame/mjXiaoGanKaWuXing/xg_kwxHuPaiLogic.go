package mjXiaoGanKWXTable

import (
	"qpGame/game/gameMaJiang"
	"qpGame/qpTable"
)

type kwxHuLogic struct {
	paiArr        [gameMaJiang.MaxHuaSe + 1][gameMaJiang.MaxDianShu_9 + 1]int8
	paiArrBak     [gameMaJiang.MaxHuaSe + 1][gameMaJiang.MaxDianShu_9 + 1]int8
	laiZiCountBak int8

	laiZiPai        int8
	laiZiCount      int8
	isSupportZiShun bool
	huPaiCount      int8

	ziMoPai int8 // 自摸的牌
	playPai int8 // 打出的牌
	//kouPai     map[int8]int8 // 扣牌
	paiXingRec gameMaJiang.Stack
	gameRule   *KWXPlayRule
}

func (this *kwxHuLogic) SetShouPaiInfo(huPaiCount int8, paiArr *[gameMaJiang.MaxHuaSe + 1][gameMaJiang.MaxDianShu_9 + 1]int8) {
	this.paiArr = *paiArr
	this.huPaiCount = huPaiCount

	this.paiArrBak = *paiArr
}

func (this *kwxHuLogic) IsDianPaoHu(kwxSeat *KWXSeat, playPai int8) bool {

	readyHuFunc := func(isIgnoreKou bool) {
		var huPaiCount int8

		this.playPai, this.ziMoPai = playPai, gameMaJiang.InvalidPai

		mjSeat := kwxSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
		shouPaiBak := mjSeat.ShouPai

		if isIgnoreKou == false {
			// 扣牌去掉
			for k, _ := range kwxSeat.KouMap {
				huaSe := uint8(k) >> 4
				if shouPaiBak[huaSe][k&0x0F] >= 3 {
					shouPaiBak[huaSe][0] -= 3
					shouPaiBak[huaSe][k&0x0F] -= 3
				}
			}
		}

		paiArr := shouPaiBak //mjSeat.ShouPai

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
	}

	readyHuFunc(false)

	if this.is7Dui() == true {
		return true
	}
	if this.IsHu332() == true {
		if kwxSeat.KouMap != nil {
			readyHuFunc(true)
		}
		return true
	}
	return false
}

func (this *kwxHuLogic) IsZiMoHu(kwxSeat *KWXSeat, moPai int8) bool {

	readyHuFunc := func(isIgnoreKou bool) {
		var huPaiCount int8

		this.playPai, this.ziMoPai = gameMaJiang.InvalidPai, moPai

		mjSeat := kwxSeat.GetXSeatData(0).(*gameMaJiang.MJSeat)
		shouPaiBak := mjSeat.ShouPai

		if isIgnoreKou == false {
			// 扣牌去掉
			for k, _ := range kwxSeat.KouMap {
				huaSe := uint8(k) >> 4
				if shouPaiBak[huaSe][k&0x0F] >= 3 {
					shouPaiBak[huaSe][0] -= 3
					shouPaiBak[huaSe][k&0x0F] -= 3
				}
			}
		}

		paiArr := shouPaiBak //mjSeat.ShouPai

		for i := gameMaJiang.MinHuaSe; i <= gameMaJiang.MaxHuaSe; i++ {
			huPaiCount += paiArr[i][0]
		}
		this.SetShouPaiInfo(huPaiCount, &paiArr)
	}

	readyHuFunc(false)

	if this.is7Dui() == true {
		return true
	}
	if this.IsHu332() == true {
		if kwxSeat.KouMap != nil {
			readyHuFunc(true)
		}
		return true
	}
	return false
}

func (this *kwxHuLogic) isHu() bool {
	if this.is7Dui() == true {
		return true
	}
	return this.IsHu332()
}

func (this *kwxHuLogic) GetHuPaiXing() []int8 {
	return this.paiXingRec.GetPaiArr()
}

func (this *kwxHuLogic) isHuPrimaryFindJiang() bool {
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

			this.paiXingRec.Init(14)
			this.paiXingRec.PushMul(huaSe*0x10|dianShu, 2)

			if this.combinationAnKeOrShunZi(gameMaJiang.MinHuaSe, gameMaJiang.MinDianShu_1) == 0 {
				return true
			}

			this.paiArr = this.paiArrBak
			this.laiZiCount = this.laiZiCountBak
		}
	}
	return false
}

func (this *kwxHuLogic) combinationAnKeOrShunZi(huaSe, dianShu int8) int8 {
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

func (this *kwxHuLogic) isAnKe(huaSe, dianShu int8) int8 {
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

func (this *kwxHuLogic) isShunZi(huaSe, dianShu int8) ([]int8, int8) {
	const ziIndex = gameMaJiang.Zi / 0x10
	if huaSe == ziIndex {
		if this.isSupportZiShun == true {
			return this.isZiShunZi(huaSe, dianShu)
		}
		return make([]int8, 0), -1
	}

	return this.isTSWShunZi(huaSe, dianShu)
}

func (this *kwxHuLogic) isTSWShunZi(huaSe, dianShu int8) ([]int8, int8) {
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

func (this *kwxHuLogic) isZiShunZi(huaSe, dianShu int8) ([]int8, int8) {

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

func (this *kwxHuLogic) is7Dui() bool {
	if this.huPaiCount != 14 {
		return false
	}

	this.paiArr = this.paiArrBak
	this.laiZiCount = this.laiZiCountBak

	this.paiXingRec.Init(14)

	var duiZiCount int8 = 0
	for huaSe := int8(0); huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
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

func (this *kwxHuLogic) IsHu332() bool {

	if this.isHuPrimaryFindJiang() == true {
		return true
	}

	return false
}

func (this *kwxHuLogic) isKaWuXing(moPai, playPai int8) bool {
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

func (this *kwxHuLogic) isPengPengHu(info *[]*gameMaJiang.OperationPaiInfo) bool {
	var jiangCount, threeCount int32

	this.paiArr = this.paiArrBak

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

func (this *kwxHuLogic) isMingSiGui(info *[]*gameMaJiang.OperationPaiInfo, isQingYiSe bool) bool {
	this.paiArr = this.paiArrBak

	// 操作区域的牌 + 手中的牌
	for _, v := range *info {
		if v.OperationPXItem == gameMaJiang.OPX_PENG {
			paiType := uint8(v.PaiArr[0]) >> 4
			paiValue := uint8(v.PaiArr[0]) & 0x0F

			// 手牌是否含有
			if this.paiArr[paiType][paiValue] < 1 {
				continue
			}

			return true
			//if this.gameRule.IsQuanPinDao == true {
			//	return true
			//} else {
			//	if this.ziMoPai == v.PaiArr[0] || this.playPai == v.PaiArr[0] {
			//		return true
			//	}
			//	if isQingYiSe == true {
			//		return true
			//	}
			//}
		}
	}
	return false
}

func (this *kwxHuLogic) isLiangDao(kwxSeat *KWXSeat) bool {
	if len(kwxSeat.LiangDaoMap) > 0 {
		return true
	}
	return false
}

func (this *kwxHuLogic) isGangShangHua(winMJSeat *gameMaJiang.MJSeat) bool {
	if winMJSeat.LianGangCount > 0 {
		return true
	}
	return false
}

func (this *kwxHuLogic) isGangShangPao(winSeatNum, gangSeatNum, playPaiSeatNum qpTable.SeatNumber) bool {
	if gangSeatNum != qpTable.INVALID_SEAT_NUMBER &&
		winSeatNum != gangSeatNum &&
		playPaiSeatNum == gangSeatNum {
		return true
	}
	return false
}

func (this *kwxHuLogic) isQiangGangHu(winSeatNum, buGangSeatNum qpTable.SeatNumber) bool {
	if buGangSeatNum != qpTable.INVALID_SEAT_NUMBER &&
		winSeatNum != buGangSeatNum {
		return true
	}
	return false
}

// paiCount:牌堆剩余牌数
func (this *kwxHuLogic) isHaiDiLao(paiCount int32) bool {
	if paiCount < 1 {
		return true
	}
	return false
}

func (this *kwxHuLogic) isAnSiGui(isQingYiSe bool) bool {
	this.paiArr = this.paiArrBak

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] != 4 {
				continue
			}
			return true
			//if this.gameRule.IsQuanPinDao == true {
			//	return true
			//} else {
			//	// 必须是胡的那张牌
			//	pai := (huaSe * 0x10) | dianShu
			//	if pai == this.ziMoPai || pai == this.playPai {
			//		return true
			//	}
			//	if isQingYiSe == true {
			//		return true
			//	}
			//}
		}
	}
	return false
}

func (this *kwxHuLogic) isQingYiSe(info *[]*gameMaJiang.OperationPaiInfo) bool {

	this.paiArr = this.paiArrBak

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

func (this *kwxHuLogic) isShouZhuaYi(kwxSeat *KWXSeat) bool {
	if len(kwxSeat.GetXSeatData(0).(*gameMaJiang.MJSeat).GetShouPai()) < 3 {
		return true
	}

	return false
}

// 大小三元
// ():大3元,小3元
func (this *kwxHuLogic) isDaXiaoSanYuan(info *[]*gameMaJiang.OperationPaiInfo) (bool, bool) {

	this.paiArr = this.paiArrBak

	var jiangCount, threeCount int
	tempZiHuaSe := gameMaJiang.Zi >> 4

	for _, v := range *info {
		tempHuaSe := v.PaiArr[0] >> 4
		if tempHuaSe == tempZiHuaSe {
			threeCount += 1
		}
	}

	if threeCount >= 3 {
		return true, false
	}

	if this.paiArr[tempZiHuaSe][0] < 1 {
		return false, false
	}
	for i := gameMaJiang.MinDianShu_1; i <= gameMaJiang.MaxZiPai; i++ {
		if this.paiArr[tempZiHuaSe][0] < 1 {
			break
		}
		if this.paiArr[tempZiHuaSe][i] == 2 {
			jiangCount += 1
			this.paiArr[tempZiHuaSe][0] -= 2
		} else if this.paiArr[tempZiHuaSe][i] == 3 {
			threeCount += 1
			this.paiArr[tempZiHuaSe][0] -= 3
		}
	}

	if threeCount >= 3 {
		return true, false
	}

	if threeCount > 1 && jiangCount == 1 {
		return false, true
	}
	return false, false
}

func (this *kwxHuLogic) isHaoHua7Dui() bool {

	this.paiArr = this.paiArrBak

	temp4Count := 0

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] < 1 {
				continue
			}
			if this.paiArr[huaSe][dianShu] == 4 {
				temp4Count += 1
			}
			if temp4Count > 0 {
				return true
			}
		}
	}
	return false
}

func (this *kwxHuLogic) is9LianBaoDeng() bool {

	if this.huPaiCount < 14 {
		return false
	}

	tempHuPaiType := int8(0)
	tempHuPaiValue := int8(0)
	if this.playPai != gameMaJiang.InvalidPai {
		tempHuPaiType = this.playPai >> 4
		tempHuPaiValue = this.playPai & 0x0F
	}
	if this.ziMoPai != gameMaJiang.InvalidPai {
		tempHuPaiType = this.ziMoPai >> 4
		tempHuPaiValue = this.ziMoPai & 0x0F
	}

	if tempHuPaiType == gameMaJiang.Zi {
		return false
	}

	tempPaiArrBakBak := this.paiArrBak
	tempPaiArrBak := this.paiArrBak

	// 去掉胡的牌
	if tempHuPaiValue != 0 {
		tempPaiArrBak[tempHuPaiType][0] -= 1
		tempPaiArrBak[tempHuPaiType][tempHuPaiValue] -= 1
	}

	for i := gameMaJiang.MinDianShu_1; i <= gameMaJiang.MaxDianShu_9; i++ {
		if i == tempHuPaiValue {
			continue
		}
		this.paiArr = tempPaiArrBak
		this.paiArr[tempHuPaiType][0] += 1
		this.paiArr[tempHuPaiType][i] += 1

		this.paiArrBak = this.paiArr

		if this.isHu() == false {
			this.paiArrBak = tempPaiArrBakBak
			return false
		}
	}
	this.paiArrBak = tempPaiArrBakBak

	return true
}

func (this *kwxHuLogic) shuKang(info *[]*gameMaJiang.OperationPaiInfo) int32 {
	this.paiArr = this.paiArrBak

	kanShu := int32(0)
	// 操作区域的牌 + 手中的牌
	for _, v := range *info {
		if v.OperationPXItem == gameMaJiang.OPX_AN_GANG ||
			v.OperationPXItem == gameMaJiang.OPX_BU_GANG ||
			v.OperationPXItem == gameMaJiang.OPX_MING_GANG {
			kanShu += 1
			continue
		}
		if v.OperationPXItem != gameMaJiang.OPX_PENG {
			continue
		}

		paiType := uint8(v.PaiArr[0]) >> 4
		paiValue := uint8(v.PaiArr[0]) & 0x0F

		// 手牌是否含有
		if this.paiArr[paiType][paiValue] < 1 {
			continue
		}

		kanShu += 1
	}

	for huaSe := gameMaJiang.MinHuaSe; huaSe <= gameMaJiang.MaxHuaSe; huaSe++ {
		if this.paiArr[huaSe][0] < 1 {
			continue
		}

		for dianShu := gameMaJiang.MinDianShu_1; dianShu <= gameMaJiang.MaxDianShu_9; dianShu++ {
			if this.paiArr[huaSe][dianShu] != 4 {
				continue
			}
			kanShu += 1
		}
	}

	return kanShu
}

func (this *kwxHuLogic) isTingPai(shouPai [gameMaJiang.MaxHuaSe + 1][gameMaJiang.MaxDianShu_9 + 1]int8) map[int8]int8 {

	tingPaiMap := map[int8]int8{}
	for huase := gameMaJiang.MinHuaSe; huase <= gameMaJiang.MaxHuaSe; huase++ {
		if shouPai[huase][0] < 1 {
			continue
		}
		for dianshu := gameMaJiang.MinDianShu_1; dianshu <= gameMaJiang.MaxDianShu_9; dianshu++ {
			shouPai[huase][0] += 1
			shouPai[huase][dianshu] += 1

			this.paiArr = shouPai
			this.paiArrBak = shouPai
			if this.isHu() == true {
				tingPaiMap[huase*0x10+dianshu] = 0
			}
			shouPai[huase][0] -= 1
			shouPai[huase][dianshu] -= 1
		}
	}
	return tingPaiMap
}
