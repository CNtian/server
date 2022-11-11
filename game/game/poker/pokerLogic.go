package pokerTable

/*
import "sort"

type PokerLogic interface {
	ParsePaiXing(shouPai map[int8]int, playPai []int8) bool
	GetPaiXing() int32
	IsGreaterX(paiCount, paiXing int32, startDianShu int8) bool
	SetRule(value interface{})
}

const DDZ_PX_WangZha int32 = 11
const DDZ_PX_ZhaDan int32 = 10
const DDZ_PX_SiDaiEr int32 = 9
const DDZ_PX_FeiJi int32 = 8
const DDZ_PX_SanDai_Yi int32 = 7
const DDZ_PX_SanDai_Er int32 = 6
const DDZ_PX_SanZhang int32 = 5
const DDZ_PX_LianDui int32 = 4
const DDZ_PX_YiDui int32 = 3
const DDZ_PX_ShunZi int32 = 2
const DDZ_PX_DandZhang int32 = 1
const DDZ_PX_Invalid = 0

type DouDiZhuLogic struct {
	Pai                 []int8
	paiXing             int32
	paiXingStartDianShu int8 // 牌型起始牌的点数
}

func (this *DouDiZhuLogic) ParsePaiXing(shouPai map[int8]int, playPai []int8) bool {
	this.Pai = playPai

	if this.Is_WangZha() == true {
		this.paiXing = DDZ_PX_WangZha
		return true
	}
	if this.Is_ZhaDan() == true {
		this.paiXing = DDZ_PX_ZhaDan
		return true
	}
	if this.Is_SiDaiEr() == true {
		this.paiXing = DDZ_PX_SiDaiEr
		return true
	}
	if this.Is_FeiJi() == true {
		this.paiXing = DDZ_PX_FeiJi
		return true
	}
	if this.Is_SanDai_Yi() == true {
		this.paiXing = DDZ_PX_SanDai_Yi
		return true
	}
	if this.Is_SanDai_Er() == true {
		this.paiXing = DDZ_PX_SanDai_Er
		return true
	}
	if this.Is_San_Zhan() == true {
		this.paiXing = DDZ_PX_SanZhang
		return true
	}
	if this.Is_Lian_Dui() == true {
		this.paiXing = DDZ_PX_LianDui
		return true
	}
	if this.Is_Yi_Dui() == true {
		this.paiXing = DDZ_PX_YiDui
		return true
	}
	if this.Is_Shun_Zi() == true {
		this.paiXing = DDZ_PX_ShunZi
		return true
	}
	if this.Is_ZhaDan() == true {
		this.paiXing = DDZ_PX_DandZhang
		return true
	}

	this.paiXing = DDZ_PX_Invalid
	return false
}

func (this *DouDiZhuLogic) GetPaiXing() int32 {
	return this.paiXing
}

func (this *DouDiZhuLogic) SetRule(value interface{}) {

}

func (this *DouDiZhuLogic) IsGreaterX(paiCount, paiXing int32, startDianShu int8) bool {

	if this.paiXing == DDZ_PX_WangZha {
		return true
	}
	if this.paiXing == DDZ_PX_ZhaDan &&
		paiXing != DDZ_PX_ZhaDan {
		return true
	}
	if this.paiXing != paiXing {
		return false
	}

	if this.paiXingStartDianShu <= startDianShu {
		return false
	}

	if len(this.Pai) == int(paiCount) {
		return true
	}

	return false
}

func (this *DouDiZhuLogic) Is_ZhaDan() bool {
	if len(this.Pai) != 4 {
		return false
	}

	dianShu := this.Pai[0] & 0x0F

	for i := 1; i < 4; i++ {
		if (this.Pai[i] & 0x0F) != dianShu {
			return false
		}
	}
	this.paiXingStartDianShu = dianShu

	return true
}

func (this *DouDiZhuLogic) Is_WangZha() bool {

	if len(this.Pai) != 2 {
		return false
	}

	var temp uint8
	temp = uint8(this.Pai[0]) + uint8(this.Pai[1])
	if temp == 0x87 {
		return true
	}

	return false
}

func (this *DouDiZhuLogic) Is_SiDaiEr() bool {
	if len(this.Pai) == 6 {
		var tempDianShuArr [256]int
		var dianShu int8

		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			tempDianShuArr[dianShu] += 1
			if tempDianShuArr[dianShu] == 4 {
				this.paiXingStartDianShu = dianShu
				return true
			}
		}
	}

	if len(this.Pai) == 8 {
		var tempDianShuArr [256]int
		var dianShu int8

		dianShuTypeTotal := make([]int8, 0)

		for i := 0; i < len(this.Pai); i++ {
			dianShu = this.Pai[i] & 0x0F
			tempDianShuArr[dianShu] += 1
			if tempDianShuArr[dianShu] == 1 {
				dianShuTypeTotal = append(dianShuTypeTotal, dianShu)
			}

			if tempDianShuArr[dianShu] == 4 {
				this.paiXingStartDianShu = dianShu
			}
		}

		duiZiCount := 0
		for _, v := range dianShuTypeTotal {
			if tempDianShuArr[v] == 4 {
				continue
			} else if tempDianShuArr[v] == 2 {
				duiZiCount += 1
			} else {
				return false
			}
		}

		if duiZiCount == 2 {
			return true
		}
	}

	return false
}

func (this *DouDiZhuLogic) Is_FeiJi() bool {
	if len(this.Pai) < 6 {
		return false
	}

	var dianShuArr [256]int8
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

	// 2 不可以带入飞机,但可以当作翅膀
	if int8(dianShu3Arr[len(dianShu3Arr)-1]) == MaxDianShu {
		dianShu3Arr = dianShu3Arr[:len(dianShu3Arr)-1]
	}

	if len(dianShu3Arr) < 2 {
		return false
	}

	//是否是连着的
	for i := 0; i < len(dianShu3Arr)-1; i++ {
		if dianShu3Arr[i]+1 != dianShu3Arr[i+1] {
			return false
		}
	}

	//判断翅膀是否合法
	for len(dianShu3Arr) > 1 {
		temp := len(this.Pai) - len(dianShu3Arr)*3

		//飞机不带
		if temp == 0 {
			this.paiXingStartDianShu = int8(dianShu3Arr[0])
			return true
		}

		//飞机带单张
		if temp == len(dianShu3Arr) {
			this.paiXingStartDianShu = int8(dianShu3Arr[0])
			return true
		}

		//飞机带对子
		if temp == len(dianShu3Arr)*2 {
			//必须是一对
			for _, v := range paiInfoMap {
				if v != 2 {
					return false
				}
			}

			this.paiXingStartDianShu = int8(dianShu3Arr[0])
			return true
		}
		dianShu3Arr = dianShu3Arr[1 : len(dianShu3Arr)-1]
	}

	return false
}

func (this *DouDiZhuLogic) Is_SanDai_Er() bool {
	if len(this.Pai) != 5 {
		return false
	}

	var tempDianShuArr [256]int8
	var dianShu int8
	var card_3 = InvalidPai
	paiMap := make(map[int8]int8)

	for i := 0; i < len(this.Pai); i++ {
		dianShu = this.Pai[i] & 0x0F
		tempDianShuArr[dianShu] += 1
		if tempDianShuArr[dianShu] == 3 {
			card_3 = this.Pai[i]
		}

		_, ok := paiMap[dianShu]
		if ok == false {
			paiMap[dianShu] = 1
		} else {
			paiMap[dianShu] += 1
		}
	}

	if len(paiMap) != 2 ||
		card_3 == InvalidPai {
		return false
	}

	delete(paiMap, int8(card_3&0x0F))

	for _, v := range paiMap {
		if v != 2 {
			return false
		}
	}
	this.paiXingStartDianShu = int8(card_3 & 0x0F)

	return true
}

func (this *DouDiZhuLogic) Is_SanDai_Yi() bool {
	if len(this.Pai) != 4 {
		return false
	}

	var tempDianShuArr [256]int8
	var dianShu int8

	for i := 0; i < len(this.Pai); i++ {
		dianShu = this.Pai[i] & 0x0F
		tempDianShuArr[dianShu] += 1
		if tempDianShuArr[dianShu] == 3 {
			this.paiXingStartDianShu = dianShu
			return true
		}
	}
	return false
}

func (this *DouDiZhuLogic) Is_San_Zhan() bool {
	if len(this.Pai) != 3 {
		return false
	}

	var tempDianShuArr [256]int8
	var dianShu int8

	for i := 0; i < len(this.Pai); i++ {
		dianShu = this.Pai[i] & 0x0F
		tempDianShuArr[dianShu] += 1
		if tempDianShuArr[dianShu] == 3 {
			this.paiXingStartDianShu = dianShu
			return true
		}
	}
	return false
}

func (this *DouDiZhuLogic) Is_Lian_Dui() bool {
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
	if tempDianShuArr[MaxDianShu] > 0 {
		return false
	}

	//是否是连着的
	endIndex := beginIndexDianShu + (int8)((len(this.Pai)/2)-1)
	if (endIndex-beginIndexDianShu) < 2 ||
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

	this.paiXingStartDianShu = beginIndexDianShu
	return true
}

func (this *DouDiZhuLogic) Is_Yi_Dui() bool {
	if len(this.Pai) == 2 {
		if (this.Pai[0] & 0x0F) == (this.Pai[1] & 0x0F) {
			this.paiXingStartDianShu = this.Pai[1] & 0x0F
			return true
		}
	}

	return false
}

func (this *DouDiZhuLogic) Is_Shun_Zi() bool {
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
	if tempDianShuArr[MaxDianShu] > 0 {
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
	this.paiXingStartDianShu = beginIndexDianShu

	return true
}
*/
