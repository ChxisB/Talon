package model

import (
	"fmt"

	style "github.com/ChxisB/talon/deps/style/v2"
	"github.com/ChxisB/talon/internal/ui/chat"
	"github.com/ChxisB/talon/internal/ui/common"
)

// todosInfo renders the todos section for the sidebar, showing the list
// of todos with their status icons.
func (m *UI) todosInfo(width, maxItems int, isSection bool) string {
	if m.session == nil || len(m.session.Todos) == 0 {
		return ""
	}

	t := m.com.Styles

	completed := 0
	for _, todo := range m.session.Todos {
		if todo.Status == "completed" {
			completed++
		}
	}

	title := fmt.Sprintf("Todos (%d/%d)", completed, len(m.session.Todos))
	if isSection {
		title = common.Section(t, title, width)
	}

	// Use FormatTodosList to render the todo items.
	list := t.Files.EmptyMessage.Render("None")
	items := chat.FormatTodosList(t, m.session.Todos, "", width)
	if items != "" {
		list = items
	}

	return style.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
}
