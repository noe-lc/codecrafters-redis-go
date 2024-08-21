package main

type CommandHistoryItem struct {
	command string
	args    []string
	success bool
	acks    int
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
