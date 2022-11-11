package mateProto

const (
	Err_Failed           = -100 // 操作失败
	ErrRoomCardNotEnough = -309 // 房卡不够
	ErrClubStop3Player   = -328 // 禁止3人局
	ErrDiamondNotEnough  = -329 // 钻石不够
)

const (
	Err_Success int32 = 0

	Err_TableFull            int32 = -1000 // 桌子已满
	Err_NewSeatFailed        int32 = -1001 // 创建新的座位失败
	Err_SeatFull             int32 = -1002 // 座位已有人
	Err_InvalidSeatNumber    int32 = -1003 // 座位号无效
	Err_NotFindIdleSeat      int32 = -1004 // 未发现空座位
	Err_GameStarted          int32 = -1005 // 游戏已经开始
	Err_NotFindPlayer        int32 = -1006 // 玩家不存在
	Err_TableStatusNotMatch  int32 = -1007 // 桌子状态不匹配
	Err_PlayerNotEnough      int32 = -1008 // 玩家人数不够
	Err_NotFindIdleTabNumber int32 = -1009 // 没有空闲桌子号
	Err_CreateTableParam     int32 = -1010 // 创建桌子参数错误
	Err_AlreadyPlaying       int32 = -1011 // 已经在桌子中
	Err_ActionNotMatchStatus int32 = -1012 // 此状态不准操作
	Err_NotMatchMsgID        int32 = -1013 // 协议号不存在
	Err_NotFindTable         int32 = -1014 // 未发现对应的桌子
	Err_ServiceStatus        int32 = -1015 // 服务器状态不允许
	Err_NotMatchTableRule    int32 = -1016 // 违背了桌子规则
	Err_TuoGuanLimit         int32 = -1017 // 托管限制
	Err_CustomPai            int32 = -1018 // 牌有错误
	Err_FindIPRepeat         int32 = -1019 // IP重复
	Err_FindGPSFail          int32 = -1020 // GPS过近
	Err_ClubRule             int32 = -1021 // 俱乐部规则错误
	Err_ClubRuleLimit        int32 = -1028 // 俱乐部条件结束

	Err_OperationIDErr    int32 = -1029 // 操作码错误
	Err_ProtocolDataErr   int32 = -1030 // 协议解析错误
	Err_OperationNotExist int32 = -1031 // 操作不存在
	Err_PaiNotExist       int32 = -1032 // 牌不存在
	Err_OperationParamErr int32 = -1033 // 操作参数错误
	Err_NotMatchPlayID    int32 = -1034 // 未找到玩法
	Err_TableNumberExist  int32 = -1035 // 桌子编号已存在
	Err_SystemError       int32 = -1036 // 系统错误
	Err_NotYouOperation   int32 = -1037 // 还没轮到你操作
	Err_YouMustOperation  int32 = -1038 // 你必须操作
	Err_PaiXingError      int32 = -1039 // 牌型错误
	Err_PaiXingYaoBuQi    int32 = -1040 // 牌型不够大
	Err_OperationRepeat   int32 = -1041 // 重复操作
	Err_CheckFailed       int32 = -1042 // 验证失败
	Err_CheckMutex        int32 = -1043 // 存在 互斥人员
	Err_GPSNotOpen        int32 = -1044 // GPS未开启

	Err_ShouPaiCount    int32 = -1045 // 手牌数量不对
	Err_SeatScoreLittle int32 = -1046 // 座位分不够
	Err_LessDiFen       int32 = -1047 // 低于底分
	Err_FindHuPlayer    int32 = -1048 // 发现有人已经胡牌了
	Err_MaxTZ           int32 = -1049 // 最大同桌
)
