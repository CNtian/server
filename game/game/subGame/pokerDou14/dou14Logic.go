package pokerDouFourteen

type dou14Logic struct {
	rule *Dou14Rule

	laiZiPai_   int8
	isAllRenPai bool
	isAllSuPai  bool
	duiZi_      int
}

func (this *dou14Logic) CleanStatus() {
}

func (this *dou14Logic) SetRule(value *Dou14Rule) {
	this.rule = value
}

func (this *dou14Logic) isHuPai(isXiaoJia bool, shouPaiMap map[int8]PaiMap, playPai int8, isHuDui bool) bool {
	this.laiZiPai_ = InvalidPai
	this.isAllRenPai = true
	this.isAllSuPai = true
	this.duiZi_ = 0

	paiCount := 0

	shouPai_1 := [16][80]int8{}
	shouPai_2 := [16][80]int8{}
	laiZiCount, laiZiCount_2 := int8(0), int8(0)

	// 添加备份牌
	for _, v := range shouPaiMap {
		for k, _ := range v {
			paiCount += 1

			if k == LaiZi {
				laiZiCount = 1
			} else {
				d, h := k&0x0F, k&0x70
				shouPai_1[d][h] = 1
				shouPai_1[d][79] += 1
			}
		}
	}

	if playPai != InvalidPai {
		paiCount += 1
		d, h := playPai&0x0F, playPai&0x70
		shouPai_1[d][h] += 1
		shouPai_1[d][79] += 1
	}

	if paiCount%2 != 0 {
		return false
	}

	shouPai_2 = shouPai_1
	laiZiCount_2 = laiZiCount
	paiTypeArr := []int8{HeiTao, HongTao, MeiHua, FangKuai}

	// 对子  手牌数量
	if isHuDui {
		if (isXiaoJia && paiCount == 6) || paiCount == 8 {
			isOk := true
			for aV := int8(1); aV < 0x0E && isOk; {
				if shouPai_1[aV][79] < 1 {
					aV++
					continue
				}

				if shouPai_1[aV][79] == 1 && laiZiCount > 0 {

					shouPai_1[aV][79] -= 1
					this.duiZi_ += 1

					laiZiCount -= 1
					this.laiZiPai_ = FangKuai | aV
					continue
				}
				if shouPai_1[aV][79] >= 2 {
					shouPai_1[aV][79] -= 2
					this.duiZi_ += 1
					continue
				}

				isOk = false
			}

			if isOk {
				// 大小王
				c := shouPai_1[0x0E][0x40] + shouPai_1[0x0F][0x40]
				if c == 0 {
					return true
				} else if c == 1 && laiZiCount > 0 {
					laiZiCount -= 1
					if shouPai_1[0x0E][0x40] < 1 {
						this.laiZiPai_ = XiaoWang
					}
					if shouPai_1[0x0F][0x40] < 1 {
						this.laiZiPai_ = DaWang
					}
					this.duiZi_ += 1
					return true
				} else if c == 2 {
					this.duiZi_ += 1
					return true
				}
			}
		}
	}

	this.laiZiPai_, this.duiZi_ = InvalidPai, 0
	this.isAllRenPai, this.isAllSuPai = true, true

	shouPai_1, laiZiCount = shouPai_2, laiZiCount_2

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
			return false
		}

		shouPai_1[aV][aHua] -= 1
		shouPai_1[aV][79] -= 1

		bV := 14 - aV
		if shouPai_1[bV][79] < 1 && laiZiCount < 1 {
			return false
		}

		find := false
		for _, v := range paiTypeArr {
			if shouPai_1[bV][v] < 1 {
				continue
			}
			shouPai_1[bV][v] -= 1
			shouPai_1[bV][79] -= 1

			if aV < 0x0B && bV < 0x0B {
				this.isAllRenPai = false
			}
			if aV > 0x0A || bV > 0x0A {
				this.isAllSuPai = false
			}

			find = true
			break
		}
		if find == true {
			continue
		}

		if find == false && laiZiCount > 0 {
			this.laiZiPai_ = FangKuai | bV
			laiZiCount -= 1

			if aV < 0x0B && bV < 0x0B {
				this.isAllRenPai = false
			}
			if aV > 0x0A || bV > 0x0A {
				this.isAllSuPai = false
			}

			continue
		}
		return false
	}

	// 大小王  赖子
	if shouPai_1[0x0E][0x40] > 0 ||
		shouPai_1[0x0F][0x40] > 0 ||
		laiZiCount > 0 {
		return false
	}

	return true
}

func (this *dou14Logic) HasBao(dou14Seat, bankerSeat *PokerDou14Seat, shouPaiMap_ map[int8]PaiMap) bool {

	for dianShu := MinDianShu; dianShu <= MaxDianShu; dianShu++ {
		if this.hasBao(dou14Seat, bankerSeat, shouPaiMap_, FangKuai|dianShu) {
			return true
		}
	}

	if _, ok := dou14Seat.shouPaiMap[0x0E]; ok == false {
		if this.hasBao(dou14Seat, bankerSeat, shouPaiMap_, XiaoWang) {
			return true
		}
	}

	if _, ok := dou14Seat.shouPaiMap[0x0F]; ok == false {
		if this.hasBao(dou14Seat, bankerSeat, shouPaiMap_, DaWang) {
			return true
		}
	}

	return false
}

