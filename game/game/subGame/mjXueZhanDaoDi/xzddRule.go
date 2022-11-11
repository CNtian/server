package mjXZDDTable

import (
	"fmt"
	"qpGame/game/gameMaJiang"
	"qpGame/localConfig"
	"qpGame/qpTable"
	"strconv"
)

type XZDDPlayRule struct {
	RuleJson        string  `json:"-"`         //
	MaxRoundCount   int32   `json:"maxRound"`  // 总局数 8\16
	MaxPlayer       int32   `json:"maxPlayer"` // 最大人数 2\3\4
	Multiple        string  `json:"multiple"`  // 倍数
	MultipleFloat64 float64 `json:"-"`
	FengDingFanShu  int64   `json:"FDFanShu"` // 封顶番数 2\3\4

	WanFa        int32 `json:"wanFa"`        // 1  2 3 4
	ShouPaiCount int32 `json:"shouPaiCount"` // 2人1房 人手牌数量 7\13
	ChangePai    int   `json:"changePai"`    // 换牌 0\3\4

	IsZiMoJiaDi         bool `json:"ziMoJiaDi"`  // true:自摸加底  false:自摸加番
	Is19JiangDui        bool `json:"jiangDui19"` // 幺九将对
	IsMenQingZhongZhang bool `json:"mqzz"`       // 门清中张
	IsTianDiHu          bool `json:"tianDiHu"`   // 天\地胡
	IsDianPaoPingHu     bool `json:"dpPingHu"`   // 点炮平胡
	IsHuJiaoZhuanYi     bool `json:"hjzy"`       // 呼叫转移
	IsPengPengHux2      bool `json:"pengPengx2"` // 碰碰胡x2
	IsKaWuXing          bool `json:"kaWuXing"`   // 卡五星
	IsCaiGua            bool `json:"isCaiGua"`   // 擦挂
	IsHZLaiZi           bool `json:"hzLaiZi"`    // 红中赖子
	IsDGHZiMo           bool `json:"dghZiMo"`    // 点杠花: A出牌,B杠,杠后胡牌
	QingYiSeFS          int  `json:"qysFanShu"`  // 清一色 番数

	// 服务器自定义(与客户端一致)
	Consumables int32 `json:"consumables"` // 消耗(房卡|钻石)
}

func (this *XZDDPlayRule) GetMaxPlayerCount() int32 {
	return this.MaxPlayer
}

func (this *XZDDPlayRule) CheckField() error {
	switch this.MaxRoundCount {
	case 6:
		this.Consumables = 1 //1
	case 8:
		this.Consumables = 1 //1
	case 16:
		this.Consumables = 2 //2
	default:
		return fmt.Errorf("maxRound := %d", this.MaxRoundCount)
	}

	switch this.Multiple {
	case "0.5", "1", "2", "3", "5", "10", "20", "30", "50":
	default:
		return fmt.Errorf("multiple := %s", this.Multiple)
	}
	this.MultipleFloat64, _ = strconv.ParseFloat(this.Multiple, 64)

	switch this.FengDingFanShu {
	case 2, 3, 4:
	default:
		return fmt.Errorf("FDFanShu := %d", this.FengDingFanShu)
	}

	switch this.ChangePai {
	case 0, 3, 4:
	default:
		return fmt.Errorf("changePai := %d", this.ChangePai)
	}

	switch this.WanFa {
	case 1, 2:
		this.MaxPlayer = 2
	case 3:
		this.MaxPlayer = 3
	case 4:
		this.MaxPlayer = 4
	default:
		return fmt.Errorf("wanFa := %d", this.WanFa)
	}

	if this.MaxPlayer == 2 && this.WanFa == 1 {
		if this.ShouPaiCount != 7 && this.ShouPaiCount != 13 {
			return fmt.Errorf("ShouPai2 := %d", this.ShouPaiCount)
		}
		if this.ChangePai != 0 {
			return fmt.Errorf("2player changePai := %d", this.ChangePai)
		}
	} else {
		this.ShouPaiCount = 13
	}
	if this.MaxPlayer < 3 && this.IsHZLaiZi == true {
		return fmt.Errorf("<3 hongZhongLaiZi")
	}

	if this.MaxPlayer == 2 {
		switch this.QingYiSeFS {
		case 0, 1, 2:
		default:
			return fmt.Errorf("qysFanShu := %d", this.QingYiSeFS)
		}
	} else {
		this.QingYiSeFS = 2
	}

	return nil
}

func newMXZDDSeat(playerID qpTable.PlayerID, seatNumber qpTable.SeatNumber) (qpTable.QPSeat, error) {
	baseSeat := qpTable.NewQPSeat(playerID, seatNumber)
	mjSeat := &gameMaJiang.MJSeat{SeatData: baseSeat}

	kwxSeat := &XZDDSeat{MJSeat: mjSeat}
	kwxSeat.CleanRoundData()

	return kwxSeat, nil
}

func NewGameXZDDTable(tableNum int32, gameRuleCfg, tableCfg string) (*GameXZDDTable, int32, string) {
	baseTable, rspCode, err := qpTable.NewQPTable(tableNum, tableCfg, newMXZDDSeat)
	if err != nil {
		return nil, rspCode, err.Error()
	}

	t := GameXZDDTable{}
	t.playPaiLogic.CleanRoundData()
	rspCode, err = t.ParseTableOptConfig(gameRuleCfg)
	if rspCode != 0 {
		return nil, rspCode, err.Error()
	}

	baseTable.GameOverFunc = func() {
		t.handleXiaoJieSuan()
		t.handleDaJieSuan()
	}
	baseTable.Consumables = t.gameRule.Consumables //t.gameRule.Consumables * t.gameRule.MaxPlayer
	baseTable.TableRule.Table = &t
	t.table = baseTable
	t.huOrder = -1
	t.firstHuSeat = qpTable.INVALID_SEAT_NUMBER
	t.table.SetMaxPlayers(t.gameRule.MaxPlayer)
	t.playPaiLogic.PlayRule = &t.gameRule
	t.playPaiLogic.HuLogic.gameRule = &t.gameRule
	t.playPaiLogic.HuLogic.laiZiPai = gameMaJiang.InvalidPai
	if t.gameRule.IsHZLaiZi == true {
		t.playPaiLogic.HuLogic.laiZiPai = gameMaJiang.Zhong
	}

	{
		paiMgr := NewXZDDPaiMgr(t.gameRule.IsHZLaiZi, localConfig.GetConfig().IsTestPai,
			t.gameRule.MaxPlayer, t.gameRule.WanFa)

		t.playPaiLogic.Table = t.table
		t.playPaiLogic.PaiMgr = paiMgr
		t.playPaiLogic.RoundOverFunc = t.RoundOver
	}
	t.playPaiLogic.BankerSeatNum = qpTable.INVALID_SEAT_NUMBER

	return &t, rspCode, ""
}
