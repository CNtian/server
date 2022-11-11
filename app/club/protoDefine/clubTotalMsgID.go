package clubProto

import (
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
)

// 查询 成员列表
const ID_GetClubMember = 350

type CS_GetClubMember struct {
	PlayerClubID int32 `json:"clubID"` // 操作人俱乐部ID
	Date         int   `json:"date"`   // 指定查询 日期 示例:20220219 当天:0

	QueryMemberUID int64 `json:"qMemberUID"` // 查询 指定成员

	QueryClubID int32 `json:"qClubID"` // 查询 指定圈子
}

type GetClubMemberData struct {
	PlayerID        int64  `json:"playerID"`
	PlayerHead      string `json:"playerHead"`
	PlayerNick      string `json:"playerNick"`
	JoinTime        int64  `json:"joinTime"`
	Score           string `json:"score"`
	ClubID          int32  `json:"clubID"`          // 圈子ID
	ClubName        string `json:"clubName"`        // 圈子名称
	ClubCreatorID   int64  `json:"clubCID"`         // 圈子创建者ID
	ClubCreatorName string `json:"clubCreatorName"` // 圈子创建者名称
	Role            int32  `json:"role"`            // 0:成员  1:管理员  2:圈主
	Status          int32  `json:"status"`          // 0:正常  1:冻结
	IsStop3         bool   `json:"isStop3"`         // 是否 禁止 玩3人局
	IsStop4         bool   `json:"isStop4"`         // 是否 禁止 玩4人局
	Remark          string `json:"remark"`          //

	GameCount     int32  `json:"gameCount" bson:"game_count"` // 对局数
	GameScoreText string `json:"gameScore" bson:"-"`          // 对局战绩
	HaoKaText     string `json:"haoKa" bson:"-"`              // 消耗数
	XiaoHaoScore  string `json:"gongXian" bson:"-"`           // 报名费
}

type SC_GetClubMember struct {
	CurDate           int64                `json:"curDate"`         // 当前时间戳
	MemberArr         []*GetClubMemberData `json:"members"`         // 成员信息
	MyClubMemberCount int                  `json:"selfMemberCount"` // 自己的俱乐部成员数量
	AllMemberCount    int                  `json:"allMemberCount"`  // 自己以下所有成员数量
}

// 查询 圈子列表
const ID_GetClubList = 351

type CS_GetClubList struct {
	PlayerClubID       int32 `json:"clubID"`         // 操作人俱乐部ID
	QueryClubID        int32 `json:"qClubID"`        // 被查询的圈子ID
	QuerySubordinateID int32 `json:"qSubordinateID"` // 查询该圈的所有下级圈子
}

type GetClubListData struct {
	ClubID          int32  `json:"clubID"`          // 圈子ID
	ClubName        string `json:"clubName"`        // 圈子名称
	ClubCreatorID   int64  `json:"clubCID"`         // 圈子创建者ID
	ClubCreatorName string `json:"clubCreatorName"` // 圈子创建者名称

	SuperiorClubID          int32  `json:"sClubID"`    // 上级圈子ID
	SuperiorClubName        string `json:"sClubName"`  // 上级圈子名称
	SuperiorClubCreatorID   int64  `json:"sClubCID"`   // 上级圈子创建者ID
	SuperiorClubCreatorName string `json:"sClubCName"` // 上级圈子创建者名称

	IsOpen           bool   `json:"open"`         // 状态 0:正常 1:打烊
	IsFrozen         bool   `json:"frozen"`       // 状态 0:正常 1:冻结
	Percent          int32  `json:"percent"`      // 百分比
	ManageFee        string `json:"manageFee"`    // 管理费
	ScoreCount       string `json:"scoreCount"`   // 总分
	BaoDi            string `json:"baoDi"`        // 保底
	Unusable         string `json:"unusable"`     // 不可用
	ClubCreatorScore string `json:"creatorScore"` // 圈主总分
}

type SC_GetClubList struct {
	ClubArr           []*GetClubListData `json:"clubData"`      // 信息
	AllSubClubCount   int                `json:"allSubCount"`   // 所有子圈数量
	DirectlyClubCount int                `json:"directlyCount"` // 直属子圈数量
}

// 俱乐部积分记录
const ID_GetClubScoreLog = 352

type CS_GetClubScoreLog struct {
	ClubID  int32   `json:"clubID"`  // 俱乐部
	LogType []int32 `json:"logType"` // 0:所有  1：管理费 2：游戏  3：消耗   4：裁判  5：奖励
	Data    int     `json:"date"`

	PageSize int `json:"pageSize"`
	CurPage  int `json:"curPage"`
}

type SC_GetClubScoreLog struct {
	LogArr []*collClub.DBClubScoreLog `json:"logs"`
}

// 俱乐部积分记录
const ID_GetClubOperationLog = 353

type CS_GetClubOperationLog struct {
	OperClubID int32 `json:"operClubID"` // 操作人 所属俱乐部ID
	ClubID     int32 `json:"clubID"`     // 俱乐部
}
type SC_GetClubOperationLog struct {
	LogArr []*collClub.DBClubOperationLog `json:"logs"`
}

