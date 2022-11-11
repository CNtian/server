package clubProto

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
)

// 处理加入俱乐部邮件
const ID_HandleInviteJoinClub = 299

// 获取玩家信息
const ID_GetPlayerClubInfo = 300

type CS_GetPlayerClubInfo struct {
	PlayerID int64 `json:"uid"`
}

type PlayerClubInfo struct {
	ClubID          int32  `json:"clubID"`
	ClubCreatorName string `json:"clubCN"`
	URL             string `json:"url"`
	ClubName        string `json:"clubName"`
	Score           string `json:"score"`
	TableCount      int32  `json:"tables"`
	PlayerCount     int32  `json:"players"`
	LastTime        int64  `json:"lastTime"`
}

type SC_GetPlayerClubInfo struct {
	ClubInfo []*PlayerClubInfo `json:"clubInfo"`
}

// 创建 俱乐部桌子
const ID_ClubCreateTable = 301

type CS_ClubCreateTable struct {
	PlayerClubID int32   `json:"clubID"`     // 玩家俱乐部ID
	ClubPlayID   int64   `json:"clubPlayID"` // 俱乐部玩法ID
	IP           string  `json:"ip"`         // 玩家IP
	Longitude    float64 `json:"lng"`        // 玩家经度
	Latitude     float64 `json:"lat"`        // 玩家纬度

	// 服务之间 传递(与客户端无关)
	MZClubID        int32   `json:"mzClubID"`        // 盟主俱乐部ID
	PayUID          int64   `json:"payUID"`          // 支付人俄ID
	PlayID          int32   `json:"playID"`          // 游戏本身ID
	PlayConfig      string  `json:"playOpt"`         // 玩法 配置
	TableConfig     string  `json:"tableCfg"`        // 桌子 配置
	ClubConfig      string  `json:"clubCfg"`         // 圈子玩法 配置
	PlayerClubScore float64 `json:"playerClubScore"` // 玩家俱乐部分
	IsStop3Players  bool    `json:"isStop3"`         // 是否禁止3人局
	IsStop4Players  bool    `json:"isStop4"`         // 是否禁止4人局
	MaxTZCount      int32   `json:"maxTZC"`          // 最大同桌数

	RobotJoinReady   int32 `json:"robotR1"` //
	RobotJoinPlaying int32 `json:"robotR2"` //
	RobotInviteTimer int64 `json:"robotR3"` //
}

// 加入 俱乐部桌子
const ID_ClubJoinTable = 302

type CS_ClubJoinTable struct {
	ClubID      int32   `json:"clubID"`   // 玩家俱乐部ID
	TableNumber int32   `json:"tableNum"` // 桌子编号
	IP          string  `json:"ip"`       // 玩家IP
	Longitude   float64 `json:"lng"`      // 玩家经度
	Latitude    float64 `json:"lat"`      // 玩家纬度

	// 服务之间 传递(与客户端无关)
	PlayerClubScore float64 `json:"playerClubScore"` // 玩家俱乐部分
}

// 快速开始
const ID_QuickStart = 303

type CS_ClubQuickStart struct {
	PlayerClubID int32   `json:"clubID"`     // 玩家俱乐部ID
	ClubPlayID   int64   `json:"clubPlayID"` // 俱乐部玩法ID
	IP           string  `json:"ip"`         // 玩家IP
	Longitude    float64 `json:"lng"`        // 玩家经度
	Latitude     float64 `json:"lat"`        // 玩家纬度

	MZClubID        int32 `json:"-"`
	GameID          int32 `json:"-"`
	RobotMaxPlayers int32 `json:"rMaxPlayers"`
}

// 新建俱乐部
const ID_NewClub = 304

type CS_NewClub struct {
	Name string `json:"name"` // 俱乐部名称
}

// 新增代理
const ID_NewProxy = 305

type CS_NewProxy struct {
	UID int64 `json:"uid"`
}

// 圈主主动 拉入玩家进俱乐部  / 邀请加入俱乐部
const ID_DragIntoClub = 306

