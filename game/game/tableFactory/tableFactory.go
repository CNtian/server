package tableFactory

import (
	"qpGame/commonDefine/mateProto"
	"qpGame/game/subGame/NiuNiu_mpQiangZhuang"
	"qpGame/game/subGame/NiuNiu_wzTongBi"
	"qpGame/game/subGame/ZhaJinHua"
	mjShiYanKWXTable "qpGame/game/subGame/mjShiYanKaWuXing"
	mjSuiZhouKWXTable "qpGame/game/subGame/mjSuiZhouKaWuXing"
	mjXianYouTable "qpGame/game/subGame/mjXianYou"
	mjXYKWXTable "qpGame/game/subGame/mjXiangYangKaWuXing"
	mjXiaoGanKWXTable "qpGame/game/subGame/mjXiaoGanKaWuXing"
	mjXZDDTable "qpGame/game/subGame/mjXueZhanDaoDi"
	mj_XueZhan_KWXTable "qpGame/game/subGame/mjXuzhanKaWuXing"
	mjYiChenKWXTable "qpGame/game/subGame/mjYiChenKaWuXing"
	pokerDouFourteen "qpGame/game/subGame/pokerDou14"
	"qpGame/game/subGame/pokerDouDiZhu"
	"qpGame/game/subGame/pokerPaoDeKuai"
	pokerYaAnD14 "qpGame/game/subGame/pokerYaAnDou14"
	"qpGame/qpTable"
)

const (
	Play_MJ_KaWuXing         = 101 // 襄阳卡五星
	Play_PK_PaoDeKuai        = 102 // 跑得快
	Play_PK_ZhaJinHua        = 103 // 诈金花
	Play_PK_NiuNiu_wztb      = 104 // 牛牛无庄通比
	Play_MJ_SuiZhouKaWuXing  = 105 // 随州卡五星
	Play_MJ_XiaoGanKaWuXing  = 106 // 孝感卡五星
	Play_PK_NiuNiu_mpqz      = 107 // 牛牛明牌抢庄
	Play_MJ_ShiYanKaWuXing   = 108 // 十堰卡五星
	Play_MJ_YiChenKaWuXing   = 109 // 宜城卡五星
	Play_MJ_XueZhanDaoDi     = 110 // 血战到底
	Play_MJ_TianTianKaWuXing = 111 // 天天卡五星
	Play_MJ_XueZhanKaWuXing  = 112 // 血战卡五星
	Play_PK_Dou14            = 114 // 斗14
	Play_MJ_XianYou          = 116 // 仙游麻将
	Play_Pk_DouDiZhu         = 117 // 斗地主
	Play_Pk_YaAnD14          = 118 // 雅安斗14
)

var supportPlayID = map[int32]string{
	Play_MJ_KaWuXing:         "襄阳卡五星",
	Play_PK_PaoDeKuai:        "跑得快",
	Play_PK_ZhaJinHua:        "诈金花",
	Play_PK_NiuNiu_mpqz:      "牛牛明牌抢庄",
	Play_PK_NiuNiu_wztb:      "牛牛无庄通比",
	Play_MJ_SuiZhouKaWuXing:  "随州卡五星",
	Play_MJ_XiaoGanKaWuXing:  "孝感卡五星",
	Play_MJ_ShiYanKaWuXing:   "十堰卡五星",
	Play_MJ_YiChenKaWuXing:   "宜城卡五星",
	Play_MJ_XueZhanDaoDi:     "血战到底",
	Play_MJ_TianTianKaWuXing: "天天卡五星",
	Play_MJ_XueZhanKaWuXing:  "血战卡五星",
	Play_PK_Dou14:            "斗14",
	Play_MJ_XianYou:          "仙游麻将",
	Play_Pk_DouDiZhu:         "斗地主",
	Play_Pk_YaAnD14:          "雅安斗14",
}

func IsSupport(playID int32) bool {
	if _, ok := supportPlayID[playID]; ok == false {
		return false
	}
	return true
}

func GetPlayName(playID int32) string {
	if v, ok := supportPlayID[playID]; ok == true {
		return v
	}
	return ""
}

func GetSupport() map[int32]string {
	return supportPlayID
}

// rule[0]=playCfg, rule[1]=tableCfg
func NewGameTable(tableNum, gameID int32, rule ...string) (qpTable.QPGameTable, int32, string) {
	playCfg := rule[0]
	tableCfg := rule[1]

	var (
		newTable         qpTable.QPGameTable
		rspCode          int32
		rspDesc, clubCfg string
	)
	if len(rule) > 2 {
		clubCfg = rule[2]
	}

	defer func() {
		if newTable != nil && rspCode == 0 {
			newTable.GetBaseQPTable().GameID = gameID
		}
	}()

	switch gameID {
	case Play_MJ_KaWuXing, Play_MJ_TianTianKaWuXing:
		newTable, rspCode, rspDesc = mjXYKWXTable.NewGameKWXTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_PK_PaoDeKuai:
		newTable, rspCode, rspDesc = pokerPDKTable.NewPaoDeKuaiTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_MJ_SuiZhouKaWuXing:
		newTable, rspCode, rspDesc = mjSuiZhouKWXTable.NewGameKWXTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_MJ_XiaoGanKaWuXing:
		newTable, rspCode, rspDesc = mjXiaoGanKWXTable.NewGameKWXTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_PK_NiuNiu_wztb:
		newTable, rspCode, rspDesc = NiuNiu_wzTongBi.NewNiuNiuWuZhuangTongBiTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_PK_ZhaJinHua:
		newTable, rspCode, rspDesc = ZhaJinHua.NewZhaJinHuaTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_PK_NiuNiu_mpqz:
		newTable, rspCode, rspDesc = NiuNiu_mpQiangZhuang.NewNiuNiuMingPaiQiangZhuangTable(tableNum, playCfg, tableCfg, clubCfg)
		return newTable, rspCode, rspDesc
	case Play_MJ_ShiYanKaWuXing:
		newTable, rspCode, rspDesc = mjShiYanKWXTable.NewGameKWXTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_MJ_YiChenKaWuXing:
		newTable, rspCode, rspDesc = mjYiChenKWXTable.NewGameKWXTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_MJ_XueZhanDaoDi:
		newTable, rspCode, rspDesc = mjXZDDTable.NewGameXZDDTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_MJ_XueZhanKaWuXing:
		newTable, rspCode, rspDesc = mj_XueZhan_KWXTable.NewGameXZKWXTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_PK_Dou14:
		newTable, rspCode, rspDesc = pokerDouFourteen.NewDou14Table(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_MJ_XianYou:
		newTable, rspCode, rspDesc = mjXianYouTable.NewGameXianYouTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_Pk_DouDiZhu:
		newTable, rspCode, rspDesc = pokerDouDiZhu.NewDouDiZhuTable(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	case Play_Pk_YaAnD14:
		newTable, rspCode, rspDesc = pokerYaAnD14.NewDou14Table(tableNum, playCfg, tableCfg)
		return newTable, rspCode, rspDesc
	default:

	}

	return nil, mateProto.Err_NotMatchPlayID, ""
}
