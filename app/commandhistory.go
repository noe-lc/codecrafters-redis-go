package main

type CommandHistoryItem struct {
	Command string
	Args    []string
	Success bool
	Acks    int
}

func (c CommandHistoryItem) GetType() string {
	return RespCommands[c.Command].Type
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