type CS_DragIntoClub struct {
	ClubID   int32 `json:"clubID"`   // 俱乐部ID
	PlayerID int64 `json:"playerID"` // 玩家ID
}

// 玩家 退出 俱乐部
const ID_ExitClub = 307

type CS_ExitClub struct {
	ClubID int32 `json:"clubID"` // 俱乐部ID
}

// 玩家申请加入 俱乐部
const ID_ApplyJoinClub = 308

type CS_ApplyJoinClub struct {
	ClubID int32 `json:"clubID"` // 俱乐部ID
}

// 新建\更新 俱乐部玩法
const ID_PutClubPlay = 310

type CS_PutClubPlay struct {
	ClubID int32 `json:"clubID"` // 俱乐部ID

	ClubPlayID   int64  `json:"clubPlayID"` // 俱乐部玩法ID
	ClubPlayName string `json:"name"`       // 俱乐部玩法名称
	GameID       int32  `json:"playID"`     // 玩法(游戏)本身ID
	PlayCfg      string `json:"playOpt" `   // 玩法(游戏)配置
	TableCfg     string `json:"tableCfg"`   // 桌子规则
	ClubCfgText  string `json:"clubRule"`   // 俱乐部规则
	IsHide       bool   `json:"isHide"`     // 是否隐藏

	ClubCfg *collClub.DBClubRule `json:"-"` // 俱乐部规则
}

// 删除 俱乐部玩法
const ID_DeleteClubPlay = 312

type CS_DeleteClubPlay struct {
	ClubID     int32 `json:"clubID"`     // 俱乐部ID
	ClubPlayID int64 `json:"clubPlayID"` // 俱乐部 玩法ID

	GameID int32 `json:"gameID"`
}

// 获取俱乐部数据
const ID_GetClubData = 313

type CS_GetClubData struct {
	ClubIDArr []int32 `json:"clubID"` // 俱乐部ID
}

// 发送给对应的游戏服 检查参数
const ID_PutClubPlay_RPC = 314

// 俱乐部成员操作
const ID_MemberOperation = 316
const (
	ClubMemberOperator_FROZEN      = 2  // 2:冻结
	ClubMemberOperator_JUDGE       = 3  // 3:裁判
	ClubMemberOperator_AUTHORITY   = 4  // 4:权限
	ClubMemberOperator_KICKOUT     = 5  // 5:踢出圈子
	ClubMemberOperator_STOP3       = 6  // 5:禁玩3人局
	ClubMemberOperator_STOP4       = 7  // 7:禁玩4人局
	ClubMemberOperator_UpgradeClub = 8  // 8:提升为圈主
	ClubMemberOperator_Remark      = 9  // 9:添加备注
	ClubMemberOperator_Robot       = 10 // 10:角色变化机器人
)

type CS_MemberOperation struct {
	ClubID     int32  `json:"clubID"` // 俱乐部ID
	PlayerID   int64  `json:"uid"`
	Action     int32  `json:"action"`
	ActionData []byte `json:"actionData"`
}

// 审查 申请加入俱乐部
const ID_CheckApplyJoinClub = 317

type CS_CheckApplyJoinClub struct {
	ClubID   int32              `json:"clubID"`
	ApplyID  primitive.ObjectID `json:"applyID"`
	ApplyUID int64              `json:"applyUID"`
	Pass     bool               `json:"pass"` // true:通过  false:拒绝
}

// 获取 俱乐部邮件
const ID_GetClubMail = 318

type CS_GetClubMail struct {
	ClubID int32 `json:"clubID"`
	Status int32 `json:"status"` // 0:所有  1:未操作
}

