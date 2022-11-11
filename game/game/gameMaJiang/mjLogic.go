package gameMaJiang

type MJAction interface {
	IsHu(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, playPai int8) bool
	IsChi(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, pai int8) bool
	IsPeng(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, pai int8) bool
	IsMingGang(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, operPai []*OperationPaiInfo, pai int8) bool
	IsZiMoGang(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, operPai []*OperationPaiInfo) bool
}

type MJBaseAction struct {
	LaiZiPai   int8
	HuPaiLogic MJHuBaseLogic
}

func (this *MJBaseAction) SetRule(supportZiShun bool, laiziPai int8) {
	this.LaiZiPai = laiziPai
	this.HuPaiLogic.SetRule(supportZiShun, laiziPai)
}

func (this *MJBaseAction) IsHu(shouPai *[MaxHuaSe + 1][MaxDianShu_9 + 1]int8, playPai int8) bool {

	var laiZiPaiCount, huPaiCount int8

	paiArr := *shouPai

	if this.LaiZiPai != InvalidPai {
		paiType := uint8(this.LaiZiPai) >> 4
		paiValue := this.LaiZiPai & 0x0F
		if paiArr[paiType][paiValue] > 0 {
			laiZiPaiCount = paiArr[paiType][paiValue]
		}
	}

	if playPai != InvalidPai {
		paiType := uint8(playPai) >> 4
		paiValue := playPai & 0x0F
		paiArr[paiType][0] += 1
		paiArr[paiType][paiValue] += 1
	}
	for i := MinHuaSe; i <= MaxHuaSe; i++ {
		huPaiCount += paiArr[i][0]
	}
	this.HuPaiLogic.SetShouPaiInfo(laiZiPaiCount, huPaiCount, &paiArr)
	return this.HuPaiLogic.IsHu()
}

func (this *MJBaseAction) IsChi(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, pai int8) bool {

	paiType := pai >> 4
	if paiType > (Wan >> 4) {
		return false
	}
	paiValue := pai & 0x0F

	switch paiValue {
	case 1:
		if shouPai[paiType][2] > 0 && shouPai[paiType][3] > 0 {
			return true
		}
	case 2:
		if shouPai[paiType][1] > 0 && shouPai[paiType][3] > 0 {
			return true
		} else if shouPai[paiType][3] > 0 && shouPai[paiType][4] > 0 {
			return true
		}
	case 8:
		if shouPai[paiType][7] > 0 && shouPai[paiType][9] > 0 {
			return true
		} else if shouPai[paiType][6] > 0 && shouPai[paiType][7] > 0 {
			return true
		}
	case 9:
		if shouPai[paiType][7] > 0 && shouPai[paiType][8] > 0 {
			return true
		}
	default:
		if shouPai[paiType][paiValue-2] > 0 && shouPai[paiType][paiValue-1] > 0 {
			return true
		} else if shouPai[paiType][paiValue-1] > 0 && shouPai[paiType][paiValue+1] > 0 {
			return true
		} else if shouPai[paiType][paiValue+1] > 0 && shouPai[paiType][paiValue+2] > 0 {
			return true
		}
	}

	return false
}

func (this *MJBaseAction) IsPeng(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, pai int8) bool {
	huaSeIndex := uint8(pai) >> 4
	if shouPai[huaSeIndex][pai&0x0F] > 1 {
		return true
	}

	return false
}

func (this *MJBaseAction) IsMingGang(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, operPai []*OperationPaiInfo, pai int8) bool {
	huaSeIndex := uint8(pai) >> 4
	if shouPai[huaSeIndex][pai&0x0F] > 2 {
		return true
	}

	return false
}

func (this *MJBaseAction) IsZiMoGang(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, operPai []*OperationPaiInfo) bool {

	for i := MinHuaSe; i <= MaxHuaSe; i++ {
		if shouPai[i][0] < 1 {
			continue
		}
		for j := MinDianShu_1; j <= MaxDianShu_9; j++ {
			if shouPai[i][j] > 3 {
				return true
			}
		}
	}

	for _, v := range operPai {
		if v.OperationPXItem == OPX_PENG {
			pai := v.PaiArr[0]
			huaSeIndex := uint8(pai) >> 4
			if shouPai[huaSeIndex][pai&0x0F] > 0 {
				return true
			}
		}
	}

	return false
}

func (this *MJBaseAction) IsZiMoGang1(shouPai [MaxHuaSe + 1][MaxDianShu_9 + 1]int8, operPai []*OperationPaiInfo, moPai int8) bool {

	for i := MinHuaSe; i <= MaxHuaSe; i++ {
		if shouPai[i][0] < 1 {
			continue
		}
		for j := MinDianShu_1; j <= MaxDianShu_9; j++ {
			if shouPai[i][j] > 3 {
				return true
			}
		}
	}

	// 补杠 过后不补
	for _, v := range operPai {
		if v.OperationPXItem == OPX_PENG {
			pai := v.PaiArr[0]
			huaSeIndex := uint8(pai) >> 4
			if shouPai[huaSeIndex][pai&0x0F] > 0 && moPai == pai {
				return true
			}
		}
	}

	return false
}
