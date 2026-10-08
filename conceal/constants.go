package conceal

const (
	WireguardMsgInitiationType  = 1
	WireguardMsgResponseType    = 2
	WireguardMsgCookieReplyType = 3
	WireguardMsgTransportType   = 4

	WireguardMsgInitiationSize   = 148
	WireguardMsgResponseSize     = 92
	WireguardMsgCookieReplySize  = 64
	WireguardMsgTransportMinSize = 32
)
