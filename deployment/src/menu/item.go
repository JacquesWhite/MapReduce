package menu

type MenuItem struct {
	title, description string
}

func (i MenuItem) Title() string       { return i.title }
func (i MenuItem) Description() string { return i.description }
func (i MenuItem) FilterValue() string { return i.title }

func Item(title, description string) MenuItem {
	return MenuItem{title: title, description: description}
}
