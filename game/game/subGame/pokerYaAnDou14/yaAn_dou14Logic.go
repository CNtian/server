package pokerYaAnD14

type huDuiZiInfo struct {
	isAllHeiPai  bool
	isAllHongPai bool
	duiZi_       int
	longDui      bool

	hongDian, heiDian int
}

type hu14Info struct {
	isAllHeiPai  bool
	isAllHongPai bool
	isHu         bool

	hongDian, heiDian int
}

type dou14Logic struct {
	rule *Dou14Rule

	_huDuiZi huDuiZiInfo
	_hu14    hu14Info
}

func (this *dou14Logic) CleanStatus() {
}

func (this *dou14Logic) SetRule(value *Dou14Rule) {
	this.rule = value
}

func addDianShu(hongDian, heidian *int, pai int8) {
	hua := pai & 0x70
	value := pai & 0x0F

	v := 0
	// < J
	if value < 0x0B {
		v += int(value)
	} else {
		v += 1
	}
	if hua == HongTao || hua == FangKuai {
		*hongDian += v
	} else {
		*heidian += v
	}
}

func (this *dou14Logic) isHuPai(shouPaiMap map[int8]PaiMap, playPai int8) {

	this._hu14 = hu14Info{}
	this._huDuiZi = huDuiZiInfo{}

	paiCount := 0
	//hongDian, heiDian := 0, 0

	shouPai_1 := [16][80]int8{}
	shouPai_2 := [16][80]int8{}

	// 添加备份牌
	for _, v := range shouPaiMap {
		for k, _ := range v {
			paiCount += 1

			d, h := k&0x0F, k&0x70
			shouPai_1[d][h] = 1
			shouPai_1[d][79] += 1
		}
	}

	if playPai != InvalidPai {
		paiCount += 1
		d, h := playPai&0x0F, playPai&0x70
		shouPai_1[d][h] += 1
		shouPai_1[d][79] += 1
	}

	if paiCount%2 != 0 {
		return
	}

	shouPai_2 = shouPai_1
	paiTypeArr := []int8{HeiTao, HongTao, MeiHua, FangKuai}

	isHuDuiInfo := huDuiZiInfo{isAllHongPai: true, isAllHeiPai: true}
	// 对子  手牌数量
	if paiCount == 8 {
		isOK := true
		for aV := int8(1); aV < 0x0E && isOK; {
			if shouPai_1[aV][79] < 1 {
				aV++
				continue
			}
			if shouPai_1[aV][79] == 3 || shouPai_1[aV][79] == 1 {
				isOK = false
				break
			}
			if shouPai_1[aV][79] == 4 {
				isHuDuiInfo.longDui = true
			}
			for _, hua := range paiTypeArr {
				if shouPai_1[aV][hua] < 1 {
					continue
				}
				addDianShu(&isHuDuiInfo.hongDian, &isHuDuiInfo.heiDian, hua|aV)
				shouPai_1[aV][hua] -= 1

				switch hua {
				case FangKuai, HongTao:
					isHuDuiInfo.isAllHeiPai = false
				case HeiTao, MeiHua:
					isHuDuiInfo.isAllHongPai = false
				}
			}

			isHuDuiInfo.duiZi_ += int(shouPai_1[aV][79]) / 2
			shouPai_1[aV][79] = 0
		}
		if isOK == true {
			this._huDuiZi = isHuDuiInfo
		} else {
			this._huDuiZi = huDuiZiInfo{}
		}
	}

	isHu14Info := hu14Info{isAllHongPai: true, isAllHeiPai: true}
	shouPai_1 = shouPai_2
	//hongDian, heiDian = 0, 0

	// A-K
	for aV := int8(1); aV < 0x0E; {
		if shouPai_1[aV][79] < 1 {
			aV++
			continue
		}
		aHua := int8(-1)
		for _, v := range paiTypeArr {
			if shouPai_1[aV][v] > 0 {
				aHua = v
				break
			}
		}
		if aHua < 0 {
			return //hongDian, heiDian
		}

		shouPai_1[aV][aHua] -= 1
		shouPai_1[aV][79] -= 1

		bV := 14 - aV
		if shouPai_1[bV][79] < 1 {
			return //hongDian, heiDian
		}

		find := false
		for _, bHua := range paiTypeArr {
			if shouPai_1[bV][bHua] < 1 {
				continue
			}
			shouPai_1[bV][bHua] -= 1
			shouPai_1[bV][79] -= 1

			switch aHua {
			case FangKuai, HongTao:
				isHu14Info.isAllHeiPai = false
			case HeiTao, MeiHua:
				isHu14Info.isAllHongPai = false
			}

			switch bHua {
			case FangKuai, HongTao:
				isHu14Info.isAllHeiPai = false
			case HeiTao, MeiHua:
				isHu14Info.isAllHongPai = false
			}
			addDianShu(&isHu14Info.hongDian, &isHu14Info.heiDian, aHua|aV)
			addDianShu(&isHu14Info.hongDian, &isHu14Info.heiDian, bHua|bV)

			find = true
			break
		}
		if find == false {
			this._hu14 = hu14Info{}
			return //hongDian, heiDian
		}
	}

	isHu14Info.isHu = true
	this._hu14 = isHu14Info
	//return hongDian, heiDian
}

type huCallBack func(*PokerDou14Seat, int8) bool

func (this *dou14Logic) HasBao(seat_ *PokerDou14Seat, fun huCallBack) bool {

	if seat_.IsExistHuanPai() != InvalidPai {
		return false
	}

	for dianShu := MinDianShu; dianShu <= MaxDianShu; dianShu++ {
		if fun(seat_, FangKuai|dianShu) {
			return true
		}
	}

	return false
}

func (this *dou14Logic) BankerHasBao(dou14Seat *PokerDou14Seat, fun huCallBack) bool {

	if dou14Seat.IsExistHuanPai() != InvalidPai {
		return false
	}

	shouPai_ := dou14Seat.GetShouPaiBak()

	for _, spV := range shouPai_ {
		bak_ := InvalidPai
		for k, _ := range spV {
			bak_ = k
			break
		}

		delete(spV, bak_)

		if this.HasBao(dou14Seat, fun) {
			return true
		}

		spV[bak_] = 1
	}

	return false
}

func (this *dou14Logic) CheckBankerBao(dou14Seat *PokerDou14Seat, playPai int8, fun huCallBack) bool {
	shouPai_ := dou14Seat.GetShouPaiBak()
	v, ok := shouPai_[playPai&0x0F]
	if ok == true {
		delete(v, playPai)
	}

	return this.HasBao(dou14Seat, fun)
}
