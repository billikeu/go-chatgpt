package params

type Answer struct {
	MsgId      string // message id: chatgptuno id,
	ParentId   string
	Chunk      string
	Text       string
	Done       bool
	ChunkIndex int
}

// create params for ask callback
/*
callback(NewCallParams(msgId, parentId, chunk, text, false), err)
*/
func NewAnswer(msgId, parentId, chunk, text string, done bool, chunkIndex int) *Answer {
	p := &Answer{
		MsgId:      msgId,
		ParentId:   parentId,
		Chunk:      chunk,
		Text:       text,
		Done:       done,
		ChunkIndex: chunkIndex,
	}
	return p
}
