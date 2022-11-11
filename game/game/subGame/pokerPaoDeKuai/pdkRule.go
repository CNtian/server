package pokerPDKTable

import (
	"fmt"
	pokerTable "qpGame/game/poker"
	"qpGame/localConfig"
	"qpGame/qpTable"
	"strconv"
)

type PDKRule struct {
	RuleJson         string  `json:"-"`         //
	MaxRoundCount    int32   `json:"maxCount"`  // 总局数 6\12
	MaxPlayer        int32   `json:"maxPlayer"` // 最大人数 2\3人
	Multiple         string  `json:"multiple"`  // 倍数 0.5\1\2\3\4\5\10\20\30\50
	MultipleFloat64  float64 `json:"-"`
	ShouPaiCount     int32   `json:"shouPaiCount"`     // 手牌数量 15\16
	FirstRoundChuPai int32   `json:"firstRoundChuPai"` // 首局出牌方式 幸运牌(0)\黑桃3(1)
	ZhuaNiaoMul      int32   `json:"zhuaNiaoMul"`      // 抓鸟翻倍 不抓鸟(0)红桃10(1)\幸运牌(2)

	Is3ABomb        bool `json:"_3ABomb"`       // 3A当炸弹
	Is3With1        bool `json:"_3With1"`       // 3带1
	IsShowPaiShu    bool `json:"showPaiShu"`    // 显示手牌数量
	IsShaoDaiTouPao bool `json:"shaoDaiTouPao"` // 最后 少带偷跑
	//IsShaoDaiShunPao bool `json:"shaoDaiShunPao"` // 最后 少带顺跑
	IsFanChun bool `json:"fanChun"` // 反春
	Is4With3  bool `json:"_4With3"` // 4带3
	Is4With2  bool `json:"_4With2"` // 4带2

	// 服务器自定义(与客户端一致)
	Consumables int32 `json:"consumables"` // 消耗(房卡|钻石)
}

func (this *PDKRule) CheckField() error {
	switch this.MaxRoundCount {
	case 6, 8:
		this.Consumables = 1 // 1
	case 16:
		this.Consumables = 2 // 2
	default:
		return fmt.Errorf("maxRound := %d", this.MaxRoundCount)
	}

	if this.MaxPlayer != 2 && this.MaxPlayer != 3 {
		return fmt.Errorf("maxPlayer := %d", this.MaxPlayer)
	}

	switch this.Multiple {
	case "0.5":
	case "1":
	case "2":
	case "3":
	case "4":
	case "5":
	case "10":
	case "20":
	case "30":
	case "50":
	default:
		return fmt.Errorf("multiple := %s", this.Multiple)
	}
	this.MultipleFloat64, _ = strconv.ParseFloat(this.Multiple, 64)

	if this.ShouPaiCount != 15 && this.ShouPaiCount != 16 {
		return fmt.Errorf("shouPaiCount := %d", this.ShouPaiCount)
	}

	if this.FirstRoundChuPai == 1 {
		if this.MaxPlayer == 2 {
			return fmt.Errorf("firstRoundChuPai := %d, maxPlayer := %d", this.FirstRoundChuPai, this.MaxPlayer)
		}
	} else if this.FirstRoundChuPai == 0 {

	} else {
		return fmt.Errorf("firstRoundChuPai := %d", this.FirstRoundChuPai)
	}

	switch this.ZhuaNiaoMul {
	case 0:
	case 1:
	case 2:
	default:
		return fmt.Errorf("zhuaNiaoMul := %d", this.ZhuaNiaoMul)
	}
	return nil
}

func (this *PDKRule) GetMaxPlayerCount() int32 {
	return this.MaxPlayer
}

func newPaoDeKuaiSeat(playerID qpTable.PlayerID, seatNumber qpTable.SeatNumber) (qpTable.QPSeat, error) {
	baseSeat := qpTable.NewQPSeat(playerID, seatNumber)

	pdkSeat := &PokerPDKSeat{seatData: baseSeat}
	pdkSeat.CleanRoundData()

	return pdkSeat, nil
}

func NewPaoDeKuaiTable(tableNum int32, gameRuleCfg, tableCfg string) (*PokerPDKTable, int32, string) {
	baseTable, rspCode, err := qpTable.NewQPTable(tableNum, tableCfg, newPaoDeKuaiSeat)
	if err != nil {
		return nil, rspCode, err.Error()
	}

	t := PokerPDKTable{}
	rspCode, err = t.ParseTableOptConfig(gameRuleCfg)
	if rspCode != 0 {
		return nil, rspCode, err.Error()
	}

	baseTable.GameOverFunc = func() {
		t.handleXiaoJieSuan()
		t.handleDaJieSuan()
	}
	baseTable.Consumables = t.gameRule.Consumables * t.gameRule.MaxPlayer
	baseTable.TableRule.Table = &t
	t.table = baseTable
	t.table.SetMaxPlayers(t.gameRule.MaxPlayer)

	t.PaiMgr = pokerTable.NewPokerPaiBaseMgr(localConfig.GetConfig().IsTestPai)
	t.PaiMgr.XiPai(baseTable.MaxPlayers, t.gameRule.ShouPaiCount)

	t.logic.SetRule(&t.gameRule)

	t.CleanRoundData()
	return &t, rspCode, ""
}
