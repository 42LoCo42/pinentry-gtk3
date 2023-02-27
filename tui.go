package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func tui() string {
	// open gpg tty
	tty, err := os.Create(ttyPath)
	if err != nil {
		return err.Error()
	}

	fmt.Fprintln(tty, descriptionTxt)

	promptPW := func(prompt string) (string, error) {
		fmt.Fprint(tty, prompt, " ")
		password, err := term.ReadPassword(int(tty.Fd()))
		if err != nil {
			return "", err
		}
		fmt.Fprintln(tty)

		return string(password), nil
	}

	switch guiMode {
	case modeDefault:
		password, err := promptPW(prompt1Txt)
		if err != nil {
			return err.Error()
		}

		fmt.Println("D ", password)

	case modeRepeat:
		for {
			password1, err := promptPW(prompt1Txt)
			if err != nil {
				return err.Error()
			}

			password2, err := promptPW(prompt2Txt)
			if err != nil {
				return err.Error()
			}

			if password1 != password2 {
				fmt.Fprintln(tty, "Passwords don't match!")
				continue
			}

			fmt.Println("D ", password1)
			break
		}

	case modeConfirm:
		fmt.Fprintln(tty, "[y]es:", okBtnTxt)
		fmt.Fprintln(tty, "[n]o: ", cancelBtnTxt)

		resp, err := bufio.NewReader(tty).ReadString('\n')
		if err != nil {
			return err.Error()
		}
		if strings.TrimSpace(resp) != "y" {
			return "not confirmed"
		}

	case modeMessage:
		// do nothing
	}

	return ""
}
