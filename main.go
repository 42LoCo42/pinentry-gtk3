package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// #cgo pkg-config: gtk+-3.0
import "C"

const (
	modeDefault = iota
	modeRepeat  = iota
	modeConfirm = iota
	modeMessage = iota

	defaultDescriptionTxt = "Enter your password"
	defaultPrompt1Txt     = "Password:"
	defaultPrompt2Txt     = "Repeat:"
	defaultErrorTxt       = ""

	defaultOkBtnTxt     = "Ok"
	defaultCancelBtnTxt = "Cancel"

	errorDontMatch = "Passwords don't match"
	strengthBarMax = 100
)

var (
	descriptionTxt string
	prompt1Txt     string
	prompt2Txt     string
	errorTxt       string

	okBtnTxt     string
	cancelBtnTxt string

	guiMode int
	ttyPath string
)

func reset() {
	descriptionTxt = defaultDescriptionTxt
	prompt1Txt = defaultPrompt1Txt
	prompt2Txt = defaultPrompt2Txt
	errorTxt = defaultErrorTxt

	okBtnTxt = defaultOkBtnTxt
	cancelBtnTxt = defaultCancelBtnTxt

	guiMode = modeDefault
}

func decodeString(s string) string {
	result, err := url.QueryUnescape(s)
	if err != nil {
		log.Print(err)
		return s
	}

	return result
}

func main() {
	reset()

	runUI := gui

	if err := gtk.InitCheck(&os.Args); err != nil {
		log.Print(err)
		runUI = tui
	}

	fmt.Println("OK pinentry-hybrid accepting commands, use HELP to see all")

	in := bufio.NewReader(os.Stdin)
	running := true
	for running {
		error := ""

		line, err := in.ReadString('\n')
		if err != nil {
			log.Fatal(errors.Wrap(err, "Could not read command"))
		}

		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		cmd := parts[0]
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}

		switch cmd {
		case "NOP":
			// do nothing
		case "BYE":
			running = false
		case "RESET":
			reset()
		case "HELP":
			for _, cmd := range []string{
				"NOP", "BYE", "RESET", "HELP", "OPTION",
				"SETDESC", "SETPROMPT", "SETREPEAT", "SETERROR", "SETOK", "SETCANCEL",
				"GETPIN", "CONFIRM", "MESSAGE",
			} {
				fmt.Println("#", cmd)
			}
		case "OPTION":
			if prefix := "ttyname="; strings.HasPrefix(arg, prefix) {
				ttyPath = strings.TrimPrefix(arg, prefix)
			} else {
				error = "Unknown option " + arg
			}

		case "SETDESC":
			descriptionTxt = decodeString(arg)
		case "SETPROMPT":
			prompt1Txt = decodeString(arg)
		case "SETREPEAT":
			prompt2Txt = decodeString(arg)
			guiMode = modeRepeat
		case "SETERROR":
			errorTxt = decodeString(arg)
		case "SETOK":
			okBtnTxt = decodeString(arg)
		case "SETCANCEL":
			cancelBtnTxt = decodeString(arg)

		case "GETPIN":
			error = runUI()
		case "CONFIRM":
			guiMode = modeConfirm
			error = runUI()
			guiMode = modeDefault
		case "MESSAGE":
			guiMode = modeMessage
			error = runUI()
			guiMode = modeDefault

		default:
			error = fmt.Sprintf("Unknown command %s, use HELP", cmd)
		}

		if error == "" {
			fmt.Println("OK")
		} else {
			fmt.Println("ERR", error)
		}
	}
}