// 圈子 战绩统计记录
const ID_GetClubGameRecord = 354

type CS_GetClubGameRecord struct {
	OperClubID int32 `json:"operClubID"` // 操作人 所属俱乐部ID
	//ClubID      int32 `json:"clubID"`      // 俱乐部
	PlayerID    int64 `json:"playerID"`    // 指定玩家ID
	Date        int32 `json:"date"`        // 指定日期 例如: 20220101
	QClubID     int32 `json:"qClubID"`     // 指定俱乐部
	QClubPlayID int64 `json:"qClubPlayID"` // 指定俱乐部玩法ID
	QTableID    int32 `json:"qTableID"`    // 指定桌子ID

	PageSize int `json:"pageSize"` // 页大小
	CurPage  int `json:"curPage"`  // 当前页
}
type SC_GetClubGameRecord struct {
	CurTime int64                                  `json:"curTime"`
	Arr     []*dbCollectionDefine.DBGameOverRecord `json:"records"`
}

// 玩家统计
const ID_GetClubPlayerTotal = 355

type CS_GetClubPlayerTotal struct {
	OperClubID int32 `json:"operClubID"` // 操作人 所属俱乐部ID
	ClubID     int32 `json:"clubID"`     // 俱乐部
	Date       int   `json:"date"`       // 指定日期
	PlayerID   int64 `json:"playerID"`   // 指定玩家ID
}

type SC_GetClubPlayerTotal struct {
	//Total   db.GetClubPlayerTotal `json:"total"`
	CurTime int64 `json:"curTime"`

	TotalZengSong      string `json:"zsS"`   // 赠送统计
	TotalGameScore     string `json:"gameS"` // 游戏统计
	TotalGongXianScore string `json:"gxS"`   // 贡献统计
	TotalJiangLiScore  string `json:"jlS"`   // 收益(奖励)统计
	TotalBaodi         string `json:"baodi"` // 保底
}

// 圈子 统计记录
const ID_GetClubTotal = 356

type CS_GetClubTotal struct {
	OperClubID int32 `json:"operClubID"` // 操作人 所属俱乐部ID
	Date       int   `json:"qDate"`      // 指定日期 示例:20210901

	QClubID   int32 `json:"qClubID"` // 俱乐部
	QPlayerID int64 `json:"qPID"`    // 队长ID
}

type RspClubTotalItem struct {
	PlayerID   int64  `json:"playerID"`
	PlayerHead string `json:"playerHead"`
	PlayerNick string `json:"playerNick"`
	JoinTime   int64  `json:"joinTime"`
	Score      string `json:"score"`

	ClubID          int32  `json:"clubID"`        // 圈子ID
	ClubName        string `json:"clubName"`      // 圈子名称
	BiLi            int32  `json:"biLi"`          // 比例
	JingJie         string `json:"jingJie"`       // 警戒
	ZongFen         string `json:"zongFen"`       // 总分
	FuFen           string `json:"fuFen"`         // 负分
	IsOpen          bool   `json:"open"`          // 状态 0:正常 1:打烊
	IsFrozen        bool   `json:"frozen"`        // 状态 0:正常 1:冻结
	IsKickOutMember bool   `json:"kickOutMember"` // 俱乐部是否可以踢出成员
	IsKickOutLeague bool   `json:"kickOutLeague"` // 俱乐部是否可以踢出
	Remark          string `json:"remark"`        //

	GameCount     int32  `json:"gameCount" ` // 对局数
	GameScoreText string `json:"gameScore"`  // 对局战绩
	HaoKaText     string `json:"haoKa"`      // 消耗数
	XiaoHaoScore  string `json:"gongXian"`   // 报名费
	JiangLi       string `json:"jian_li"`    // 奖励
}

type SC_GetClubTotal struct {
	Total              []*RspClubTotalItem `json:"total"`
	CurDate            int                 `json:"curDate"`
	AllSubClubCount    int                 `json:"allSubClubCount"`
	AllDirSubClubCount int                 `json:"allDirSubClubCount"`
}

// 俱乐部玩法比例
const ID_GetClubPlayPercent = 358

type CS_GetClubPlayPercent struct {
	OperClubID   int32 `json:"operClubID"`   // 操作人 所属俱乐部ID
	TargetClubID int32 `json:"targetClubID"` // 俱乐部ID
}

// 查询统计
const ID_QueryTotal = 359

type CS_QueryTotal struct {
	SubID int    `json:"subID"`
	Data  string `json:"param"`
}

// 获取机器人俱乐部配置
const ID_GetRobotCfg = 360

type CS_GetRobotCfg struct {
	ClubID int32 `json:"clubID"` // 圈子ID
}

// 获取俱乐部所有机器人配置
const ID_GetRobotItemCfg = 361

type CS_GetRobotItemCfg struct {
	ClubID   int32 `json:"clubID"`   // 圈子ID
	PageSize int64 `json:"pageSize"` // 页大小
	CurPage  int64 `json:"curPage"`  // 当前页

	TargetUID int64 `json:"uid"` // 查询时使用
	Date      int   `json:"date"`
}

// robot更新了
const ID_UpdateRobotCfg = 362

// Max = 370
