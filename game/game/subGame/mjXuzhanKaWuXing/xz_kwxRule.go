package mj_XueZhan_KWXTable

import (
	"fmt"
	"qpGame/game/gameMaJiang"
	"qpGame/localConfig"
	"qpGame/qpTable"
	"strconv"
)

type KWXPlayRule struct {
	RuleJson        string  `json:"-"`         //
	MaxRoundCount   int32   `json:"maxRound"`  // 总局数 8\16
	MaxPlayer       int32   `json:"maxPlayer"` // 最大人数 2\3
	Multiple        string  `json:"multiple"`  // 倍数 0.5\1\2\3\5\10\20\30\50
	MultipleFloat64 float64 `json:"-"`
	//XuanPiao        int64   `json:"xuanPiao"` // 选漂 0(不漂)\100(每局漂)\1(固定漂)\2(固定漂)\3(固定漂)\5(固定漂)
	FengDingFanShu int64 `json:"FDFanShu"` // 封顶番数 8\16
	//MaiMa           int32   `json:"maiMa"`    // 买码 0(不买码)\1(亮牌自摸买码)\2(自摸买码)
	//MaShu int32 `json:"maShu"` // 码数 0(不买码)\1(买1码)\2(买1赠1)\3(进五进十)

	IsBuFenLiang     bool `json:"buFenLiang"`     // 部分亮
	IsQuanPinDao     bool `json:"quanPinDao"`     // 全频道
	IsQuBaJiu        bool `json:"quBaJiu"`        // 去八九
	IsQuFeng         bool `json:"quFeng"`         // 去风(中发白)
	IsPengPengHux4   bool `json:"pengPengx4"`     // 碰碰胡x4
	IsKaWuXingx4     bool `json:"kaWuXingx4"`     // 卡五星x4
	IsGangShangHuax4 bool `json:"gangShangHuax4"` // 杠上花x4

	//IsPaoQiaMoBa bool `json:"paoQiaMoBa"` // 跑恰摸八

	// 服务器自定义(与客户端一致)
	Consumables int32 `json:"consumables"` // 消耗(房卡|钻石)
}

func (this *KWXPlayRule) GetMaxPlayerCount() int32 {
	return this.MaxPlayer
}

func (this *KWXPlayRule) CheckField() error {
	if this.MaxPlayer < 2 || this.MaxPlayer > 3 {
		return fmt.Errorf("maxPlayer := %d", this.MaxPlayer)
	}
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
	case "0.5":
	case "1":
	case "2":
	case "3":
	case "5":
	case "10":
	case "20":
	case "30":
	case "50":
	default:
		return fmt.Errorf("multiple := %s", this.Multiple)
	}
	this.MultipleFloat64, _ = strconv.ParseFloat(this.Multiple, 64)

	//switch this.XuanPiao {
	//case 0, 1, 2, 3, 5, 100:
	//default:
	//	return fmt.Errorf("xuanPiao := %d", this.XuanPiao)
	//}
	if this.FengDingFanShu != 8 && this.FengDingFanShu != 16 {
		return fmt.Errorf("FDFanShu := %d", this.FengDingFanShu)
	}

	//if this.MaiMa < 0 || this.MaiMa > 2 {
	//	return fmt.Errorf("maiMa := %d", this.MaiMa)
	//}
	//
	//if this.MaShu < 0 || this.MaShu > 3 {
	//	return fmt.Errorf("maShu := %d", this.MaShu)
	//}

	if this.IsQuBaJiu == true && this.MaxPlayer == 3 {
		return fmt.Errorf("quBaJiu := %v", this.IsQuBaJiu)
	}
	if this.IsQuFeng == true && this.MaxPlayer == 3 {
		return fmt.Errorf("quFeng := %v", this.IsQuFeng)
	}

	return nil
}

func newMJKWXSeat(playerID qpTable.PlayerID, seatNumber qpTable.SeatNumber) (qpTable.QPSeat, error) {
	baseSeat := qpTable.NewQPSeat(playerID, seatNumber)
	mjSeat := &gameMaJiang.MJSeat{SeatData: baseSeat}

	kwxSeat := &KWXSeat{MJSeat: mjSeat}
	kwxSeat.CleanRoundData()

	return kwxSeat, nil
}

func NewGameXZKWXTable(tableNum int32, gameRuleCfg, tableCfg string) (*GameKWXTable, int32, string) {
	baseTable, rspCode, err := qpTable.NewQPTable(tableNum, tableCfg, newMJKWXSeat)
	if err != nil {
		return nil, rspCode, err.Error()
	}

	t := GameKWXTable{}
	rspCode, err = t.ParseTableOptConfig(gameRuleCfg)
	if rspCode != 0 {
		return nil, rspCode, err.Error()
	}

	baseTable.GameOverFunc = func() {
		t.handleXiaoJieSuan()
		t.handleDaJieSuan()
	}
	baseTable.Consumables = t.gameRule.Consumables
	baseTable.TableRule.Table = &t
	t.table = baseTable
	t.table.SetMaxPlayers(t.gameRule.MaxPlayer)
	t.playPaiLogic.PlayRule = &t.gameRule
	t.playPaiLogic.HuLogic.gameRule = &t.gameRule

	{
		paiMgr := NewKWXPaiMgr(
			!(t.gameRule.IsQuFeng),
			!(t.gameRule.IsQuBaJiu),
			localConfig.GetConfig().IsTestPai,
		)

		t.playPaiLogic.Table = t.table
		t.playPaiLogic.PaiMgr = paiMgr
		t.playPaiLogic.RoundOverFunc = t.RoundOver
	}
	t.playPaiLogic.BankerSeatNum = qpTable.INVALID_SEAT_NUMBER
	t.playPaiLogic.CleanRoundData()

	return &t, rspCode, ""
}