// 俱乐部操作
const ID_ClubOperation = 319
const (
	ClubOperator_FROZEN                = 2  // 2:冻结
	ClubOperator_BaoDi                 = 3  // 3:保底
	ClubOperator_Percentage            = 4  // 4:百分比
	ClubOperator_ManageFee             = 5  // 5:管理费
	ClubOperator_DiscardCombine        = 6  // 6:解除合并
	ClubOperator_SetNotice             = 7  // 7:修改俱乐部公告
	ClubOperator_Open                  = 8  // 8:营业\打烊
	ClubOperator_SetClubName           = 9  // 9:修改俱乐部名称
	ClubOperator_SetMemberExit         = 10 // 10:允许 玩家是否 自主退出
	ClubOperator_kickOutMember         = 11 // 11:俱乐部是否可以踢出成员
	ClubOperator_kickOutLeague         = 12 // 12:俱乐部是否可以踢出联盟
	ClubOperator_SetClubPlayPercent    = 13 // 设置俱乐部玩法百分比
	ClubOperator_PutActivity           = 14 // 活动
	ClubOperator_Stocktaking           = 15 // 盘点
	ClubOperator_SetClubBaoDiPercent   = 16 // 设置俱乐部保底百分比
	ClubOperator_SetMZNotice           = 17 // 设置盟的 公告
	ClubOperator_SetShowRankingList    = 18 // 是否显示排行榜
	ClubOperator_SetShowScoreWater     = 19 // 玩家不能看自己的流水
	ClubOperator_UpdateVirtualTableCfg = 20 // 设置盟的 虚拟桌

	ClubOperator_SetShowBaoMingFee = 21 // 是否显示报名费
	ClubOperator_SetBiLiShowWay    = 22 // 显示比例的方式  0:真点位  1：点中点
	ClubOperator_SetMaxTZCount     = 24 // 最大同桌数 0:不设置  10 20 30 50
	ClubOperator_HideClubPlay      = 25 // 隐藏玩法
	ClubOperator_PlayerGongXianWay = 26 // 玩家贡献 方式   0:所有人平均分  1:赢家分
)

type CS_ClubOperation struct {
	OperationClubID int32  `json:"operClubID"`   // 操作者俱乐部ID
	TargetClubID    int32  `json:"targetClubID"` // 目标俱乐部ID
	Action          int32  `json:"action"`
	ActionData      []byte `json:"actionData"`
}

// 获取俱乐部玩法列表
const ID_GetClubPlayList = 320

type CS_GetClubPlayList struct {
	ClubID     int32  `json:"clubID"`
	VersionNum uint64 `json:"versionNum"`
}

type SC_GetClubPlayList struct {
	VersionNum  uint64                 `json:"versionNum"`
	ClubPlayArr []*collClub.DBClubPlay `json:"clubPlay"`
	GameIDArr   []int32                `json:"gameID"`
}

// 申请合并俱乐部
const ID_ApplyMergeClub = 321

type CS_ApplyMergeClub struct {
	OperationClubID int32 `json:"clubID"`    // 操作者俱乐部
	TargetClubID    int32 `json:"tarClubID"` // 目标俱乐部
}

// 审查 申请加入俱乐部
const ID_CheckApplyMergeClub = 322

type CS_CheckApplyMergeClub struct {
	OperationClubID int32              `json:"clubID"` // 操作者俱乐部ID
	ApplyID         primitive.ObjectID `json:"applyID"`
	ApplyClubID     int32              `json:"applyClubID"`
	Pass            bool               `json:"pass"` // true:通过  false:拒绝
}

// 获取桌子快照数据
const ID_TableGet = 323

type CS_GetTable struct {
	ClubID     int32 `json:"clubID"` // 俱乐部ID
	GameID     int32 `json:"gameID"` // 玩法(游戏)ID
	ClubPlayID int64 `json:"playID"` // 俱乐部玩法ID
	//X               uint64 `json:"x"`      // 序列号
	QueryVersionNum uint64 `json:"verNum"` // 玩法版本号
	StopIndex       int    `json:"ps"`     // 分页大小
	BeginIndex      int    `json:"cp"`     // 当前页

	CurVersionNum     uint64 `json:"-"`
	ClubVersionNumber uint64 `json:"-"`
}

type TableInfo struct {
	TableNum int32   `json:"tableNum"`
	PlayID   int64   `json:"playID"`  // 俱乐部玩法ID
	Players  []int64 `json:"players"` // 玩家ID
}

