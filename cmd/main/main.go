package main

import (
	"bytes"
	"errors"
	"fmt"
	"local-password-manager/internal/db"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/ProtonMail/gopenpgp/v2/helper"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	*sync.Mutex
	Name      string
	Username  string
	Password  []byte
	Recipient string
}

type PubKey struct {
	gorm.Model
	*sync.Mutex
	Recipient string
	Key       []byte
}

var pg *gorm.DB

/*
 * Program's entry point.
 */
func main() {

	dependencyCheck()

	// Verfying Card
	validateYK()

	clean()

	// Setting up TUI
	//
	app := tview.NewApplication()
	setKillKey(app) // used to terminate app

	// Main App
	list := mainMenu(app)

	go tui(app, nil, nil, list) // Starting tui go routine

	for {
		// Program will stay active until user terminates it with CTL-C or DEL key in
		// Some cases
	}

}

/*
 * Validate if GPG is installed.
 * TODO Find a more efficient way to check for packages
 */
func dependencyCheck() {

	cmd := exec.Command("/bin/bash", "-c", "gpg --version")

	var out bytes.Buffer

	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

/*
 * Validates Yubikey. It will ask user for password to unlock
 * the key.
 */
func validateYK() {

	fmt.Println("Please insert Yubikey and press Enter...")
	fmt.Scanln()

	cmd := exec.Command("/bin/bash", "-c", "gpg-card verify")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

/*
 * Removes artifacts from host.
 */
func clean() {
	lpwe := "/var/tmp/lpwe"
	os.Remove(lpwe)
}

/*
 * Function used to set the keystroke to terminate tview applications.
 */
func setKillKey(app *tview.Application) {

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		if event.Rune() == rune(tcell.KeyCtrlC) {
			clean()
			app.Stop()
			os.Exit(0)
		}

		return event
	})
}

/*
 * Displays the Main Menu. Validate that the database is Connected
 * before allowing certain options
 */
func mainMenu(app *tview.Application) *tview.List {

	list := tview.NewList()

	list.SetBorder(true)
	list.SetTitle("Password Manager")
	list.SetTitleAlign(tview.AlignLeft)

	list.AddItem("Connect to Database", "", 'a', func() {

		app.Suspend(func() {

			app := tview.NewApplication()
			setKillKey(app)
			form := pgUserInput(app)
			tui(app, form, nil, nil)
		})

	})

	list.AddItem("Add Pubkey", "", 'c', func() {
		if pg != nil {

			app.Suspend(func() {
				app := tview.NewApplication()
				setKillKey(app)
				form := addPubKeyMenu(app)
				tui(app, form, nil, nil)

			})
		} else {
			app.Suspend(func() {
				displayText("Database Not Connected.\n" +
					"Press Enter to continue.")
			})

		}

	})

	list.AddItem("Add Account", "", 'b', func() {
		if pg != nil {

			app.Suspend(func() {
				app := tview.NewApplication()
				setKillKey(app)
				form, err := addAccountMenu(app)
				if err != nil {
					return
				}
				tui(app, form, nil, nil)

			})
		} else {
			app.Suspend(func() {
				displayText("Database Not Connected.\n" +
					"Press Enter to continue.")
			})

		}

	})

	list.AddItem("Retrieve Account", "", 'd', func() {

		if pg != nil {

			app.Suspend(func() {
				app := tview.NewApplication()
				setKillKey(app)
				form, err := getAccountMenu(app)
				if err != nil {
					return
				}
				tui(app, form, nil, nil)

			})
		} else {
			app.Suspend(func() {
				displayText("Database Not Connected.\n" +
					"Press Enter to continue.")
			})
		}

	})

	list.AddItem("Delete Account", "", 'e', func() {

		if pg != nil {

			app.Suspend(func() {
				app := tview.NewApplication()
				setKillKey(app)
				form, err := deleteAccountMenu(app)
				if err != nil {
					return
				}
				tui(app, form, nil, nil)

			})
		} else {
			app.Suspend(func() {
				displayText("Database Not Connected.\n" +
					"Press Enter to continue.")
			})
		}

	})
	list.AddItem("Exit", "", 'q', func() {
		app.Suspend(func() {
			os.Remove("./temp")
			os.Exit(0)
		})
	})

	return list
}

/*
 * Displays the database connect menu
 */
