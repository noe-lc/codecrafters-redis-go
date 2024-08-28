package main

type CommandHistoryItem struct {
	RespCommand *RespCommand
	Args        []string
	Success     bool
	Acks        int
}

type CommandHistory []CommandHistoryItem

func (c *CommandHistory) Append(item CommandHistoryItem) {
	*c = append(*c, item)
}

func (c CommandHistory) GetEntry(index int) CommandHistoryItem {
	return c[index]
}

func (c CommandHistory) GetModifiableEntry(index int) *CommandHistoryItem {
	return &(c[index])
}