type SC_GetTable struct {
	//WaitingTable int `json:"wTables"` // 等待桌子
	//PlayingTable int `json:"pTables"` // 已经在玩
	//X                 uint64   `json:"x"`           //
	//TableCount        int32    `json:"tableCount"`  // 桌子数量
	//PlayerCount       int32    `json:"playerCount"` // 玩家数量
	Tables            []string `json:"tables"` // 桌子
	TableCount        int      `json:"tableC"` // 桌子总数
	BeginIndex        int      `json:"cp"`     // 请求页
	VersionNum        uint64   `json:"verNum"` // 玩法版本号
	ClubVersionNumber uint64   `json:"CVN"`    // 俱乐部版本号
}

// 每秒获取桌子
const ID_PerSeconGetTables = 324

// 俱乐部游戏ID 列表 (请求参数\结果返回  参考 俱乐部玩法列表 320)
const ID_GetGameList = 325

// 添加\更新 互斥成员
const ID_PutMutexPlayer = 326

type CS_PutMutexPlayer struct {
	ClubID   int32              `json:"clubID"`   // 操作者俱乐部ID
	ID       primitive.ObjectID `json:"groupID"`  // 互斥组ID null:新建
	PlayerID []int64            `json:"playerID"` // 被操作玩家ID
}

// 删除 互斥成员
const ID_DeleteMutexPlayerGroup = 327

type CS_DeleteMutexPlayerGroup struct {
	ClubID int32              `json:"clubID"`
	ID     primitive.ObjectID `json:"groupID"` // 互斥组ID
}

// 获取 互斥成员
const ID_GetMutexPlayer = 328

type CS_GetMutexPlayer struct {
	ClubID int32 `json:"clubID"`
}
type SC_GetMutexPlayer struct {
	Arr []collClub.DBMemberMutexGroup `json:"list"`
}

// 俱乐部 分 变化
const ID_CurScoreChanged = 329

type SC_CurScoreChanged struct {
	ClubID   int32  `json:"clubID"`
	Score    string `json:"score"`
	MZClubID int32  `json:"mzID"`
}

// 通知 玩家加入新俱乐部
const ID_NoticePlayerJoinClub = 330

type SC_NoticePlayerJoinClub struct {
	ClubID int32 `json:"clubID"`
}

// 通知  玩家离开俱乐部
const ID_NoticePlayerExitClub = 331

type SC_NoticePlayerExitClub struct {
	ClubID int32 `json:"clubID"`
}

// 获取俱乐部简介
const ID_GetClubIntro = 332

type CS_GetClubIntro struct {
	ClubID int32 `json:"clubID"`
}

type SC_GetClubIntro struct {
	Notice string `json:"notice"`
	Name   string `json:"name"`
	CVN    uint64 `json:"cvn"`
}

// 获取 成员裁判日志
const ID_GetMemberJudgeLog = 334

type CS_GetMemberJudgeLog struct {
	OperationClubID int32 `json:"operClubID"` // 操作者俱乐部ID
	ClubID          int32 `json:"clubID"`     // 目标俱乐部ID
	UID             int64 `json:"uid"`        // 目标玩家ID
	Date            int   `json:"date"`

	Category int `json:"category"` //

	PageSize int `json:"pageSize"`
	CurPage  int `json:"curPage"`
}

// 强制解散
const ID_ForceDissolveTable = 335

type CS_ForceDissolveTable struct {
	OperationClubID int32 `json:"operClubID"` // 操作者俱乐部ID
	TableID         int32 `json:"tableID"`
}

// 发起退出联盟
const ID_ApplyExitLeague = 336

type CS_ApplyLeaveLeague struct {
	OperationClubID int32 `json:"operClubID"` // 操作者俱乐部ID
}

// 审核 退出 联盟
const ID_CheckExitLeague = 337

type CS_CheckExitLeague struct {
	OperationClubID int32              `json:"clubID"` // 操作者俱乐部ID
	ApplyID         primitive.ObjectID `json:"applyID"`
	//ApplyClubID     int32              `json:"applyClubID"`
	Pass bool `json:"pass"` // true:通过  false:拒绝
}

