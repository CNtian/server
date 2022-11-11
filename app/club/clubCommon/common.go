package clubCommon

import "vvService/commonPackge/mateProto"

type PostEvent interface {
	PostMaTeEvent(te *mateProto.MessageMaTe)
}
