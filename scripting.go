package main

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/leighmacdonald/tf-tui/shared"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type FnNames string

const (
	Add         FnNames = "Add"
	PlayerState FnNames = "PlayerState"
)

var (
	CallbackNames = []FnNames{
		Add,
		PlayerState,
	}
)

type OnAdd func(a int, b int) int

type OnPlayerState func(state shared.PlayerState) shared.PlayerState

type Scripting struct {
	interpreter   *interp.Interpreter
	importedFnMap map[string]reflect.Value
	onAdd         []OnAdd
	onPlayerState []OnPlayerState
}

func NewScripting() (*Scripting, error) {
	interpreter := interp.New(interp.Options{Unrestricted: true})
	if err := interpreter.Use(stdlib.Symbols); err != nil {
		return nil, err
	}

	custom := make(map[string]map[string]reflect.Value)
	custom["tftui/tftui"] = make(map[string]reflect.Value)
	custom["tftui/tftui"]["PlayerState"] = reflect.ValueOf((*shared.PlayerState)(nil))
	custom["tftui/tftui"]["MaxDataSize"] = reflect.ValueOf(shared.MaxDataSize)

	if err := interpreter.Use(custom); err != nil {
		return nil, err
	}

	return &Scripting{interpreter: interpreter}, nil
}

func (s *Scripting) LoadDir(scriptDir string) error {
	scripts, errScripts := findScripts(scriptDir)
	if errScripts != nil {
		return errScripts
	}

	for _, scriptMeta := range scripts {
		_, err := s.interpreter.EvalPath(path.Join(scriptDir, scriptMeta.pkg, scriptMeta.filename))
		if err != nil {
			return err
		}

		for _, name := range CallbackNames {
			fn, errEval := s.interpreter.Eval(fmt.Sprintf("%s.%s", scriptMeta.pkg, name))
			if errEval != nil {
				continue
			}

			switch name {
			case Add:
				call, ok := fn.Interface().(func(int, int) int)
				if !ok {
					continue
				}

				s.onAdd = append(s.onAdd, call)
			case PlayerState:
				call, ok := fn.Interface().(func(shared.PlayerState) shared.PlayerState)
				if !ok {
					continue
				}

				s.onPlayerState = append(s.onPlayerState, call)
			}
		}
	}

	return nil
}

type scriptEntry struct {
	pkg      string
	filename string
}

func findScripts(rootPath string) ([]scriptEntry, error) {
	dirList, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	var scripts []scriptEntry
	for _, dir := range dirList {
		if !dir.IsDir() {
			continue
		}

		fileList, errFiles := os.ReadDir(path.Join(rootPath, dir.Name()))
		if errFiles != nil {
			return nil, errFiles
		}

		for _, e := range fileList {
			if e.IsDir() {
				continue
			}

			if !strings.HasSuffix(e.Name(), ".go") {
				continue
			}

			scripts = append(scripts, scriptEntry{pkg: dir.Name(), filename: e.Name()})
		}

	}

	return scripts, nil
}
