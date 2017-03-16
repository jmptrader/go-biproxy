package common

type ChannelByteMessage struct {
	Data []byte
	Err  error
}

type ChannelInt64Message struct {
	Int int64
	Err error
}
