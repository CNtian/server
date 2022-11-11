package qpTable

// 俱乐部规则(必须和qp-app服的俱乐部数据 保持一致)
type DBClubRule struct {
	MinEntryScoreText string  `json:"entryScore" bson:"-"`  // 最小进入分
	MinEntryScoreInt  float64 `json:"-" bson:"entry_score"` // 最小进入分
	// 结束条件 0:整局负分 1:小局负分  2:低于多少分结束 3:低于多少分观看
	GameOverCon   int32   `json:"gameOverCon" bson:"over_con"`
	OverScoreText string  `json:"overScore" bson:"-"`  // 2:低于多少分结束
	OverScoreInt  float64 `json:"-" bson:"over_score"` // 2:低于多少分结束

	GongXianMode int32 `json:"GXMode" bson:"gx_mode"` // 0:比例(赢家)  1:固定方式

	MaxWinnerText   string  `json:"maxWinner" bson:"-"`    // 1:固定方式 (大赢家)
	OtherPlayerText string  `json:"otherPlayer" bson:"-"`  // 1:固定方式 (其他玩家)
	MaxWinner       float64 `json:"-" bson:"max_Winner"`   // 1:固定方式 (大赢家)
	OtherPlayer     float64 `json:"-" bson:"other_player"` // 1:固定方式 (其他玩家)

	AllWinnerText string  `json:"winnerRatio" bson:"-"`  // 0:比例 所有赢家的百分比
	AllWinner     float64 `json:"-" bson:"winner_ratio"` // 0:比例 所有赢家的百分比

	// 牛牛
	//ZhuangMinScore        string  `json:"zhuangMinScore"` // 当庄最低分数
	//ZhuangMinScoreFloat64 float64 `json:"-"`              // 当庄最低分数
}
