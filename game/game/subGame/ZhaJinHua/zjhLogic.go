package ZhaJinHua

const (
	zjh_BaoZi   = 6
	zjh_ShunJin = 5
	zjh_JinHua  = 4
	zjh_ShunZi  = 3
	zjh_DuiZi   = 2
	zjh_SanPai  = 1
	zjh_NULL    = 0
)

type logicPaiData struct {
	dianShu int8
	huaSe   int8
}

type zhaJinHuaPaiXing struct {
	paiXing int32
	paiArr  []logicPaiData
}

type zhaJinHuaLogic struct {
	rule *ZhaJinHuaRule
}

// ():牌型,最大牌
func (this *zhaJinHuaLogic) GetPaiXing(paiArr []int8) *zhaJinHuaPaiXing {

	a := paiArr[0] & 0x0F
	at := paiArr[0] >> 4

	b := paiArr[1] & 0x0F
	bt := paiArr[1] >> 4

	c := paiArr[2] & 0x0F
	ct := paiArr[2] >> 4

	if a == _2DianShu {
		a = 0x02
	}
	if b == _2DianShu {
		b = 0x02
	}
	if c == _2DianShu {
		c = 0x02
	}

	// 排序
	if a < b {
		a, b = b, a
		at, bt = bt, at
	}
	if b < c {
		b, c = c, b
		bt, ct = ct, bt
	}
	if a < b {
		a, b = b, a
		at, bt = bt, at
	}

	zjhPaiXing := &zhaJinHuaPaiXing{
		paiXing: zjh_SanPai,
		paiArr: []logicPaiData{
			{dianShu: a, huaSe: at},
			{dianShu: b, huaSe: bt},
			{dianShu: c, huaSe: ct}},
	}

	if a == b && b == c {
		zjhPaiXing.paiXing = zjh_BaoZi
		return zjhPaiXing
	}

	// AKQ - 432
	if a-1 == b && b-1 == c {
		if at == bt && at == ct && bt == ct {
			zjhPaiXing.paiXing = zjh_ShunJin
		} else {
			zjhPaiXing.paiXing = zjh_ShunZi
		}
		return zjhPaiXing
	}
	// 32A   	(AKQ > A32 > KQJ)
	if a == ADianShu && b == 0x03 && c == 0x02 {
		if at == bt && at == ct && bt == ct {
			zjhPaiXing.paiXing = zjh_ShunJin
		} else {
			zjhPaiXing.paiXing = zjh_ShunZi
		}

		if this.rule.Second_32A == false {
			zjhPaiXing.paiArr[0].dianShu, zjhPaiXing.paiArr[0].huaSe = b, bt
			zjhPaiXing.paiArr[1].dianShu, zjhPaiXing.paiArr[1].huaSe = c, ct
			zjhPaiXing.paiArr[2].dianShu, zjhPaiXing.paiArr[2].huaSe = a, at
		}

		return zjhPaiXing
	}

	if at == bt && at == ct && bt == ct {
		zjhPaiXing.paiXing = zjh_JinHua
		return zjhPaiXing
	}

	// QQ2
	if a == b {
		zjhPaiXing.paiXing = zjh_DuiZi
		return zjhPaiXing
	}
	// 33A
	if b == c {
		zjhPaiXing.paiXing = zjh_DuiZi
		zjhPaiXing.paiArr[0].dianShu, zjhPaiXing.paiArr[0].huaSe = b, bt
		zjhPaiXing.paiArr[1].dianShu, zjhPaiXing.paiArr[1].huaSe = c, ct
		zjhPaiXing.paiArr[2].dianShu, zjhPaiXing.paiArr[2].huaSe = a, at
		return zjhPaiXing
	}

	return zjhPaiXing
}