func (this *dou14Logic) hasBao(dou14Seat, bankerSeat *PokerDou14Seat, shouPaiMap_ map[int8]PaiMap, tempPai int8) bool {

	isHuDui := true
	if len(dou14Seat.touPai) > 0 || len(dou14Seat.gangPai) > 0 {
		isHuDui = false
	}

	if this.isHuPai(dou14Seat.isXiaoJia, shouPaiMap_, tempPai, isHuDui) == false {
		return false
	}

	// 胡 对，不能有偷牌
	if this.duiZi_ > 0 && len(dou14Seat.touPai) > 0 {
		return false
	}

	daWang, xiaoWang := false, false

	shouPaiScore, gangScore := 0, 0
	touScore, diFenScore := 0, 0

	_4Count := make(map[int8]int)
	{
		// 手牌
		for _, v := range shouPaiMap_ {
			for k, _ := range v {
				c := k & 0x0F
				_4Count[c] += 1
				if c > 0x0A && c < 0x0E {
					shouPaiScore += 1
				}
				if c == 0x0E {
					xiaoWang = true
				} else if c == 0x0F {
					daWang = true
				}
			}
		}

		if tempPai != InvalidPai {
			c := tempPai & 0x0F
			_4Count[c] += 1
			if c > 0x0A && c < 0x0E {
				shouPaiScore += 1
			}
			if c == 0x0E {
				xiaoWang = true
			} else if c == 0x0F {
				daWang = true
			}
		}

		if this.laiZiPai_ != InvalidPai {
			c := this.laiZiPai_ & 0x0F
			_4Count[c] += 1
			if c > 0x0A && c < 0x0E {
				shouPaiScore += 1
			}
			if c == 0x0E {
				xiaoWang = true
			} else if c == 0x0F {
				daWang = true
			}
		}

		// 杠
		for _, v := range dou14Seat.gangPai {
			c := v[0] & 0x0F
			_4Count[c] += 1
			if c > 0x0A && c < 0x0E {
				gangScore += 4
			} else {
				gangScore += 2
			}
			if len(v) >= 5 {
				gangScore = +1
			}
		}

		// 偷
		for _, v := range dou14Seat.touPai {
			c := v[0] & 0x0F
			_4Count[c] += 1
			if len(v) == 1 {
				touScore += 1
				if c == 0x0E {
					xiaoWang = true
				} else if c == 0x0F {
					daWang = true
				} else if c == 7 {
					touScore += 1
				}
			}
			if len(v) == 3 {
				if c > 0x0A && c < 0x0E {
					touScore += 3
				} else {
					touScore += 1
				}
			}
			//if len(v) == 1 && v[0] == LaiZi {
			//	touScore += 1
			//	continue
			//}
		}
	}

	long := 0
	for _, v := range _4Count {
		if v == 4 {
			long += 1
		}
	}

	// 对子
	if len(dou14Seat.touPai) < 1 {
		if dou14Seat.isXiaoJia && this.duiZi_ == 3 {
			if (xiaoWang && daWang) || long > 0 {
				for _, v := range shouPaiMap_ {
					if len(v) == 4 {
						diFenScore = 20
						break
					}
				}
			} else {
				diFenScore = 10
			}
		}
		if this.duiZi_ == 4 {
			putOk := false
			if (xiaoWang && daWang) || long > 0 {
				for _, v := range shouPaiMap_ {
					if len(v) == 4 {
						putOk = true
						break
					}
				}
			}
			if len(shouPaiMap_) == 2 {
				putOk = true
				diFenScore = 40
			}
			if putOk == false {
				diFenScore = 10
			}
		}
	}

	if len(dou14Seat.touPai) < 1 && len(dou14Seat.chiPai) < 1 &&
		len(dou14Seat.pengPai) < 1 && len(dou14Seat.gangPai) < 1 {
		// 全素
		if this.isAllSuPai {
			if _, ok := shouPaiMap_[7]; ok == true {
				diFenScore = 10 // 软素
			} else {
				diFenScore = 20 // 硬素
			}
		}
		// 全人
		if this.isAllRenPai {
			diFenScore = 10
		}
	}

	if diFenScore < 1 {
		diFenScore = shouPaiScore + gangScore + touScore
	}

	if dou14Seat.isXiaoJia == true {
		diFenScore *= 2
	}

	return diFenScore >= 2
}

func (this *dou14Logic) BankerHasBao(dou14Seat *PokerDou14Seat) bool {
	shouPai_ := dou14Seat.GetShouPaiBak()

	for _, spV := range shouPai_ {
		bak_ := InvalidPai
		for k, _ := range spV {
			bak_ = k
			break
		}
		if bak_ == DaWang ||
			bak_ == XiaoWang ||
			bak_ == LaiZi {
			continue
		}
		delete(spV, bak_)

		if this.HasBao(dou14Seat, dou14Seat, shouPai_) {
			return true
		}

		spV[bak_] = struct{}{}
	}

	return false
}

func (this *dou14Logic) CheckBankerBao(dou14Seat *PokerDou14Seat, playPai int8) bool {
	shouPai_ := dou14Seat.GetShouPaiBak()
	v, ok := shouPai_[playPai&0x0F]
	if ok == true {
		delete(v, playPai)
	}

	return this.HasBao(dou14Seat, dou14Seat, shouPai_)
}