func pgUserInput(app *tview.Application) *tview.Form {

	// Creating Form to Connect to PG
	//
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Password manager").
		SetTitleAlign(tview.AlignLeft)

	form.MouseHandler()
	form.AddInputField("User", "postgres", 20, nil, nil)
	form.AddPasswordField("Password", "password", 20, 0, nil)
	form.AddInputField("Host", "localhost", 20, nil, nil)
	form.AddInputField("Name", "postgres", 20, nil, nil)
	form.AddInputField("Port", "5432", 20, nil, nil)
	form.AddDropDown("SSL Enabled:", []string{"disable", "require", "verify-ca", "verify-all"}, 0, nil)

	form.AddButton("Connect", func() {

		_ = dbConnect(form, app)
		app.Stop()

	})

	form.AddButton("Back", func() { app.Stop() })

	return form

}

/*
 * Connects the program to the database
 */
func dbConnect(form *tview.Form, app *tview.Application) error {

	_, res := form.GetFormItem(5).(*tview.DropDown).GetCurrentOption()

	//Extracting config from tui
	conf := db.Config{
		User:     form.GetFormItem(0).(*tview.InputField).GetText(),
		Password: form.GetFormItem(1).(*tview.InputField).GetText(),
		Host:     form.GetFormItem(2).(*tview.InputField).GetText(),
		Name:     form.GetFormItem(3).(*tview.InputField).GetText(),
		Port:     form.GetFormItem(4).(*tview.InputField).GetText(),
		Ssl:      res,
	}

	var err error
	pg, err = db.Connect(&conf) // Connecting to DB
	if err != nil {

		// Suspend form tui
		app.Suspend(func() {

			displayText("Failed to connect to database.\n" +
				"Press Enter to continue.")
		})

		return err
	}

	pg.AutoMigrate(&Account{}, &PubKey{})

	return nil
}

/*
 * Displays account request menu.
 */
func getAccountMenu(app *tview.Application) (*tview.Form, error) {

	// Creating Form to get records from PG
	//
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Password manager").
		SetTitleAlign(tview.AlignLeft)

	accounts := []Account{}

	gormErr := pg.Find(&accounts).Error

	if gormErr != nil {
		displayText("Connection Error:\n" + gormErr.Error())
		return form, gormErr
	}

	if len(accounts) == 0 {
		displayText("Database Empty.\n" +
			"Press Enter to Continue.")
		return form, errors.New("Database Empty")
	}

	var options []string

	for _, account := range accounts {
		options = append(options, account.Name)
	}

	form.AddDropDown("Accounts", options, 0, nil)

	form.AddButton("Add", func() {
		getAccount(form, app, accounts)
		app.Stop()
	})

	form.AddButton("Back", func() { app.Stop() })

	return form, nil

}

/*
 * Retrieves account information from database
 */
func getAccount(form *tview.Form, app *tview.Application, accounts []Account) {

	lpwe := "/var/tmp/lpwe"

	idx, _ := form.GetFormItem(0).(*tview.DropDown).GetCurrentOption()

	os.WriteFile(lpwe, accounts[idx].Password, 0644)

	cmd := exec.Command("/bin/bash", "-c", "gpg --decrypt --quiet --batch --yes --armor --recipient "+
		accounts[idx].Recipient+" "+lpwe)

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	app.Suspend(func() {
		displayText("Password: " + out.String())
	})

	os.Remove(lpwe)

	return

}

/*
 * Displays the menu to add accounts to database
 */
func addAccountMenu(app *tview.Application) (*tview.Form, error) {

	// Creating Form
	//
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Password manager").
		SetTitleAlign(tview.AlignLeft)

	pubKeys := []PubKey{}

	gormErr := pg.Find(&pubKeys).Error

	if gormErr != nil {
		displayText("Connection Error:\n" + gormErr.Error())
		return form, gormErr
	}

	if len(pubKeys) == 0 {
		displayText("Database Empty.\n" +
			"Press Enter to Continue.")
		return form, errors.New("Database Empty")
	}

	var options []string

	for _, keys := range pubKeys {
		options = append(options, keys.Recipient)
	}

	form.MouseHandler()
	form.AddInputField("Account Name", "", 30, nil, nil)
	form.AddInputField("Username", "", 30, nil, nil)
	form.AddPasswordField("Password", "", 30, 0, nil)
	form.AddDropDown("Recipient", options, 0, nil)

	form.AddButton("Add", func() {
		addAccount(form, app, pubKeys)
		app.Stop()
	})

	form.AddButton("Back", func() { app.Stop() })

	return form, nil

}

/*
 * Adds account with encrypted password to the database.
 */
