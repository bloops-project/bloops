package main

import (
	"encoding/json"
	"fmt"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func main() {
	bundle := i18n.NewBundle(language.Russian)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	for _, lang := range []string{"en", "ru"} {
		bundle.MustLoadMessageFile(fmt.Sprintf("/home/robotomize/bloop/cmd/example/active.%v.json", lang))
	}
	//bundle := i18n.NewBundle(language.English)
	//bundle.AddMessages(language.English, &i18n.Message{
	//	ID:    "hello world",
	//	Other: "hello world",
	//})
	//bundle1 := i18n.NewBundle(language.Spanish)
	//bundle1.AddMessages(language.Spanish, &i18n.Message{
	//	ID:    "hello world",
	//	Other: "ваыа",
	//})
	//bundle2 := i18n.NewBundle(language.Russian)
	//bundle2.AddMessages(language.Russian, &i18n.Message{
	//	ID:    "hello world",
	//	Other: "Привет мир",
	//})

	//
	//bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	//bundle.LoadMessageFile("ru.json")

	loc := i18n.NewLocalizer(bundle, language.English.String())
	//
	fmt.Println(loc.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "hello world",
	}))
	loc1 := i18n.NewLocalizer(bundle, language.Russian.String())
	fmt.Println(loc1.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "hello world",
	}))
}
