package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type FnNames string

const (
	onAdd         FnNames = "Add"
	onPlayerState FnNames = "PlayerState"
)

var CallbackNames = []FnNames{
	onAdd,
	onPlayerState,
}

type OnAdd func(a int, b int) int

type OnPlayerState func(state G15PlayerState) G15PlayerState

var (
	errInvalidScriptInterpreter = errors.New("invalid interpreter")
	errInvalidScriptDir         = errors.New("invalid script directory")
	errInvalidScriptFile        = errors.New("invalid script file")
	errInvalidScriptNamespace   = errors.New("invalid script namespace")
)

type Scripting struct {
	interpreter   *interp.Interpreter
	importedFnMap map[string]reflect.Value
	onAdd         []OnAdd
	onPlayerState []OnPlayerState
}

func NewScripting() (*Scripting, error) {
	interpreter := interp.New(interp.Options{Unrestricted: true})
	if err := interpreter.Use(stdlib.Symbols); err != nil {
		return nil, errors.Join(err, errInvalidScriptInterpreter)
	}

	custom := make(map[string]map[string]reflect.Value)
	custom["tftui/tftui"] = make(map[string]reflect.Value)
	custom["tftui/tftui"]["PlayerState"] = reflect.ValueOf((*G15PlayerState)(nil))
	custom["tftui/tftui"]["g15PlayerCount"] = reflect.ValueOf(g15PlayerCount)

	if err := interpreter.Use(custom); err != nil {
		return nil, errors.Join(err, errInvalidScriptNamespace)
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
			return errors.Join(err, errInvalidScriptFile)
		}

		for _, name := range CallbackNames {
			evaluatedFunc, errEval := s.interpreter.Eval(fmt.Sprintf("%s.%s", scriptMeta.pkg, name))
			if errEval != nil {
				continue
			}

			switch name {
			case onAdd:
				call, ok := evaluatedFunc.Interface().(func(int, int) int)
				if !ok {
					continue
				}

				s.onAdd = append(s.onAdd, call)
			case onPlayerState:
				call, ok := evaluatedFunc.Interface().(func(G15PlayerState) G15PlayerState)
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
		return nil, errors.Join(err, errInvalidScriptDir)
	}

	var scripts []scriptEntry
	for _, dir := range dirList {
		if !dir.IsDir() {
			continue
		}

		fileList, errFiles := os.ReadDir(path.Join(rootPath, dir.Name()))
		if errFiles != nil {
			return nil, errors.Join(errFiles, errInvalidScriptDir)
		}

		for _, entry := range fileList {
			if entry.IsDir() {
				continue
			}

			if !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}

			scripts = append(scripts, scriptEntry{pkg: dir.Name(), filename: entry.Name()})
		}
	}

	return scripts, nil
}