func addAccount(form *tview.Form, app *tview.Application, pubkeys []PubKey) {

	idx, _ := form.GetFormItem(3).(*tview.DropDown).GetCurrentOption()
	pubkey := string(pubkeys[idx].Key)
	passwd := form.GetFormItem(2).(*tview.InputField).GetText()

	if pubkey == "" || passwd == "" {
		app.Suspend(func() {
			displayText("ERROR!!! Ensure all blocks are filled out.")

		})
		return
	}

	armor, err := helper.EncryptMessageArmored(pubkey, passwd)
	if err != nil {
		app.Suspend(func() {
			displayText(err.Error())
		})
		return
	}

	account := Account{
		Name:      form.GetFormItem(0).(*tview.InputField).GetText(),
		Username:  form.GetFormItem(1).(*tview.InputField).GetText(),
		Password:  []byte(armor),
		Recipient: pubkeys[idx].Recipient,
	}

	if account.Name == "" || account.Recipient == "" || account.Username == "" {
		app.Suspend(func() {
			displayText("ERROR!!! Ensure all blocks are filled out.")
		})
		return
	}

	gormErr := pg.Create(&account).Error
	if gormErr != nil {

		// Suspend form tui
		app.Suspend(func() {
			displayText("Connection Error:\n" + gormErr.Error())
		})
	}
	return

}

/*
 * Displays menu to add public key to database. It is mandatory to have
 * at least one public key available on the database to be able to add
 * accounts.
 */
func addPubKeyMenu(app *tview.Application) *tview.Form {

	// Creating Form to add public key to server
	//
	form := tview.NewForm()

	form.SetBorder(true).SetTitle("Password manager").
		SetTitleAlign(tview.AlignLeft)

	form.AddInputField("Recipient", "", 30, nil, nil)
	form.AddTextArea("Public Key", "", 80, 20, 0, nil)

	form.AddButton("Add", func() {
		addPubKey(form, app)
		app.Stop()
	})

	form.AddButton("Back", func() { app.Stop() })

	return form

}

/*
 * Adds public key to Database.
 */
func addPubKey(form *tview.Form, app *tview.Application) {

	recipient := form.GetFormItem(0).(*tview.InputField).GetText()
	pubKey := form.GetFormItem(1).(*tview.TextArea).GetText()

	if pubKey == "" || recipient == "" {
		app.Suspend(func() {
			displayText("ERROR!!! Ensure all blocks are filled out.")

		})
		return
	}

	pk := PubKey{Recipient: recipient, Key: []byte(pubKey)}

	gormErr := pg.Create(&pk).Error
	if gormErr != nil {

		// Suspend form tui
		app.Suspend(func() {
			displayText("Connection Error:\n" + gormErr.Error())
		})
	}
	return

}

/*
 * Displays account request menu.
 */
func deleteAccountMenu(app *tview.Application) (*tview.Form, error) {

	// Creating Form to get records from PG
	//
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Password manager").
		SetTitleAlign(tview.AlignLeft)

	accounts := []Account{}

	gormErr := pg.Find(&accounts).Error

	if gormErr != nil {
		displayText("Connection Error:\n" + gormErr.Error())
		return form, gormErr
	}

	if len(accounts) == 0 {
		displayText("Database Empty.\n" +
			"Press Enter to Continue.")
		return form, errors.New("Database Empty")
	}

	var options []string

	for _, account := range accounts {
		options = append(options, account.Name)
	}

	form.AddDropDown("Accounts", options, 0, nil)

	form.AddButton("Delete", func() {
		deleteAccount(form, app, accounts)
		app.Stop()
	})

	form.AddButton("Back", func() { app.Stop() })

	return form, nil

}

/*
 * Retrieves account information from database
 */
func deleteAccount(form *tview.Form, app *tview.Application, accounts []Account) {

	idx, _ := form.GetFormItem(0).(*tview.DropDown).GetCurrentOption()

	err := pg.Delete(&Account{}, "name", accounts[idx].Name).Error

	if err != nil {
		log.Fatal(err)
	}

	app.Suspend(func() {
		displayText("Account Deleted!")
	})

	return

}

/*
 * Used to display strings to tui
 */
func displayText(msg string) {

	// creating new application to display error msg
	app := tview.NewApplication()
	var text = tview.NewTextView()

	text.SetBorder(true).SetTitle("Password Manager").SetTitleAlign(tview.AlignLeft)

	text.SetText(msg)

	text.SetDoneFunc(func(key tcell.Key) { app.Stop() }) // Supports several keystrokes
	tui(app, nil, text, nil)

}

/*
 * Starts the tui for specific tview (Application, Form, or TextView)
 */
func tui(app *tview.Application, form *tview.Form, text *tview.TextView, list *tview.List) {

	if text != nil {

		err := app.SetRoot(text, true).SetFocus(text).Run()
		if err != nil {
			log.Fatal(err.Error())
		}
	} else if form != nil {

		err := app.SetRoot(form, true).SetFocus(form).Run()
		if err != nil {
			log.Fatal(err.Error())
		}

	} else if list != nil {
		err := app.SetRoot(list, true).SetFocus(list).Run()
		if err != nil {
			log.Fatal(err.Error())
		}

	}
}
