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
	zone "github.com/lrstanley/bubblezone"
)

func NewPluginList(title string) list.Model {
	newList := list.New(nil, PluginDelegate[GamePluginItem[tf.GamePlugin]]{}, 2, 2)
	newList.SetStatusBarItemName("plugin", "plugins")
	setListDefaults(&newList, title)

	return newList
}

func NewCVarList() list.Model {
	newList := list.New(nil, CVarDelegate[CVarItem[tf.CVar]]{}, 2, 2)
	newList.SetStatusBarItemName("cvar", "cvars")
	setListDefaults(&newList, "Game Config")

	return newList
}

func setListDefaults(newList *list.Model, title string) {
	newList.Title = title
	newList.DisableQuitKeybindings()
	newList.SetShowStatusBar(false)
	newList.SetShowHelp(false)
	newList.SetShowTitle(false)
	newList.Styles.Title = styles.PluginTitle
	newList.Styles.TitleBar = lipgloss.NewStyle().Padding(0).Align(lipgloss.Center)
}

type GamePluginItem[T any] struct {
	Item T
}

func (i GamePluginItem[T]) FilterValue() string { return "" }

type PluginDelegate[T any] struct{}

func (d PluginDelegate[T]) Height() int                             { return 1 }
func (d PluginDelegate[T]) Spacing() int                            { return 0 }
func (d PluginDelegate[T]) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d PluginDelegate[T]) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(GamePluginItem[tf.GamePlugin])
	if !ok {
		return
	}

	str := fmt.Sprintf("#%d: %s ", i.Item.Index, i.Item.Name)
	var err error
	render := styles.PluginItem.Inline(true).Padding(0).Render
	if index == m.Index() {
		_, err = fmt.Fprint(w, render(styles.ListSelectedRow.Render(str)))
	} else {
		_, err = fmt.Fprint(w, render(styles.ListUnelectedRow.Render(str)))
	}

	if err != nil {
		slog.Error("Failed to render item delegate", slog.String("error", err.Error()))
	}
}

type CVarItem[T any] struct {
	Item T
}

func (i CVarItem[T]) FilterValue() string { return "" }

type CVarDelegate[T any] struct{}

func (d CVarDelegate[T]) Height() int                             { return 1 }
func (d CVarDelegate[T]) Spacing() int                            { return 0 }
func (d CVarDelegate[T]) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d CVarDelegate[T]) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	cvar, ok := listItem.(CVarItem[tf.CVar])
	if !ok {
		return
	}

	var (
		line    = fmt.Sprintf("%s: %s ", cvar.Item.Name, cvar.Item.Value)
		content string
	)

	if index == m.Index() {
		content = styles.ListSelectedRow.Render(line)
	} else {
		content = styles.ListUnelectedRow.Render(line)
	}

	if _, err := fmt.Fprint(w, zone.Mark("", content)); err != nil {
		slog.Error("Failed to render item delegate", slog.String("error", err.Error()))
	}
}
