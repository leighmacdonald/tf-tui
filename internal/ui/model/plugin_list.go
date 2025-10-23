package model

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func NewPluginList(title string) list.Model {
	newList := list.New(nil, PluginDelegate[PluginItem[tf.GamePlugin]]{}, 2, 2)
	newList.Title = title
	newList.DisableQuitKeybindings()
	newList.SetShowStatusBar(false)
	newList.SetShowHelp(false)
	newList.Styles.Title = styles.PluginTitle
	newList.Styles.TitleBar = lipgloss.NewStyle().Padding(0).Align(lipgloss.Center)
	newList.SetStatusBarItemName("plugin", "plugins")

	return newList
}

type PluginItem[T any] struct {
	Item T
}

func (i PluginItem[T]) FilterValue() string { return "" }

type PluginDelegate[T any] struct{}

func (d PluginDelegate[T]) Height() int                             { return 1 }
func (d PluginDelegate[T]) Spacing() int                            { return 0 }
func (d PluginDelegate[T]) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d PluginDelegate[T]) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(PluginItem[tf.GamePlugin])
	if !ok {
		return
	}

	str := fmt.Sprintf("#%d: %s ", i.Item.Index, i.Item.Name)
	var err error
	render := styles.PluginItem.Render
	if index == m.Index() {
		_, err = fmt.Fprint(w, render(styles.SelectedCellStyleRed.Render(str)))
	} else {
		_, err = fmt.Fprint(w, render(str))
	}

	if err != nil {
		slog.Error("Failed to render item delegate", slog.String("error", err.Error()))
	}
}
