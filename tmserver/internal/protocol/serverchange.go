package protocol

// MsgCNFServerChange is _MSG_DBCNFServerChange (Basedef.h:1369-1377):
//
//	42 | FLAG_DB2GAME | FLAG_GAME2CLIENT
//
// The dbServer builds it and the tmServer forwards it to the client verbatim
// (ProcessDBMessage.cpp:1471-1476), so the wire value keeps BOTH direction flags
// even though only the last hop is tmServer → client.
const MsgCNFServerChange Type = 42 | FlagDB2Game | FlagGame2Client

const (
	// serverChangeAccountLen is ACCOUNTNAME_LENGTH (Basedef.h:125).
	serverChangeAccountLen = 16
	// serverChangeEncLen is the legacy Enc[52] — the same width as the client's
	// AccountLogin.Zero[52], which is where it comes back on the next login.
	serverChangeEncLen = 52

	// MsgCNFServerChangeBodySize is AccountName[16] + Enc[52]; the full frame is
	// HeaderSize + this.
	MsgCNFServerChangeBodySize = serverChangeAccountLen + serverChangeEncLen
)

// EncodeCNFServerChangeBody builds the body of MSG_DBCNFServerChange: the account
// name the token belongs to and the enc/token string itself, both NUL-padded.
func EncodeCNFServerChangeBody(accountName, enc string) []byte {
	b := make([]byte, MsgCNFServerChangeBodySize)
	copy(b[:serverChangeAccountLen], accountName)
	copy(b[serverChangeAccountLen:], enc)
	return b
}