// 俱乐部有新邮件
const ID_NewClubMail = 338

type CS_NewClubMail struct {
	ClubID int32 `json:"clubID"` // 俱乐部ID
}

// 查询玩家所在联盟
const ID_QueryPlayerLeague = 339

type CS_QueryPlayerLeague struct {
	OperationClubID int32 `json:"operClubID"` // 操作者俱乐部ID

	PlayerClubID int32 `json:"-"`
	PlayerID     int64 `json:"uid"`
}
type QueryPlayerLeagueItem struct {
	UID      int64  `json:"uid"`
	HeadURL  string `json:"headURL"`
	Name     string `json:"playerN"`
	ClubName string `json:"clubN"`
	ClubID   int32  `json:"clubID"`
}

// 审核退出俱乐部
const ID_CheckExitClub = 341

type CS_CheckExitClub struct {
	ClubID    int32              `json:"clubID"`
	ApplyID   primitive.ObjectID `json:"applyID"` // 邮件ID
	ApplyerID int64              `json:"applyer"` // 申请人的玩家ID
	Pass      bool               `json:"pass"`
}

// 虚拟桌子配置
const ID_GetVirtualTableConfigItem = 342

//type CS_PlayerSceneChanged struct {
//	ClubID int32 `json:"clubID"`
//}

// 获取盟里面空闲玩家
const ID_GetMzOnlineMember = 343

// 邀请
const ID_InviteMzOnlineMember = 345

type CS_InviteMzOnlineMember struct {
	TableID  int32   `json:"tableID"`  // 被邀请 进入的桌子
	PlayName string  `json:"playName"` // 玩法名称
	Nick     string  `json:"nick"`     // 邀请人昵称
	PlayerID []int64 `json:"playerID"` // 被邀请人
}

// 通知邀请加入桌子
const ID_NoticeInviteJoinTable = 345

//type CS_NoticeInviteJoinTable struct {
//	PlayName string `json:"playName"` // 玩法名称
//	Nick     string `json:"nick"`     // 邀请人昵称
//	PlayerID int64  `json:"playerID"`
//	TableID  int32  `json:"tableID"` // 被邀请 进入的桌子
//}

// 清零
const ID_SetClubScore0 = 346

type CS_SetClubScore0 struct {
	OperationClubID int32 `json:"operClubID"` // 操作者俱乐部ID

	//TargetClubID int32 `json:"clubID"`

	Opt int32 `json:"opt"` // 1:成员  2:圈主(队长)
}

// 获取 俱乐部的备注
const ID_GetClubMemberRemark = 347

type CS_ID_GetClubMemberRemark struct {
	ClubID int32 `json:"clubID"`
}

type ClubMemberRemark struct {
	UID  int64  `json:"uid"`
	Name string `json:"name"`
}
type SC_ID_GetClubMemberRemark struct {
	ClubID int32 `json:"clubID"`

	Data []ClubMemberRemark `json:"data"`
}

// 设置圈子等级
const ID_UpdateClubLevel = 348

type CS_UpdateClubLevel struct {
	ClubID int32 `json:"mzID"`
}

// 设置圈子状态
const ID_UpdateClubStatus = 349

type CS_UpdateClubStatus struct {
	ClubID int32 `json:"mzID"`
	IsOpen bool  `json:"isOpen"`
}

// 查询 统计 消息ID 起始位置
const ID_Query_Total_Start = 350
const ID_Query_Total_Start_MAX = 370

// 获取活动
const ID_GetActivity = 371

type CS_GetActivity struct {
	ClubID int32 `json:"clubID"`
}

// 获取活动排序
const ID_GetActivitySort = 372

type CS_GetActivitySort struct {
	ClubID   int32 `json:"clubID"`
	CateGory int32 `json:"category"`

	CurPage  int `json:"curPage"`
	PageSize int `json:"pageSize"`
}

type SC_GetActivitySort struct {
	AcStartTime int64                    `json:"acST"`
	ItemCount   int                      `json:"count"`
	Item        interface{}              `json:"item"`
	Self        int                      `json:"self"`
	Value       interface{}              `json:"value"`
	LastAcRule  *collClub.DBClubActivity `json:"lAct"`
}

