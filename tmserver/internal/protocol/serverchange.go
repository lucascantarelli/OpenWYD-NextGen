package protocol

// MsgCNFServerChange é _MSG_DBCNFServerChange (Basedef.h:1369-1377):
//
//	42 | FLAG_DB2GAME | FLAG_GAME2CLIENT
//
// O dbServer o monta e o tmServer o repassa ao cliente tal e qual
// (ProcessDBMessage.cpp:1471-1476), então o valor de wire mantém AS DUAS flags
// de direção, mesmo o último salto sendo tmServer → cliente.
const MsgCNFServerChange Type = 42 | FlagDB2Game | FlagGame2Client

const (
	// serverChangeAccountLen é ACCOUNTNAME_LENGTH (Basedef.h:125).
	serverChangeAccountLen = 16
	// serverChangeEncLen é o Enc[52] legado — a mesma largura do
	// AccountLogin.Zero[52] do cliente, que é onde ele volta no próximo login.
	serverChangeEncLen = 52

	// MsgCNFServerChangeBodySize é AccountName[16] + Enc[52]; o frame completo é
	// HeaderSize + isto.
	MsgCNFServerChangeBodySize = serverChangeAccountLen + serverChangeEncLen
)

// EncodeCNFServerChangeBody monta o corpo de MSG_DBCNFServerChange: o nome da
// conta dona do token e a própria string enc/token, ambas com padding de NUL.
func EncodeCNFServerChangeBody(accountName, enc string) []byte {
	b := make([]byte, MsgCNFServerChangeBodySize)
	copy(b[:serverChangeAccountLen], accountName)
	copy(b[serverChangeAccountLen:], enc)
	return b
}