func TestAAA(paiArr []int8) interface{} {
	a := paiArr[0] & 0x0F
	at := paiArr[0] >> 4

	b := paiArr[1] & 0x0F
	bt := paiArr[1] >> 4

	c := paiArr[2] & 0x0F
	ct := paiArr[2] >> 4

	if a == _2DianShu {
		a = 0x02
	}
	if b == _2DianShu {
		b = 0x02
	}
	if c == _2DianShu {
		c = 0x02
	}

	// 排序
	if a < b {
		a, b = b, a
		at, bt = bt, at
	}
	if b < c {
		b, c = c, b
		bt, ct = ct, bt
	}
	if a < b {
		a, b = b, a
		at, bt = bt, at
	}

	zjhPaiXing := &zhaJinHuaPaiXing{
		paiXing: zjh_SanPai,
		paiArr: []logicPaiData{
			{dianShu: a, huaSe: at},
			{dianShu: b, huaSe: bt},
			{dianShu: c, huaSe: ct}},
	}

	if a == b && b == c {
		zjhPaiXing.paiXing = zjh_BaoZi
		return zjhPaiXing
	}

	// AKQ - 432
	if a-1 == b && b-1 == c {
		if at == bt && at == ct && bt == ct {
			zjhPaiXing.paiXing = zjh_ShunJin
		} else {
			zjhPaiXing.paiXing = zjh_ShunZi
		}
		return zjhPaiXing
	}
	// 32A   	(AKQ > A32 > KQJ)
	if a == ADianShu && b == 0x03 && c == 0x02 {
		if at == bt && at == ct && bt == ct {
			zjhPaiXing.paiXing = zjh_ShunJin
		} else {
			zjhPaiXing.paiXing = zjh_ShunZi
		}
		return zjhPaiXing
	}

	if at == bt && at == ct && bt == ct {
		zjhPaiXing.paiXing = zjh_JinHua
		return zjhPaiXing
	}

	// QQ2
	if a == b {
		zjhPaiXing.paiXing = zjh_DuiZi
		return zjhPaiXing
	}
	// 33A
	if b == c {
		zjhPaiXing.paiXing = zjh_DuiZi
		zjhPaiXing.paiArr[0].dianShu, zjhPaiXing.paiArr[0].huaSe = b, bt
		zjhPaiXing.paiArr[1].dianShu, zjhPaiXing.paiArr[1].huaSe = c, ct
		zjhPaiXing.paiArr[2].dianShu, zjhPaiXing.paiArr[2].huaSe = a, at
		return zjhPaiXing
	}

	return zjhPaiXing
}

// 发起者是A
// A > B:true  A <= B:false
func (this *zhaJinHuaLogic) initiatorACompareB(A *zhaJinHuaPaiXing, B *zhaJinHuaPaiXing) bool {
	if A.paiXing > B.paiXing {
		return true
	}
	if A.paiXing < B.paiXing {
		return false
	}

	// 同牌型 比 点数
	for i := 0; i < 3; i++ {
		if A.paiArr[i].dianShu > B.paiArr[i].dianShu {
			return true
		}
		if A.paiArr[i].dianShu < B.paiArr[i].dianShu {
			return false
		}
	}

	if this.rule.DaXiao == 0 {
		return false
	}

	// 同牌型 比 花色
	if A.paiXing == zjh_DuiZi {
		if (A.paiArr[0].huaSe*0x10) == HeiTao ||
			(A.paiArr[1].huaSe*0x10) == HeiTao {
			return true
		}
		return false
	}
	if A.paiArr[0].huaSe > B.paiArr[0].huaSe {
		return true
	}

	return false
}

// 游戏结束
// A > B:1  A == B:0  A < B:-1
func (this *zhaJinHuaLogic) compareInGameOver(A *zhaJinHuaPaiXing, B *zhaJinHuaPaiXing) (r int) {

	if A.paiXing > B.paiXing {
		return 1
	}
	if A.paiXing < B.paiXing {
		return -1
	}

	// 同牌型 比 点数
	for i := 0; i < 3; i++ {
		if A.paiArr[i].dianShu > B.paiArr[i].dianShu {
			return 1
		}
		if A.paiArr[i].dianShu < B.paiArr[i].dianShu {
			return -1
		}
	}

	if this.rule.DaXiao == 0 {
		return 0
	}

	// 同牌型 比 花色
	if A.paiXing == zjh_DuiZi {
		if (A.paiArr[0].huaSe*0x10) == HeiTao ||
			(A.paiArr[1].huaSe*0x10) == HeiTao {
			return 1
		}
		return -1
	}
	if A.paiArr[0].huaSe > B.paiArr[0].huaSe {
		return 1
	}
	return -1
}