// 获取活动奖励
const ID_GetActivityAward = 373

type CS_GetActivityAward struct {
	ClubID   int32 `json:"clubID"`
	Category int32 `json:"category"` // 1:局数  2:分数
}

// 获取活动奖励列表
const ID_GetActivityAwardList = 374

type CS_GetActivityAwardList struct {
	ClubID int32 `json:"clubID"`
}

// 活动黑名单
const ID_ActivityBlackList = 375

type CS_ActivityBlackList struct {
	ClubID int32  `json:"clubID"`
	SubID  int    `json:"subID"` // 1:新增  2:删除  3:获取
	Param  []byte `json:"param"`
}
type BlackListUpdateOne struct {
	UID int64 `json:"uid"`
}

// 获取2人一起玩的数据
const ID_GetTwoPlayerTogetherData = 376

type CS_GetTwoPlayerTogetherData struct {
	MZClubID int32   `json:"clubID"`
	Date     int32   `json:"date"`
	PlayerID []int64 `json:"playerID"` // 指定玩家搜索
}

// 获取俱乐部玩法
const ID_GetClubPlayInfo = 377

type CS_GetClubPlay struct {
	ClubID     int32 `json:"clubID"`
	ClubPlayID int64 `json:"clubPID"`
}

type SC_GetClubPlay struct {
	VersionNum  uint64                 `json:"versionNum"`
	ClubPlayArr []*collClub.DBClubPlay `json:"clubPlay"`
	GameIDArr   []int32                `json:"gameID"`
}

const ID_TableSnapshoot_Start = 400

const ID_MaxValue = 500

// 获取所有盟
const ID_GetProxyClubID = 3490

type SC_GetProxyClubID struct {
	ClubID   int32  `json:"clubID"`
	ClubName string `json:"clubName"`
}

// 获取代理
const ID_GetProxyList = 3491

type CS_GetProxy struct {
	Date     int `json:"date"`
	CurPage  int `json:"curPage"`  // 当前页
	PageSize int `json:"pageSize"` // 页大小
}

type SC_GetProxy struct {
	//Now     int64                                     `json:"now"`
	MZCount int                                        `json:"count"` // 俱乐部总数
	Data    []*dbCollectionDefine.DBDailyMengZHuPlayer `json:"item"`  // 详细信息
}

// 获取代理
const ID_GetProxyReportList = 3492

type CS_GetProxyReportList struct {
	ClubID int32 `json:"clubID"`
}

type SC_GetProxyReportList struct {
	Data []dbCollectionDefine.DBDailyMengZHuPlayer `json:"item"` // 详细信息
}

// 赠送房卡
const ID_GiveRoomCard = 3493

type CS_GiveRoomCard struct {
	ToClubID int32 `json:"toCID"`
	Value    int32 `json:"value"`
}

// 赠送房卡 列表
const ID_GiveRoomCardList = 3494

type CS_GiveRoomCardList struct {
	Item []collClub.DBRoomCardDealLog `json:"item"`
}

// 通知创建盟主俱乐部
const ID_CreateMengZhuClub = 3495

// 取消代理
const ID_CancelProxy = 3496

type CS_CancelProxy struct {
	ClubID []int32 `json:"clubID"`
}

// 邀请成为代理
const ID_InviteToProxy = 3497

type CS_InviteToProxy struct {
	UID int64 `json:"uid"`
}

// 处理 --- 邀请成为代理
const ID_HandleInviteToProxy = 3498

type CS_HandleInviteToProxy struct {
	EmailID primitive.ObjectID `json:"id"`
	Action  int                `json:"action"` // 1:同意  0:拒绝  2:删除
}

// 通知玩家新邮件
const ID_NoticePlayerNewEmail = 3499

// 通知重新获取俱乐部
const ID_NoticeReGetClub = 3500

// 通知总分校正完毕
const ID_NoticeClubStockingFinish = 3501
