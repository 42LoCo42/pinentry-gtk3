package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/nbutton23/zxcvbn-go"
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

	fmt.Println("OK pinentry-hybrid accepting commands, use HELP to see all")
	gtk.Init(&os.Args)

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
				"NOP", "BYE", "RESET", "HELP",
				"SETDESC", "SETPROMPT", "SETREPEAT", "SETOK", "SETCANCEL",
				"GETPIN", "CONFIRM", "MESSAGE",
			} {
				fmt.Println("#", cmd)
			}

		case "SETDESC":
			descriptionTxt = decodeString(arg)
		case "SETPROMPT":
			prompt1Txt = decodeString(arg)
		case "SETREPEAT":
			prompt2Txt = decodeString(arg)
			guiMode = modeRepeat
		case "SETOK":
			okBtnTxt = decodeString(arg)
		case "SETCANCEL":
			cancelBtnTxt = decodeString(arg)

		case "GETPIN":
			error = gui()
		case "CONFIRM":
			guiMode = modeConfirm
			error = gui()
			guiMode = modeDefault
		case "MESSAGE":
			guiMode = modeMessage
			error = gui()
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

func gui() string {
	builder, err := gtk.BuilderNew()
	if err != nil {
		log.Fatal(err)
	}

	// if err := builder.AddFromFile("foo.glade"); err != nil {
	if err := builder.AddFromResource("/org/gtk/pinentry-hybrid/gui.glade"); err != nil {
		log.Fatal(errors.Wrap(err, "Could not load UI file"))
	}

	var (
		window = GetObject[*gtk.Window](builder, "window")

		description = GetObject[*gtk.Label](builder, "description")
		prompt1     = GetObject[*gtk.Label](builder, "prompt1")
		prompt2     = GetObject[*gtk.Label](builder, "prompt2")
		error       = GetObject[*gtk.Label](builder, "error")

		password1 = GetObject[*gtk.Entry](builder, "password1")
		password2 = GetObject[*gtk.Entry](builder, "password2")

		entropy  = GetObject[*gtk.Label](builder, "entropy")
		strength = GetObject[*gtk.ProgressBar](builder, "strength")

		btnOk     = GetObject[*gtk.Button](builder, "btnOk")
		btnCancel = GetObject[*gtk.Button](builder, "btnCancel")

		passwordOk = true
	)

	updateAllTexts := func() {
		description.SetText(descriptionTxt)
		prompt1.SetText(prompt1Txt)
		prompt2.SetText(prompt2Txt)
		error.SetText(errorTxt)

		btnOk.SetLabel(okBtnTxt)
		btnCancel.SetLabel(cancelBtnTxt)
	}

	updateAllTexts()

	enableCurrentGuiMode := func() {
		// defaults

		prompt1.SetVisible(true)
		prompt2.SetVisible(false)

		password1.SetVisible(true)
		password2.SetVisible(false)
		error.SetVisible(false)

		entropy.SetVisible(true)
		strength.SetVisible(true)

		btnOk.SetVisible(true)
		btnCancel.SetVisible(true)

		setConfirm := func() {
			prompt1.SetVisible(false)
			password1.SetVisible(false)
			entropy.SetVisible(false)
			strength.SetVisible(false)
		}

		// special modes
		switch guiMode {
		case modeRepeat:
			prompt2.SetVisible(true)
			password2.SetVisible(true)

		case modeConfirm:
			setConfirm()
			btnCancel.GrabFocus()

		case modeMessage:
			setConfirm()
			btnOk.GrabFocus()
			btnCancel.SetVisible(false)
		}
	}

	enableCurrentGuiMode()

	getPassword := func(entry *gtk.Entry) string {
		password, err := entry.GetText()
		if err != nil {
			log.Fatal(errors.Wrap(err, "Could not get password from entry"))
		}

		return password
	}

	var ok *bool = nil

	quit := func(b bool) {
		if ok == nil {
			ok = &b
		}

		window.Destroy()
		gtk.MainQuit()
	}

	onOk := func() {
		if !passwordOk {
			return
		}

		if guiMode == modeDefault || guiMode == modeRepeat {
			fmt.Println("D", getPassword(password1))
		}

		quit(true)
	}

	onCancel := func() {
		quit(false)
	}

	onPasswordUpdate := func() {
		pw1, err := password1.GetText()
		if err != nil {
			log.Fatal(errors.Wrap(err, "Could not get password 1"))
		}

		pwStrength := zxcvbn.PasswordStrength(pw1, nil)
		entropy.SetText(fmt.Sprintf(
			"Entropy: %d bits",
			int(math.Round(pwStrength.Entropy))),
		)
		strength.SetFraction(pwStrength.Entropy / strengthBarMax)

		if guiMode == modeDefault {
			passwordOk = true
			return
		}

		pw2, err := password2.GetText()
		if err != nil {
			log.Fatal(errors.Wrap(err, "Could not get password 2"))
		}

		passwordOk = pw1 == pw2
		error.SetVisible(!passwordOk)
		if !passwordOk {
			errorTxt = errorDontMatch
		}

		updateAllTexts()
	}

	ConnectMap{
		window: {
			"destroy": onCancel,
		},

		btnOk: {
			"clicked": onOk,
		},

		btnCancel: {
			"clicked": onCancel,
		},

		password1: {
			"activate": func() {
				if guiMode == modeRepeat {
					password2.GrabFocus()
				} else {
					onOk()
				}
			},
			"changed": onPasswordUpdate,
		},

		password2: {
			"activate": onOk,
			"changed":  onPasswordUpdate,
		},
	}.run()

	gtk.Main()

	if *ok {
		return ""
	} else {
		return "Operation cancelled"
	}
}

func GetObject[T any](builder *gtk.Builder, name string) T {
	raw, err := builder.GetObject(name)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "Could not get object %s", name))
	}

	object, ok := raw.(T)
	if !ok {
		log.Fatal(errors.Errorf("Conversion of %s to %T failed", name, *new(T)))
	}

	return object
}

type IConnect interface {
	Connect(signal string, target any) glib.SignalHandle
}

type ConnectMap map[IConnect]map[string]any

func (data ConnectMap) run() {
	for object, signals := range data {
		for signal, target := range signals {
			object.Connect(signal, target)
		}
	}
}
